package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/jcechace/pbmate/datefield"
	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// restoreMode determines the restore type based on the selected list item.
type restoreMode int

const (
	restoreModeSnapshot restoreMode = iota
	restoreModePITR
)

// restoreScope constants for the restore scope selector.
const (
	restoreScopeFull      = "full"
	restoreScopeSelective = "selective"
)

// restoreFormResult holds the user's selections from the restore form.
type restoreFormResult struct {
	scope            string    // "full" or "selective" (snapshot and logical PITR mode)
	pitrPreset       string    // selected PITR preset value (timestamp string or "custom")
	pitrTarget       time.Time // user-selected datetime (PITR custom mode only)
	pitrBaseName     string    // selected base backup name (PITR mode only)
	namespaces       string    // comma-separated, optional
	usersAndRoles    bool
	parallelColls    string // "" = server default
	insertionWorkers string // "" = server default
	confirmed        bool
}

// isSelective returns true when the user selected selective restore scope.
func (r *restoreFormResult) isSelective() bool {
	return r.scope == restoreScopeSelective
}

// effectivePITRTarget returns the PITR target as an sdk.Timestamp.
func (r *restoreFormResult) effectivePITRTarget() sdk.Timestamp {
	return resolvePITRTarget(r.pitrPreset, r.pitrTarget)
}

// toSnapshotCommand converts the form result into a StartSnapshotRestore.
func (r *restoreFormResult) toSnapshotCommand(backupName string) sdk.StartSnapshotRestore {
	cmd := sdk.StartSnapshotRestore{
		BackupName:          backupName,
		NumParallelColls:    parseOptionalInt(r.parallelColls),
		NumInsertionWorkers: parseOptionalInt(r.insertionWorkers),
	}
	// Only include selective fields when scope is selective.
	// Prevents stale values leaking after switching back to full.
	if r.isSelective() {
		cmd.Namespaces = r.parseNamespaces()
		cmd.UsersAndRoles = r.usersAndRoles
	}
	return cmd
}

// toPITRCommand converts the form result into a StartPITRRestore.
func (r *restoreFormResult) toPITRCommand(backupName string) sdk.StartPITRRestore {
	cmd := sdk.StartPITRRestore{
		BackupName:          backupName,
		Target:              r.effectivePITRTarget(),
		NumParallelColls:    parseOptionalInt(r.parallelColls),
		NumInsertionWorkers: parseOptionalInt(r.insertionWorkers),
	}
	// Only include selective fields when scope is selective.
	// Prevents stale values leaking after switching back to full.
	if r.isSelective() {
		cmd.Namespaces = r.parseNamespaces()
		cmd.UsersAndRoles = r.usersAndRoles
	}
	return cmd
}

// parseNamespaces returns the selective namespaces for a restore form.
// Returns nil when scope is not selective.
func (r *restoreFormResult) parseNamespaces() []string {
	if !r.isSelective() {
		return nil
	}
	return splitNamespaces(r.namespaces)
}

// --- Restore target form (Step 1) ---

// restoreTargetResult holds the user's selections from the restore target form.
// This is Step 1 of the restore wizard — it determines what to restore.
// Step 2 (the restore options form) determines how.
type restoreTargetResult struct {
	restoreType  restoreMode // restoreModeSnapshot or restoreModePITR
	profileName  string      // selected profile filter (snapshot mode)
	backupName   string      // selected backup name (snapshot mode)
	pitrPreset   string      // selected PITR preset (pitr mode)
	pitrTarget   time.Time   // custom target datetime (pitr mode, when preset == "custom")
	pitrBaseName string      // selected base backup name (pitr mode)
	confirmed    bool
}

// effectivePITRTarget returns the PITR target as an sdk.Timestamp from Step 1.
func (r *restoreTargetResult) effectivePITRTarget() sdk.Timestamp {
	return resolvePITRTarget(r.pitrPreset, r.pitrTarget)
}

// newRestoreTargetForm creates the restore target form (Step 1 of the wizard).
// The user picks the restore type (Snapshot/PITR) and the specific target.
// Groups are built dynamically based on type — the form is rebuilt when type
// or PITR preset changes (see restoreTargetOverlay).
func newRestoreTargetForm(formTheme huh.Theme, backups []sdk.Backup, timelines []sdk.Timeline, initial *restoreTargetResult) (*huh.Form, *restoreTargetResult) {
	result := &restoreTargetResult{
		restoreType: restoreModeSnapshot,
		profileName: defaultConfigName,
		confirmed:   true,
	}
	if initial != nil {
		result.restoreType = initial.restoreType
		result.profileName = initial.profileName
		result.backupName = initial.backupName
		result.pitrPreset = initial.pitrPreset
		result.pitrTarget = initial.pitrTarget
		result.pitrBaseName = initial.pitrBaseName
	}

	// Type options: PITR only available when timelines exist.
	typeOpts := []huh.Option[restoreMode]{
		huh.NewOption("Snapshot", restoreModeSnapshot),
	}
	if len(timelines) > 0 {
		typeOpts = append(typeOpts, huh.NewOption("PITR", restoreModePITR))
	}

	groups := []*huh.Group{
		huh.NewGroup(
			huh.NewSelect[restoreMode]().
				Title("Type").
				Options(typeOpts...).
				Inline(true).
				Value(&result.restoreType),
		),
	}

	switch result.restoreType {
	case restoreModeSnapshot:
		// Profile filter: distinct profiles from completed backups.
		profileOpts := completedBackupProfiles(backups)
		if len(profileOpts) > 0 {
			// Ensure selected profile is valid; fall back to first available.
			if !hasOptionValue(profileOpts, result.profileName) {
				result.profileName = profileOpts[0].Value
			}
			groups = append(groups, huh.NewGroup(
				huh.NewSelect[string]().
					Title("Profile").
					Options(profileOpts...).
					Inline(true).
					Value(&result.profileName),
			))
		}

		// Backup selector: completed backups filtered by selected profile.
		backupOpts := completedBackupOptions(backups, result.profileName)
		if len(backupOpts) > 0 {
			// Default to the first backup if none pre-selected or no longer in list.
			if !hasOptionValue(backupOpts, result.backupName) {
				result.backupName = backupOpts[0].Value
			}
			groups = append(groups, huh.NewGroup(
				huh.NewSelect[string]().
					Title("Backup").
					Options(backupOpts...).
					Value(&result.backupName),
			))
		}

	case restoreModePITR:
		// Auto-select latest timeline.
		timeline := latestTimeline(timelines)
		if timeline != nil {
			// Default pitrPreset to latest.
			if result.pitrPreset == "" {
				result.pitrPreset = timeline.End.Time().UTC().Format(pitrTargetFormat)
			}

			start := timeline.Start.Time().UTC()
			end := timeline.End.Time().UTC()
			duration := end.Sub(start).Truncate(time.Second)
			rangeNote := fmt.Sprintf("%s  →  %s\n(%s)",
				start.Format("Jan 02 15:04:05"), end.Format("Jan 02 15:04:05"), duration)

			groups = append(groups, huh.NewGroup(
				huh.NewNote().
					Title("Timeline").
					Description(rangeNote),

				huh.NewSelect[string]().
					Title("Restore to").
					Options(pitrPresetOptions(timeline)...).
					Inline(true).
					Value(&result.pitrPreset),
			))

			if result.pitrPreset == pitrPresetCustom {
				// Seed pitrTarget from timeline end when first switching to Custom.
				if result.pitrTarget.IsZero() {
					result.pitrTarget = end
				}
				groups = append(groups, huh.NewGroup(
					datefield.New(result.pitrTarget).
						Mode(datefield.ModeDateTimeSec).
						Title("Custom target").
						Value(&result.pitrTarget),
				))
			}

			// Base backup selector: filter backups valid for the
			// effective PITR target using the same criteria as the SDK.
			groups = append(groups, pitrBaseGroup(result.effectivePITRTarget(), backups, timelines, &result.pitrBaseName))
		}
	}

	groups = append(groups, huh.NewGroup(
		huh.NewConfirm().
			Title("Configure restore options?").
			WithButtonAlignment(lipgloss.Left).
			Affirmative("Next").
			Negative("Cancel").
			Value(&result.confirmed),
	))

	form := newStandardForm(groups, formTheme)

	return form, result
}

// --- Snapshot restore form (Step 2) ---

// backupContextDescription builds a short description of a backup for display
// in the restore form header.
func backupContextDescription(bk *sdk.Backup) string {
	parts := []string{bk.Type.String()}

	parts = append(parts, bk.Status.String())

	if bk.Size > 0 {
		parts = append(parts, humanBytes(bk.Size))
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
// a specific backup. Groups are built dynamically based on scope — the form
// is rebuilt when scope changes (see restoreFormOverlay). initial carries
// values from a previous form state during rebuild (nil for first open).
func newSnapshotRestoreForm(formTheme huh.Theme, bk *sdk.Backup, initial *restoreFormResult) (*huh.Form, *restoreFormResult) {
	result := &restoreFormResult{
		scope:     restoreScopeFull,
		confirmed: true,
	}
	if initial != nil {
		result.scope = initial.scope
		result.namespaces = initial.namespaces
		result.usersAndRoles = initial.usersAndRoles
		result.parallelColls = initial.parallelColls
		result.insertionWorkers = initial.insertionWorkers
	}

	isPhysicalType := bk.IsPhysical() || bk.IsIncremental()

	// Physical/incremental restores operate at the file level — scope,
	// namespace filtering, and tuning knobs don't apply. Show only the
	// backup context and confirmation.
	if isPhysicalType {
		groups := []*huh.Group{
			huh.NewGroup(
				huh.NewNote().
					Title(bk.Name).
					Description(backupContextDescription(bk)),

				huh.NewConfirm().
					Title("Restore snapshot?").
					WithButtonAlignment(lipgloss.Left).
					Affirmative("Restore").
					Negative("Cancel").
					Value(&result.confirmed),
			),
		}
		form := newStandardForm(groups, formTheme)
		return form, result
	}

	// Logical backups support scope selection and tuning.
	scopeOpts := []huh.Option[string]{
		huh.NewOption("Full", restoreScopeFull),
		huh.NewOption("Selective", restoreScopeSelective),
	}

	// Build groups dynamically based on scope.
	groups := []*huh.Group{
		// Context header + scope selector.
		huh.NewGroup(
			huh.NewNote().
				Title(bk.Name).
				Description(backupContextDescription(bk)),

			huh.NewSelect[string]().
				Title("Scope").
				Options(scopeOpts...).
				Inline(true).
				Value(&result.scope),
		),
	}

	if result.scope == restoreScopeSelective {
		groups = append(groups, huh.NewGroup(
			huh.NewInput().
				Title("Namespaces").
				Placeholder("*.*  (all)").
				Value(&result.namespaces),

			huh.NewConfirm().
				Title("Include users and roles?").
				Inline(true).
				Affirmative("Yes").
				Negative("No").
				Value(&result.usersAndRoles),
		))
	}

	groups = append(groups,
		// Tuning + confirmation.
		huh.NewGroup(
			huh.NewInput().
				Title("Parallel Collections").
				Placeholder("server default").
				Value(&result.parallelColls),

			huh.NewInput().
				Title("Insertion Workers").
				Placeholder("server default").
				Value(&result.insertionWorkers),

			huh.NewConfirm().
				Title("Restore snapshot?").
				WithButtonAlignment(lipgloss.Left).
				Affirmative("Restore").
				Negative("Cancel").
				Value(&result.confirmed),
		),
	)

	form := newStandardForm(groups, formTheme)

	return form, result
}

// --- PITR restore form ---

// newPITRRestoreForm creates a single-screen form for point-in-time restore.
// Groups are built dynamically based on pitrPreset, base backup type, and
// scope — the form is rebuilt when any of these change (see restoreFormOverlay).
// initial carries values from a previous form state during rebuild (nil for
// first open).
//
// When the selected base backup is physical or incremental, scope and tuning
// knobs are omitted — PBM uses the physical restore path where they don't
// apply. This mirrors the snapshot form behavior for physical backups.
func newPITRRestoreForm(formTheme huh.Theme, timeline *sdk.Timeline, backups []sdk.Backup, timelines []sdk.Timeline, initial *restoreFormResult) (*huh.Form, *restoreFormResult) {
	result := &restoreFormResult{
		scope:     restoreScopeFull,
		confirmed: true,
	}

	// Default to latest.
	result.pitrPreset = timeline.End.Time().UTC().Format(pitrTargetFormat)

	if initial != nil {
		result.scope = initial.scope
		result.pitrPreset = initial.pitrPreset
		result.pitrTarget = initial.pitrTarget
		result.pitrBaseName = initial.pitrBaseName
		result.namespaces = initial.namespaces
		result.usersAndRoles = initial.usersAndRoles
		result.parallelColls = initial.parallelColls
		result.insertionWorkers = initial.insertionWorkers
	}

	// Build range description with dates.
	start := timeline.Start.Time().UTC()
	end := timeline.End.Time().UTC()
	duration := end.Sub(start).Truncate(time.Second)
	rangeNote := fmt.Sprintf("%s  →  %s\n(%s)",
		start.Format("Jan 02 15:04:05"), end.Format("Jan 02 15:04:05"), duration)

	// Build groups dynamically based on preset, base type, and scope.
	groups := []*huh.Group{
		// Timeline context + target preset.
		huh.NewGroup(
			huh.NewNote().
				Title("Timeline").
				Description(rangeNote),

			huh.NewSelect[string]().
				Title("Restore to").
				Options(pitrPresetOptions(timeline)...).
				Inline(true).
				Value(&result.pitrPreset),
		),
	}

	if result.pitrPreset == pitrPresetCustom {
		// Seed pitrTarget from timeline end when first switching to Custom.
		if result.pitrTarget.IsZero() {
			result.pitrTarget = end
		}
		groups = append(groups, huh.NewGroup(
			datefield.New(result.pitrTarget).
				Mode(datefield.ModeDateTimeSec).
				Title("Custom target").
				Value(&result.pitrTarget),
		))
	}

	// Base backup selector for PITR.
	groups = append(groups, pitrBaseGroup(result.effectivePITRTarget(), backups, timelines, &result.pitrBaseName))

	// Determine whether the selected base is physical/incremental.
	// Physical PITR restores use the file-level restore path — scope,
	// namespace filtering, and tuning knobs don't apply.
	isPhysicalBase := false
	if base := findBackupByName(backups, result.pitrBaseName); base != nil {
		isPhysicalBase = base.IsPhysical() || base.IsIncremental()
	}

	if !isPhysicalBase {
		// Logical base: show scope selector and tuning knobs.
		groups = append(groups, huh.NewGroup(
			huh.NewSelect[string]().
				Title("Scope").
				Options(
					huh.NewOption("Full", restoreScopeFull),
					huh.NewOption("Selective", restoreScopeSelective),
				).
				Inline(true).
				Value(&result.scope),
		))

		if result.scope == restoreScopeSelective {
			groups = append(groups, huh.NewGroup(
				huh.NewInput().
					Title("Namespaces").
					Placeholder("*.*  (all)").
					Value(&result.namespaces),

				huh.NewConfirm().
					Title("Include users and roles?").
					Inline(true).
					Affirmative("Yes").
					Negative("No").
					Value(&result.usersAndRoles),
			))
		}

		groups = append(groups,
			huh.NewGroup(
				huh.NewInput().
					Title("Parallel Collections").
					Placeholder("server default").
					Value(&result.parallelColls),

				huh.NewInput().
					Title("Insertion Workers").
					Placeholder("server default").
					Value(&result.insertionWorkers),

				huh.NewConfirm().
					Title("Restore PITR?").
					WithButtonAlignment(lipgloss.Left).
					Affirmative("Restore").
					Negative("Cancel").
					Value(&result.confirmed),
			),
		)
	} else {
		// Physical/incremental base: only confirmation.
		groups = append(groups,
			huh.NewGroup(
				huh.NewConfirm().
					Title("Restore PITR?").
					WithButtonAlignment(lipgloss.Left).
					Affirmative("Restore").
					Negative("Cancel").
					Value(&result.confirmed),
			),
		)
	}

	form := newStandardForm(groups, formTheme)

	return form, result
}
