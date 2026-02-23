package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/huh"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// pitrTargetFormat is the datetime format for PITR target input/display.
const pitrTargetFormat = "2006-01-02T15:04:05"

// pitrTargetFormatAlt is an alternative format accepted for PITR target input.
const pitrTargetFormatAlt = "2006-01-02 15:04:05"

// restoreMode determines the restore type based on the selected list item.
type restoreMode int

const (
	restoreModeSnapshot restoreMode = iota
	restoreModePITR
)

// restoreFormResult holds the user's selections from the restore form.
type restoreFormResult struct {
	pitrTarget       string // human-readable datetime (PITR mode only)
	namespaces       string // comma-separated, optional
	usersAndRoles    bool
	parallelColls    string // "" = server default
	insertionWorkers string // "" = server default
	confirmed        bool
}

// isSelective returns true when the user specified namespace filters.
func (r *restoreFormResult) isSelective() bool {
	ns := strings.TrimSpace(r.namespaces)
	return ns != "" && ns != "*.*"
}

// toSnapshotCommand converts the form result into a StartSnapshotRestore.
func (r restoreFormResult) toSnapshotCommand(backupName string) sdk.StartSnapshotRestore {
	return sdk.StartSnapshotRestore{
		BackupName:          backupName,
		Namespaces:          r.parseNamespaces(),
		UsersAndRoles:       r.usersAndRoles,
		NumParallelColls:    parseOptionalInt(r.parallelColls),
		NumInsertionWorkers: parseOptionalInt(r.insertionWorkers),
	}
}

// toPITRCommand converts the form result into a StartPITRRestore.
// Returns an error if the PITR target cannot be parsed.
func (r restoreFormResult) toPITRCommand(backupName string) (sdk.StartPITRRestore, error) {
	target, err := parsePITRTarget(r.pitrTarget)
	if err != nil {
		return sdk.StartPITRRestore{}, err
	}
	return sdk.StartPITRRestore{
		BackupName:          backupName,
		Target:              target,
		Namespaces:          r.parseNamespaces(),
		UsersAndRoles:       r.usersAndRoles,
		NumParallelColls:    parseOptionalInt(r.parallelColls),
		NumInsertionWorkers: parseOptionalInt(r.insertionWorkers),
	}, nil
}

// parseNamespaces splits the comma-separated namespace string into a slice.
// Returns nil for empty or "*.*" (full restore).
func (r restoreFormResult) parseNamespaces() []string {
	if !r.isSelective() {
		return nil
	}
	nss := strings.Split(r.namespaces, ",")
	for i := range nss {
		nss[i] = strings.TrimSpace(nss[i])
	}
	return nss
}

// parsePITRTarget parses a human-readable datetime string into an SDK Timestamp.
// Accepts both "2006-01-02T15:04:05" and "2006-01-02 15:04:05" formats.
// Input is interpreted as UTC.
func parsePITRTarget(s string) (sdk.Timestamp, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return sdk.Timestamp{}, fmt.Errorf("PITR target is required")
	}

	for _, layout := range []string{pitrTargetFormat, pitrTargetFormatAlt} {
		if t, err := time.Parse(layout, s); err == nil {
			return sdk.Timestamp{T: uint32(t.UTC().Unix())}, nil
		}
	}

	return sdk.Timestamp{}, fmt.Errorf("invalid PITR target %q: expected format %s", s, pitrTargetFormat)
}

// newSnapshotRestoreForm creates a form for restoring from a specific backup.
func newSnapshotRestoreForm(formTheme *huh.Theme, backupName string) (*huh.Form, *restoreFormResult) {
	result := &restoreFormResult{confirmed: true}

	form := huh.NewForm(
		restoreNamespacesGroup(result),
		restoreUsersAndRolesGroup(result),
		restoreTuningGroup(result),

		// Confirmation.
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Restore snapshot from %s?", backupName)).
				Affirmative("Restore").
				Negative("Cancel").
				Value(&result.confirmed),
		),
	).
		WithTheme(formTheme).
		WithWidth(formOverlayInnerWidth).
		WithShowHelp(false).
		WithShowErrors(false).
		WithKeyMap(formKeyMap())

	return form, result
}

// newPITRRestoreForm creates a form for point-in-time restore from a timeline.
func newPITRRestoreForm(formTheme *huh.Theme, timeline *sdk.Timeline) (*huh.Form, *restoreFormResult) {
	result := &restoreFormResult{confirmed: true}

	// Pre-fill target with the latest available timestamp.
	result.pitrTarget = timeline.End.Time().UTC().Format(pitrTargetFormat)

	// Build range description for display.
	start := timeline.Start.Time().UTC().Format(pitrTargetFormat)
	end := timeline.End.Time().UTC().Format(pitrTargetFormat)
	rangeNote := fmt.Sprintf("Range: %s \u2192 %s", start, end)

	form := huh.NewForm(
		// PITR target.
		huh.NewGroup(
			huh.NewNote().
				Title("PITR Timeline").
				Description(rangeNote),

			huh.NewInput().
				Title("Restore to").
				Placeholder(pitrTargetFormat).
				Value(&result.pitrTarget),
		),

		restoreNamespacesGroup(result),
		restoreUsersAndRolesGroup(result),
		restoreTuningGroup(result),

		// Confirmation.
		huh.NewGroup(
			huh.NewConfirm().
				TitleFunc(func() string {
					return fmt.Sprintf("Restore PITR to %s?", result.pitrTarget)
				}, &result.pitrTarget).
				Affirmative("Restore").
				Negative("Cancel").
				Value(&result.confirmed),
		),
	).
		WithTheme(formTheme).
		WithWidth(formOverlayInnerWidth).
		WithShowHelp(false).
		WithShowErrors(false).
		WithKeyMap(formKeyMap())

	return form, result
}

// --- Shared form groups ---

// restoreNamespacesGroup returns the optional namespace input group.
func restoreNamespacesGroup(result *restoreFormResult) *huh.Group {
	return huh.NewGroup(
		huh.NewInput().
			Title("Namespaces").
			Placeholder("*.*  (all)").
			Value(&result.namespaces),
	)
}

// restoreUsersAndRolesGroup returns the users-and-roles confirm group.
// Only shown when the user specified namespace filters (selective restore).
func restoreUsersAndRolesGroup(result *restoreFormResult) *huh.Group {
	return huh.NewGroup(
		huh.NewConfirm().
			Title("Include users and roles?").
			Description("Only for selective (namespace-filtered) restores").
			Affirmative("Yes").
			Negative("No").
			Value(&result.usersAndRoles),
	).WithHideFunc(func() bool {
		return !result.isSelective()
	})
}

// restoreTuningGroup returns the performance tuning input group.
func restoreTuningGroup(result *restoreFormResult) *huh.Group {
	return huh.NewGroup(
		huh.NewInput().
			Title("Parallel Collections").
			Placeholder("server default").
			Value(&result.parallelColls),

		huh.NewInput().
			Title("Insertion Workers").
			Placeholder("server default").
			Value(&result.insertionWorkers),
	)
}

// --- PITR base backup selection ---

// findBaseBackup finds the latest completed backup whose last write timestamp
// is at or before the given PITR target. Returns the backup name, or an error
// if no eligible backup exists.
func findBaseBackup(target sdk.Timestamp, backups []sdk.Backup) (string, error) {
	var best *sdk.Backup
	for i := range backups {
		bk := &backups[i]
		if !bk.Status.Equal(sdk.StatusDone) {
			continue
		}
		if bk.LastWriteTS.IsZero() {
			continue
		}
		if bk.LastWriteTS.T > target.T {
			continue
		}
		if best == nil || bk.LastWriteTS.T > best.LastWriteTS.T {
			best = bk
		}
	}
	if best == nil {
		return "", fmt.Errorf("no completed backup found before target %s",
			target.Time().UTC().Format(pitrTargetFormat))
	}
	return best.Name, nil
}
