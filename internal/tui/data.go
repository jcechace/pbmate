package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

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
type logFollowMsg struct {
	entries []sdk.LogEntry
	err     error
}

// logFollowDoneMsg signals that the follow channel has closed.
type logFollowDoneMsg struct{}

// waitForLogEntry returns a tea.Cmd that blocks until at least one entry
// arrives on the channel, then drains any additional buffered entries so
// they are delivered as a single batch (one Update / one re-render).
func waitForLogEntry(ch <-chan sdk.LogEntry) tea.Cmd {
	return func() tea.Msg {
		// Block for the first entry.
		entry, ok := <-ch
		if !ok {
			return logFollowDoneMsg{}
		}
		batch := []sdk.LogEntry{entry}

		// Drain any additional entries that are already buffered.
		for {
			select {
			case e, ok := <-ch:
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

// backupsData holds the result of a single backups poll cycle.
type backupsData struct {
	backups []sdk.Backup
	err     error
}

// backupsDataMsg wraps backupsData as a BubbleTea message.
type backupsDataMsg struct{ backupsData }

// backupActionMsg carries the result of a backup action (start, cancel, delete).
type backupActionMsg struct {
	action string // "start", "cancel", "delete"
	err    error
}

// startBackupCmd returns a tea.Cmd that starts a logical backup with defaults.
func startBackupCmd(client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		_, err := client.Backups.Start(ctx, sdk.StartBackupOptions{
			Type: sdk.BackupTypeLogical,
		})
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
// SDK client. Errors from individual calls are coalesced into the first
// encountered error; partial data is still returned. When skipLogs is true,
// log fetching is skipped (e.g. during follow mode where logs stream separately).
func fetchOverviewCmd(client *sdk.Client, skipLogs bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var d overviewData

		agents, err := client.Cluster.Agents(ctx)
		if err != nil && d.err == nil {
			d.err = err
		}
		d.agents = agents

		ops, err := client.Cluster.RunningOperations(ctx)
		if err != nil && d.err == nil {
			d.err = err
		}
		d.operations = ops

		pitr, err := client.PITR.Status(ctx)
		if err != nil && d.err == nil {
			d.err = err
		}
		d.pitr = pitr

		timelines, err := client.PITR.Timelines(ctx)
		if err != nil && d.err == nil {
			d.err = err
		}
		d.timelines = timelines

		backups, err := client.Backups.List(ctx, sdk.ListBackupsOptions{Limit: recentBackupsLimit})
		if err != nil && d.err == nil {
			d.err = err
		}
		d.recentBackups = backups

		ct, err := client.Cluster.ClusterTime(ctx)
		if err != nil && d.err == nil {
			d.err = err
		}
		d.clusterTime = ct

		// Fetch config for storage info.
		cfg, err := client.Config.Get(ctx)
		if err != nil && d.err == nil {
			d.err = err
		}
		if cfg != nil {
			d.storageName = formatStorageSummary(cfg.Storage)
		}

		// Fetch recent log entries (skip when follow mode is streaming them).
		if !skipLogs {
			logs, err := client.Logs.Get(ctx, logFetchCount)
			if err != nil && d.err == nil {
				d.err = err
			}
			d.logEntries = logs
		}

		return overviewDataMsg{d}
	}
}

// fetchBackupsCmd returns a tea.Cmd that fetches the full backup list.
func fetchBackupsCmd(client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		backups, err := client.Backups.List(ctx, sdk.ListBackupsOptions{})
		return backupsDataMsg{backupsData{backups: backups, err: err}}
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
