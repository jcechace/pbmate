package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/dustin/go-humanize"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// pitrTargetFormat is the datetime format for PITR target input/display.
const pitrTargetFormat = "2006-01-02T15:04:05"

// pitrTargetFormatAlt is an alternative format accepted for PITR target input.
const pitrTargetFormatAlt = "2006-01-02 15:04:05"

// pitrPresetCustom is the sentinel value for the "Custom..." PITR preset.
const pitrPresetCustom = "custom"

// restoreMode determines the restore type based on the selected list item.
type restoreMode int

const (
	restoreModeSnapshot restoreMode = iota
	restoreModePITR
)

// restoreFormResult holds the user's selections from the restore form.
type restoreFormResult struct {
	pitrPreset       string // selected PITR preset value (timestamp or "custom")
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

// effectivePITRTarget returns the PITR target to use. If the user selected
// a preset, that value is used; otherwise the custom input is used.
func (r *restoreFormResult) effectivePITRTarget() string {
	if r.pitrPreset != pitrPresetCustom {
		return r.pitrPreset
	}
	return r.pitrTarget
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
	target, err := parsePITRTarget(r.effectivePITRTarget())
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

// --- Snapshot restore form ---

// backupContextDescription builds a short description of a backup for display
// in the restore form header.
func backupContextDescription(bk *sdk.Backup) string {
	parts := []string{bk.Type.String()}

	parts = append(parts, bk.Status.String())

	if bk.Size > 0 {
		parts = append(parts, humanize.IBytes(uint64(bk.Size)))
	}

	if bk.ConfigName.String() != "" {
		parts = append(parts, bk.ConfigName.String())
	}

	desc := strings.Join(parts, "  ")

	// Add chain context for incremental backups.
	if bk.IsIncremental() && bk.SrcBackup != "" {
		desc += fmt.Sprintf("\nChain parent: %s", bk.SrcBackup)
	}

	return desc
}

// newSnapshotRestoreForm creates a single-screen form for restoring from
// a specific backup. Shows backup context in the header.
func newSnapshotRestoreForm(formTheme *huh.Theme, bk *sdk.Backup) (*huh.Form, *restoreFormResult) {
	result := &restoreFormResult{confirmed: true}

	form := huh.NewForm(
		// Context header.
		huh.NewGroup(
			huh.NewNote().
				Title(bk.Name).
				Description(backupContextDescription(bk)),

			huh.NewInput().
				Title("Namespaces").
				Placeholder("*.*  (all)").
				Value(&result.namespaces),
		),

		// Users and roles — only for selective restores.
		huh.NewGroup(
			huh.NewConfirm().
				Title("Include users and roles?").
				Inline(true).
				Affirmative("Yes").
				Negative("No").
				Value(&result.usersAndRoles),
		).WithHideFunc(func() bool {
			return !result.isSelective()
		}),

		// Tuning.
		huh.NewGroup(
			huh.NewInput().
				Title("Parallel Collections").
				Placeholder("server default").
				Value(&result.parallelColls),

			huh.NewInput().
				Title("Insertion Workers").
				Placeholder("server default").
				Value(&result.insertionWorkers),
		),

		// Confirmation.
		huh.NewGroup(
			huh.NewConfirm().
				Title("Restore snapshot?").
				Affirmative("Restore").
				Negative("Cancel").
				Value(&result.confirmed),
		),
	).
		WithTheme(formTheme).
		WithWidth(formOverlayDefaultWidth).
		WithShowHelp(false).
		WithShowErrors(false).
		WithKeyMap(formKeyMap())

	return form, result
}

// --- PITR restore form ---

// pitrPresetOptions builds select options for the PITR target time.
// Includes "Latest", relative offsets from the timeline end, and "Custom...".
func pitrPresetOptions(timeline *sdk.Timeline) []huh.Option[string] {
	end := timeline.End.Time().UTC()
	start := timeline.Start.Time().UTC()
	duration := end.Sub(start)

	latest := end.Format(pitrTargetFormat)
	opts := []huh.Option[string]{
		huh.NewOption(fmt.Sprintf("Latest  (%s)", end.Format("15:04:05")), latest),
	}

	// Add relative offsets that fit within the timeline range.
	type preset struct {
		label  string
		offset time.Duration
	}
	presets := []preset{
		{"-5 min", 5 * time.Minute},
		{"-15 min", 15 * time.Minute},
		{"-30 min", 30 * time.Minute},
		{"-1 hour", time.Hour},
		{"-6 hours", 6 * time.Hour},
	}
	for _, p := range presets {
		if duration > p.offset {
			t := end.Add(-p.offset)
			opts = append(opts, huh.NewOption(
				fmt.Sprintf("%s  (%s)", p.label, t.Format("15:04:05")),
				t.Format(pitrTargetFormat),
			))
		}
	}

	opts = append(opts, huh.NewOption("Custom...", pitrPresetCustom))
	return opts
}

// newPITRRestoreForm creates a single-screen form for point-in-time restore.
// Offers preset time selections with a "Custom" fallback for manual entry.
func newPITRRestoreForm(formTheme *huh.Theme, timeline *sdk.Timeline) (*huh.Form, *restoreFormResult) {
	result := &restoreFormResult{confirmed: true}

	// Default to latest.
	result.pitrPreset = timeline.End.Time().UTC().Format(pitrTargetFormat)
	result.pitrTarget = result.pitrPreset

	// Build range description.
	start := timeline.Start.Time().UTC()
	end := timeline.End.Time().UTC()
	duration := end.Sub(start).Truncate(time.Second)
	rangeNote := fmt.Sprintf("%s  →  %s  (%s)",
		start.Format("15:04:05"), end.Format("15:04:05"), duration)

	form := huh.NewForm(
		// PITR target with presets.
		huh.NewGroup(
			huh.NewNote().
				Title("Timeline").
				Description(rangeNote),

			huh.NewSelect[string]().
				Title("Restore to").
				Options(pitrPresetOptions(timeline)...).
				Value(&result.pitrPreset),
		),

		// Custom timestamp input — only shown when "Custom..." is selected.
		huh.NewGroup(
			huh.NewInput().
				Title("Custom target").
				Placeholder(pitrTargetFormat).
				Value(&result.pitrTarget).
				Validate(func(s string) error {
					_, err := parsePITRTarget(s)
					return err
				}),
		).WithHideFunc(func() bool {
			return result.pitrPreset != pitrPresetCustom
		}),

		// Namespaces.
		huh.NewGroup(
			huh.NewInput().
				Title("Namespaces").
				Placeholder("*.*  (all)").
				Value(&result.namespaces),
		),

		// Users and roles — only for selective restores.
		huh.NewGroup(
			huh.NewConfirm().
				Title("Include users and roles?").
				Inline(true).
				Affirmative("Yes").
				Negative("No").
				Value(&result.usersAndRoles),
		).WithHideFunc(func() bool {
			return !result.isSelective()
		}),

		// Tuning.
		huh.NewGroup(
			huh.NewInput().
				Title("Parallel Collections").
				Placeholder("server default").
				Value(&result.parallelColls),

			huh.NewInput().
				Title("Insertion Workers").
				Placeholder("server default").
				Value(&result.insertionWorkers),
		),

		// Confirmation.
		huh.NewGroup(
			huh.NewConfirm().
				TitleFunc(func() string {
					target := result.effectivePITRTarget()
					return fmt.Sprintf("Restore PITR to %s?", target)
				}, &result.pitrPreset).
				Affirmative("Restore").
				Negative("Cancel").
				Value(&result.confirmed),
		),
	).
		WithTheme(formTheme).
		WithWidth(formOverlayDefaultWidth).
		WithShowHelp(false).
		WithShowErrors(false).
		WithKeyMap(formKeyMap())

	return form, result
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
