package tui

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/sync/errgroup"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

const (
	logFetchCount      = 50 // number of recent log entries to fetch per poll
	recentBackupsLimit = 1  // number of recent backups to fetch for the overview
)

// overviewData holds the result of a single overview poll cycle.
type overviewData struct {
	agents        []sdk.Agent
	operations    []sdk.Operation
	pitr          *sdk.PITRStatus
	timelines     []sdk.Timeline
	recentBackups []sdk.Backup
	clusterTime   sdk.Timestamp
	storageName   string // main storage type + path summary
	logEntries    []sdk.LogEntry
	err           error
}

// overviewDataMsg wraps overviewData as a BubbleTea message.
type overviewDataMsg struct{ overviewData }

// logFollowMsg carries one or more log entries from the follow goroutine.
// Entries are batched: the cmd blocks for the first entry then drains all
// additionally buffered entries without blocking.
// The session field ties the message to a specific follow session so stale
// messages from a previous session are discarded.
type logFollowMsg struct {
	session uint64
	entries []sdk.LogEntry
	err     error
}

// logFollowDoneMsg signals that the follow channel has closed.
// err is set if the stream ended due to an error (e.g. connection lost).
type logFollowDoneMsg struct {
	session uint64
	err     error
}

// waitForLogEntry returns a tea.Cmd that blocks until at least one entry
// arrives on the channel or the context is cancelled. It drains any
// additional buffered entries so they are delivered as a single batch.
// When the entries channel closes, the error channel is checked for a
// follow-stream error to surface to the user.
func waitForLogEntry(ctx context.Context, session uint64, entries <-chan sdk.LogEntry, errs <-chan error) tea.Cmd {
	return func() tea.Msg {
		// Block for the first entry, with context cancellation escape.
		select {
		case <-ctx.Done():
			return logFollowDoneMsg{session: session, err: ctx.Err()}
		case entry, ok := <-entries:
			if !ok {
				// Entries channel closed — check for an error.
				return logFollowDoneMsg{session: session, err: drainErr(errs)}
			}

			batch := []sdk.LogEntry{entry}

			// Drain any additional entries that are already buffered.
			for {
				select {
				case e, ok := <-entries:
					if !ok {
						// Channel closed mid-drain; deliver what we have.
						return logFollowMsg{session: session, entries: batch}
					}
					batch = append(batch, e)
				default:
					return logFollowMsg{session: session, entries: batch}
				}
			}
		}
	}
}

// fetchOverviewCmd returns a tea.Cmd that fetches all overview data from the
// SDK client concurrently. Errors from individual calls are coalesced into
// the first encountered error; partial data is still returned. When skipLogs
// is true, log fetching is skipped (e.g. during follow mode where logs stream
// separately). The logFilter is applied to the log query — zero value defaults
// to Info severity with no other filters.
func fetchOverviewCmd(ctx context.Context, client *sdk.Client, skipLogs bool, logFilter sdk.LogFilter) tea.Cmd {
	return func() tea.Msg {
		var d overviewData
		var errs firstErrCollector

		// All fetches are independent reads — run them concurrently.
		// Goroutines always return nil so errgroup never cancels early.
		g, gctx := errgroup.WithContext(ctx)

		g.Go(func() error {
			v, err := client.Cluster.Agents(gctx)
			d.agents = v
			errs.set(err)
			return nil
		})

		g.Go(func() error {
			v, err := client.Cluster.RunningOperations(gctx)
			d.operations = v
			errs.set(err)
			return nil
		})

		g.Go(func() error {
			v, err := client.PITR.Status(gctx)
			d.pitr = v
			errs.set(err)
			return nil
		})

		g.Go(func() error {
			v, err := client.PITR.Timelines(gctx)
			d.timelines = v
			errs.set(err)
			return nil
		})

		g.Go(func() error {
			v, err := client.Backups.List(gctx, sdk.ListBackupsOptions{Limit: recentBackupsLimit})
			d.recentBackups = v
			errs.set(err)
			return nil
		})

		g.Go(func() error {
			v, err := client.Cluster.ClusterTime(gctx)
			d.clusterTime = v
			errs.set(err)
			return nil
		})

		g.Go(func() error {
			cfg, err := client.Config.Get(gctx)
			if err != nil && !errors.Is(err, sdk.ErrNotFound) {
				errs.set(err)
			}
			if cfg != nil {
				d.storageName = formatStorageSummary(cfg.Storage)
			}
			return nil
		})

		if !skipLogs {
			g.Go(func() error {
				v, err := client.Logs.Get(gctx, sdk.GetLogsOptions{
					LogFilter: logFilter,
					Limit:     logFetchCount,
				})
				d.logEntries = v
				errs.set(err)
				return nil
			})
		}

		_ = g.Wait()
		d.err = errs.err
		return overviewDataMsg{d}
	}
}
