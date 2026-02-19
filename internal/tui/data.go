package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	sdk "github.com/jcechace/pbmate/sdk/v2"
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
// encountered error; partial data is still returned.
func fetchOverviewCmd(client *sdk.Client) tea.Cmd {
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

		backups, err := client.Backups.List(ctx, sdk.ListBackupsOptions{Limit: 5})
		if err != nil && d.err == nil {
			d.err = err
		}
		d.recentBackups = backups

		ct, err := client.Cluster.ClusterTime(ctx)
		if err != nil && d.err == nil {
			d.err = err
		}
		d.clusterTime = ct

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
