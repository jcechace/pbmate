package tui

import (
	"context"
	"fmt"
	"os"
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

// backupFormReadyMsg carries fetched profiles so the backup form can be created.
type backupFormReadyMsg struct {
	profiles []sdk.StorageProfile
	kind     backupFormKind
}

// fetchProfilesCmd returns a tea.Cmd that fetches storage profiles for the
// backup form. Errors are silently ignored — the form will just show "Main".
func fetchProfilesCmd(ctx context.Context, client *sdk.Client, kind backupFormKind) tea.Cmd {
	return func() tea.Msg {
		profiles, _ := client.Config.ListProfiles(ctx)
		return backupFormReadyMsg{profiles: profiles, kind: kind}
	}
}

// startBackupCmd returns a tea.Cmd that starts a backup with the given command.
func startBackupCmd(ctx context.Context, client *sdk.Client, cmd sdk.StartBackupCommand) tea.Cmd {
	return func() tea.Msg {
		_, err := client.Backups.Start(ctx, cmd)
		return backupActionMsg{action: "start", err: err}
	}
}

// cancelBackupCmd returns a tea.Cmd that cancels the running backup.
func cancelBackupCmd(ctx context.Context, client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		_, err := client.Backups.Cancel(ctx)
		return backupActionMsg{action: "cancel", err: err}
	}
}

// deleteBackupCmd returns a tea.Cmd that deletes the named backup.
func deleteBackupCmd(ctx context.Context, client *sdk.Client, name string) tea.Cmd {
	return func() tea.Msg {
		_, err := client.Backups.Delete(ctx, sdk.DeleteBackupByName{Name: name})
		return backupActionMsg{action: "delete", err: err}
	}
}

// fetchOverviewCmd returns a tea.Cmd that fetches all overview data from the
// SDK client concurrently. Errors from individual calls are coalesced into
// the first encountered error; partial data is still returned. When skipLogs
// is true, log fetching is skipped (e.g. during follow mode where logs stream
// separately).
func fetchOverviewCmd(ctx context.Context, client *sdk.Client, skipLogs bool) tea.Cmd {
	return func() tea.Msg {
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
func fetchBackupsCmd(ctx context.Context, client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
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

// restoreActionMsg carries the result of a restore action (start).
type restoreActionMsg struct {
	action string // "restore"
	err    error
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
	mode       restoreMode
	backup     *sdk.Backup   // set for snapshot mode (full object for context display)
	backupName string        // set for snapshot mode
	timeline   *sdk.Timeline // set for PITR mode
	backups    []sdk.Backup  // set for PITR mode (for base backup auto-selection)
}

// startRestoreCmd returns a tea.Cmd that starts a restore with the given command.
func startRestoreCmd(ctx context.Context, client *sdk.Client, cmd sdk.StartRestoreCommand) tea.Cmd {
	return func() tea.Msg {
		_, err := client.Restores.Start(ctx, cmd)
		return restoreActionMsg{action: "restore", err: err}
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
func fetchRestoresCmd(ctx context.Context, client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		restores, err := client.Restores.List(ctx, sdk.ListRestoresOptions{})
		return restoresDataMsg{restoresData{restores: restores, err: err}}
	}
}

// --- Config tab data ---

// configData holds the result of a single config poll cycle.
type configData struct {
	config   *sdk.Config
	yaml     []byte
	profiles []sdk.StorageProfile
	err      error
}

// configDataMsg wraps configData as a BubbleTea message.
type configDataMsg struct{ configData }

// profileYAMLMsg carries a lazily-fetched profile YAML.
type profileYAMLMsg struct {
	name string
	yaml []byte
	err  error
}

// fetchConfigCmd returns a tea.Cmd that fetches config data concurrently.
func fetchConfigCmd(ctx context.Context, client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		var d configData

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
			v, err := client.Config.Get(gctx)
			d.config = v
			setErr(err)
			return nil
		})

		g.Go(func() error {
			v, err := client.Config.GetYAML(gctx)
			d.yaml = v
			setErr(err)
			return nil
		})

		g.Go(func() error {
			v, err := client.Config.ListProfiles(gctx)
			d.profiles = v
			setErr(err)
			return nil
		})

		_ = g.Wait()
		return configDataMsg{d}
	}
}

// fetchProfileYAMLCmd returns a tea.Cmd that fetches the YAML for a
// single storage profile by name.
func fetchProfileYAMLCmd(ctx context.Context, client *sdk.Client, name string) tea.Cmd {
	return func() tea.Msg {
		yaml, err := client.Config.GetProfileYAML(ctx, name)
		return profileYAMLMsg{name: name, yaml: yaml, err: err}
	}
}

// configActionMsg carries the result of a config action (apply config, set/create profile).
type configActionMsg struct {
	action string // "apply config", "set profile", "create profile"
	err    error
}

// applyConfigCmd returns a tea.Cmd that reads a YAML file and applies it
// as the main PBM configuration.
func applyConfigCmd(ctx context.Context, client *sdk.Client, filePath string) tea.Cmd {
	return func() tea.Msg {
		f, err := os.Open(filePath)
		if err != nil {
			return configActionMsg{action: "apply config", err: fmt.Errorf("open %s: %w", filePath, err)}
		}
		defer func() { _ = f.Close() }()

		err = client.Config.SetYAML(ctx, f)
		return configActionMsg{action: "apply config", err: err}
	}
}

// applyProfileCmd returns a tea.Cmd that reads a YAML file and applies it
// to a named storage profile (create or replace).
func applyProfileCmd(ctx context.Context, client *sdk.Client, name, filePath, action string) tea.Cmd {
	return func() tea.Msg {
		f, err := os.Open(filePath)
		if err != nil {
			return configActionMsg{action: action, err: fmt.Errorf("open %s: %w", filePath, err)}
		}
		defer func() { _ = f.Close() }()

		_, err = client.Config.SetProfile(ctx, name, f)
		return configActionMsg{action: action, err: err}
	}
}

// --- Resync ---

// resyncActionMsg carries the result of a resync operation.
type resyncActionMsg struct {
	action string // "resync"
	err    error
}

// resyncCmd returns a tea.Cmd that dispatches a resync command to the SDK.
func resyncCmd(ctx context.Context, client *sdk.Client, cmd sdk.ResyncCommand) tea.Cmd {
	return func() tea.Msg {
		_, err := client.Config.Resync(ctx, cmd)
		return resyncActionMsg{action: "resync", err: err}
	}
}

// --- Remove profile ---

// removeProfileCmd returns a tea.Cmd that removes a named storage profile.
func removeProfileCmd(ctx context.Context, client *sdk.Client, name string) tea.Cmd {
	return func() tea.Msg {
		_, err := client.Config.RemoveProfile(ctx, name)
		return configActionMsg{action: "remove profile", err: err}
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
