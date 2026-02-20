package tui

import (
	"context"
	"fmt"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/sync/errgroup"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

const (
	logFetchCount      = 50 // number of recent log entries to fetch per poll
	recentBackupsLimit = 1  // number of recent backups to fetch for the overview
)

// connectMsg carries the result of the background SDK connection attempt.
type connectMsg struct {
	client *sdk.Client
	err    error
}

// connectCmd returns a tea.Cmd that connects to PBM in the background.
func connectCmd(uri string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		client, err := sdk.NewClient(ctx, sdk.WithMongoURI(uri))
		return connectMsg{client: client, err: err}
	}
}

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
type logFollowMsg struct {
	entries []sdk.LogEntry
	err     error
}

// logFollowDoneMsg signals that the follow channel has closed.
// err is set if the stream ended due to an error (e.g. connection lost).
type logFollowDoneMsg struct {
	err error
}

// waitForLogEntry returns a tea.Cmd that blocks until at least one entry
// arrives on the channel, then drains any additional buffered entries so
// they are delivered as a single batch (one Update / one re-render).
// When the entries channel closes, the error channel is checked for a
// follow-stream error to surface to the user.
func waitForLogEntry(entries <-chan sdk.LogEntry, errs <-chan error) tea.Cmd {
	return func() tea.Msg {
		// Block for the first entry.
		entry, ok := <-entries
		if !ok {
			// Entries channel closed — check for an error.
			return logFollowDoneMsg{err: drainErr(errs)}
		}
		batch := []sdk.LogEntry{entry}

		// Drain any additional entries that are already buffered.
		for {
			select {
			case e, ok := <-entries:
				if !ok {
					// Channel closed mid-drain; deliver what we have.
					return logFollowMsg{entries: batch}
				}
				batch = append(batch, e)
			default:
				return logFollowMsg{entries: batch}
			}
		}
	}
}

// drainErr reads one error from the channel without blocking.
// Returns nil if the channel is empty or closed.
func drainErr(errs <-chan error) error {
	if errs == nil {
		return nil
	}
	select {
	case err := <-errs:
		return err
	default:
		return nil
	}
}

// backupsData holds the result of a single backups poll cycle.
type backupsData struct {
	backups   []sdk.Backup
	timelines []sdk.Timeline
	err       error
}

// backupsDataMsg wraps backupsData as a BubbleTea message.
type backupsDataMsg struct{ backupsData }

// backupActionMsg carries the result of a backup action (start, cancel, delete).
type backupActionMsg struct {
	action string // "start", "cancel", "delete"
	err    error
}

// deleteConfirmMsg requests a delete confirmation overlay. The baseName is the
// backup that will actually be deleted (always the chain base for incremental).
// The title and description are displayed in the overlay.
type deleteConfirmMsg struct {
	baseName    string
	title       string
	description string
}

// requestDeleteConfirm returns a tea.Cmd that emits a deleteConfirmMsg.
func requestDeleteConfirm(baseName, title, description string) tea.Cmd {
	return func() tea.Msg {
		return deleteConfirmMsg{baseName: baseName, title: title, description: description}
	}
}

// backupFormReadyMsg carries fetched profiles so the backup form can be created.
type backupFormReadyMsg struct {
	profiles []sdk.StorageProfile
	kind     backupFormKind
}

// fetchProfilesCmd returns a tea.Cmd that fetches storage profiles for the
// backup form. Errors are silently ignored — the form will just show "Main".
func fetchProfilesCmd(client *sdk.Client, kind backupFormKind) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		profiles, _ := client.Config.ListProfiles(ctx)
		return backupFormReadyMsg{profiles: profiles, kind: kind}
	}
}

// startBackupWithOptsCmd returns a tea.Cmd that starts a backup with the given options.
func startBackupWithOptsCmd(client *sdk.Client, opts sdk.StartBackupOptions) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		_, err := client.Backups.Start(ctx, opts)
		return backupActionMsg{action: "start", err: err}
	}
}

// cancelBackupCmd returns a tea.Cmd that cancels the running backup.
func cancelBackupCmd(client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		_, err := client.Backups.Cancel(ctx)
		return backupActionMsg{action: "cancel", err: err}
	}
}

// deleteBackupCmd returns a tea.Cmd that deletes the named backup.
func deleteBackupCmd(client *sdk.Client, name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		_, err := client.Backups.Delete(ctx, name)
		return backupActionMsg{action: "delete", err: err}
	}
}

// fetchOverviewCmd returns a tea.Cmd that fetches all overview data from the
// SDK client concurrently. Errors from individual calls are coalesced into
// the first encountered error; partial data is still returned. When skipLogs
// is true, log fetching is skipped (e.g. during follow mode where logs stream
// separately).
func fetchOverviewCmd(client *sdk.Client, skipLogs bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var d overviewData

		// setErr records the first error encountered; goroutine-safe.
		var mu sync.Mutex
		setErr := func(err error) {
			if err != nil {
				mu.Lock()
				if d.err == nil {
					d.err = err
				}
				mu.Unlock()
			}
		}

		// All fetches are independent reads — run them concurrently.
		// Goroutines always return nil so errgroup never cancels early.
		g, gctx := errgroup.WithContext(ctx)

		g.Go(func() error {
			v, err := client.Cluster.Agents(gctx)
			d.agents = v
			setErr(err)
			return nil
		})

		g.Go(func() error {
			v, err := client.Cluster.RunningOperations(gctx)
			d.operations = v
			setErr(err)
			return nil
		})

		g.Go(func() error {
			v, err := client.PITR.Status(gctx)
			d.pitr = v
			setErr(err)
			return nil
		})

		g.Go(func() error {
			v, err := client.PITR.Timelines(gctx)
			d.timelines = v
			setErr(err)
			return nil
		})

		g.Go(func() error {
			v, err := client.Backups.List(gctx, sdk.ListBackupsOptions{Limit: recentBackupsLimit})
			d.recentBackups = v
			setErr(err)
			return nil
		})

		g.Go(func() error {
			v, err := client.Cluster.ClusterTime(gctx)
			d.clusterTime = v
			setErr(err)
			return nil
		})

		g.Go(func() error {
			cfg, err := client.Config.Get(gctx)
			setErr(err)
			if cfg != nil {
				d.storageName = formatStorageSummary(cfg.Storage)
			}
			return nil
		})

		if !skipLogs {
			g.Go(func() error {
				v, err := client.Logs.Get(gctx, sdk.GetLogsOptions{Limit: logFetchCount})
				d.logEntries = v
				setErr(err)
				return nil
			})
		}

		_ = g.Wait()
		return overviewDataMsg{d}
	}
}

// fetchBackupsCmd returns a tea.Cmd that fetches the backup list and PITR
// timelines concurrently.
func fetchBackupsCmd(client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var d backupsData

		var mu sync.Mutex
		setErr := func(err error) {
			if err != nil {
				mu.Lock()
				if d.err == nil {
					d.err = err
				}
				mu.Unlock()
			}
		}

		g, gctx := errgroup.WithContext(ctx)

		g.Go(func() error {
			v, err := client.Backups.List(gctx, sdk.ListBackupsOptions{})
			d.backups = v
			setErr(err)
			return nil
		})

		g.Go(func() error {
			v, err := client.PITR.Timelines(gctx)
			d.timelines = v
			setErr(err)
			return nil
		})

		_ = g.Wait()
		return backupsDataMsg{d}
	}
}

// restoresData holds the result of a single restores poll cycle.
type restoresData struct {
	restores []sdk.Restore
	err      error
}

// restoresDataMsg wraps restoresData as a BubbleTea message.
type restoresDataMsg struct{ restoresData }

// fetchRestoresCmd returns a tea.Cmd that fetches the full restore list.
func fetchRestoresCmd(client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		restores, err := client.Restores.List(ctx, sdk.ListRestoresOptions{})
		return restoresDataMsg{restoresData{restores: restores, err: err}}
	}
}

// formatStorageSummary returns a compact string describing the storage config.
func formatStorageSummary(s sdk.StorageConfig) string {
	if s.Type.IsZero() {
		return ""
	}
	if s.Path != "" {
		return fmt.Sprintf("%s %s", s.Type, s.Path)
	}
	return s.Type.String()
}
