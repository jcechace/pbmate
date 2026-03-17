package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/sync/errgroup"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// backupsData holds the result of a single backups poll cycle.
type backupsData struct {
	backups   []sdk.Backup
	timelines []sdk.Timeline
	err       error
}

// backupsDataMsg wraps backupsData as a BubbleTea message.
type backupsDataMsg struct{ backupsData }

// restoresData holds the result of a single restores poll cycle.
type restoresData struct {
	restores []sdk.Restore
	err      error
}

// restoresDataMsg wraps restoresData as a BubbleTea message.
type restoresDataMsg struct{ restoresData }

// deleteCheckRequest is emitted by the backups sub-model when the user presses
// delete. The root model handles it by running a CanDelete pre-check.
type deleteCheckRequest struct {
	baseName    string
	title       string
	description string
}

// canDeleteMsg carries the result of a CanDelete pre-check. When err is nil,
// the confirm dialog should be shown. When err is non-nil, the error is
// displayed in the flash bar instead.
type canDeleteMsg struct {
	baseName    string
	title       string
	description string
	err         error
}

// backupFormReadyMsg carries fetched profiles and existing backups so the
// backup form can be created with chain-awareness for incremental backups.
type backupFormReadyMsg struct {
	profiles []sdk.StorageProfile
	backups  []sdk.Backup
	kind     backupFormKind
}

// restoreTargetRequest is emitted by the backups sub-model when the user
// presses R (generic restore). Opens Step 1 of the restore wizard with all
// available backups and timelines.
type restoreTargetRequest struct {
	backups   []sdk.Backup
	timelines []sdk.Timeline
}

// restoreRequest is emitted by the backups sub-model when the user presses
// r (restore selected). Skips Step 1 and goes directly to Step 2.
// The mode determines the form variant:
//   - snapshot: restore from the selected backup (backupName is set)
//   - pitr: restore to a point in time (timeline and backups are set)
type restoreRequest struct {
	mode      restoreMode
	backup    *sdk.Backup    // set for snapshot mode (full object for context display)
	timeline  *sdk.Timeline  // set for PITR mode
	backups   []sdk.Backup   // set for PITR mode (for base backup filtering)
	timelines []sdk.Timeline // set for PITR mode (for base backup filtering)
}

// physicalRestoreConfirmRequest is emitted when a restore targets a physical
// or incremental backup (including PITR with a physical/incremental base).
// Instead of dispatching immediately, the root model opens a final warning
// overlay because physical restores shut down mongod on all nodes.
type physicalRestoreConfirmRequest struct {
	cmd        sdk.StartRestoreCommand
	backupName string // base backup name (for display)
	backupType string // "physical" or "incremental" (for display)
	isPITR     bool   // true when this is a PITR restore with physical base
}

// physicalRestoreResultMsg carries the result of a physical restore dispatch.
// On success the TUI exits; on error the flash bar shows the error.
type physicalRestoreResultMsg struct {
	err error
}

// bulkDeleteRequest is emitted by the backups sub-model when the user
// opens the bulk delete form. The initial field pre-selects the target
// (e.g. PITR when pressing d on a timeline item).
type bulkDeleteRequest struct {
	initial *bulkDeleteFormResult // nil for default (backups), non-nil for pre-selection
}

// --- Backup commands ---

// canDeleteCmd returns a tea.Cmd that checks whether a backup can be deleted
// before showing the confirmation dialog.
func canDeleteCmd(ctx context.Context, client *sdk.Client, baseName, title, description string) tea.Cmd {
	return func() tea.Msg {
		err := client.Backups.CanDelete(ctx, baseName)
		return canDeleteMsg{
			baseName:    baseName,
			title:       title,
			description: description,
			err:         err,
		}
	}
}

// fetchBackupFormDataCmd returns a tea.Cmd that fetches storage profiles for
// the backup form. Errors are silently ignored — the form will just show "Main".
// backups is the already-fetched backup list (passed through, no extra call).
func fetchBackupFormDataCmd(ctx context.Context, client *sdk.Client, kind backupFormKind, backups []sdk.Backup) tea.Cmd {
	return func() tea.Msg {
		profiles, _ := client.Config.ListProfiles(ctx)
		return backupFormReadyMsg{profiles: profiles, backups: backups, kind: kind}
	}
}

// startBackupCmd returns a tea.Cmd that starts a backup with the given command.
func startBackupCmd(ctx context.Context, client *sdk.Client, cmd sdk.StartBackupCommand) tea.Cmd {
	return func() tea.Msg {
		_, err := client.Backups.Start(ctx, cmd)
		return actionResultMsg{action: "start", err: err}
	}
}

// cancelBackupCmd returns a tea.Cmd that cancels the running backup.
func cancelBackupCmd(ctx context.Context, client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		_, err := client.Backups.Cancel(ctx)
		return actionResultMsg{action: "cancel", err: err}
	}
}

// deleteBackupCmd returns a tea.Cmd that deletes the named backup.
func deleteBackupCmd(ctx context.Context, client *sdk.Client, name string) tea.Cmd {
	return func() tea.Msg {
		_, err := client.Backups.Delete(ctx, sdk.DeleteBackupByName{Name: name})
		return actionResultMsg{action: "delete", err: err}
	}
}

// deleteBackupsBulkCmd returns a tea.Cmd that deletes backups matching the
// given command (DeleteBackupsBefore or DeleteBackupsOlderThan).
func deleteBackupsBulkCmd(ctx context.Context, client *sdk.Client, cmd sdk.DeleteBackupCommand) tea.Cmd {
	return func() tea.Msg {
		_, err := client.Backups.Delete(ctx, cmd)
		return actionResultMsg{action: "bulk delete", err: err}
	}
}

// --- Restore commands ---

// startRestoreCmd returns a tea.Cmd that starts a restore with the given command.
func startRestoreCmd(ctx context.Context, client *sdk.Client, cmd sdk.StartRestoreCommand) tea.Cmd {
	return func() tea.Msg {
		_, err := client.Restores.Start(ctx, cmd)
		return actionResultMsg{action: "restore", err: err}
	}
}

// startPhysicalRestoreCmd dispatches a restore command and returns a
// physicalRestoreResultMsg instead of the normal actionResultMsg. This
// allows the root model to distinguish physical restores and trigger
// a clean TUI exit on success.
func startPhysicalRestoreCmd(ctx context.Context, client *sdk.Client, cmd sdk.StartRestoreCommand) tea.Cmd {
	return func() tea.Msg {
		_, err := client.Restores.Start(ctx, cmd)
		return physicalRestoreResultMsg{err: err}
	}
}

// --- Fetch commands ---

// fetchBackupsCmd returns a tea.Cmd that fetches the backup list and PITR
// timelines concurrently.
func fetchBackupsCmd(ctx context.Context, client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		var d backupsData
		var errs firstErrCollector

		g, gctx := errgroup.WithContext(ctx)

		g.Go(func() error {
			v, err := client.Backups.List(gctx, sdk.ListBackupsOptions{})
			d.backups = v
			errs.set(err)
			return nil
		})

		g.Go(func() error {
			v, err := client.PITR.Timelines(gctx)
			d.timelines = v
			errs.set(err)
			return nil
		})

		_ = g.Wait()
		d.err = errs.err
		return backupsDataMsg{d}
	}
}

// fetchRestoresCmd returns a tea.Cmd that fetches the full restore list.
func fetchRestoresCmd(ctx context.Context, client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		restores, err := client.Restores.List(ctx, sdk.ListRestoresOptions{})
		return restoresDataMsg{restoresData{restores: restores, err: err}}
	}
}

// --- PITR commands ---

// enablePITRCmd returns a tea.Cmd that enables PITR oplog slicing.
func enablePITRCmd(ctx context.Context, client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		err := client.PITR.Enable(ctx)
		return actionResultMsg{action: "enable PITR", err: err}
	}
}

// disablePITRCmd returns a tea.Cmd that disables PITR oplog slicing.
func disablePITRCmd(ctx context.Context, client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		err := client.PITR.Disable(ctx)
		return actionResultMsg{action: "disable PITR", err: err}
	}
}

// deletePITRCmd returns a tea.Cmd that deletes PITR chunks matching the
// given command (DeletePITRBefore or DeletePITROlderThan).
func deletePITRCmd(ctx context.Context, client *sdk.Client, cmd sdk.DeletePITRCommand) tea.Cmd {
	return func() tea.Msg {
		_, err := client.PITR.Delete(ctx, cmd)
		return actionResultMsg{action: "bulk delete", err: err}
	}
}
