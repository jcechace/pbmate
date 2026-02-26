package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

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

// restoreScope constants for the restore scope selector.
const (
	restoreScopeFull      = "full"
	restoreScopeSelective = "selective"
)

// restoreFormResult holds the user's selections from the restore form.
type restoreFormResult struct {
	scope            string // "full" or "selective" (snapshot mode)
	pitrPreset       string // selected PITR preset value (timestamp or "custom")
	pitrTarget       string // human-readable datetime (PITR mode only)
	pitrBaseName     string // selected base backup name (PITR mode only)
	namespaces       string // comma-separated, optional
	usersAndRoles    bool
	parallelColls    string // "" = server default
	insertionWorkers string // "" = server default
	confirmed        bool
}

// isSelective returns true when the user selected selective restore scope.
func (r *restoreFormResult) isSelective() bool {
	return r.scope == restoreScopeSelective
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
// Returns an error if the PITR target cannot be parsed.
func (r restoreFormResult) toPITRCommand(backupName string) (sdk.StartPITRRestore, error) {
	target, err := parsePITRTarget(r.effectivePITRTarget())
	if err != nil {
		return sdk.StartPITRRestore{}, err
	}
	cmd := sdk.StartPITRRestore{
		BackupName:          backupName,
		Target:              target,
		NumParallelColls:    parseOptionalInt(r.parallelColls),
		NumInsertionWorkers: parseOptionalInt(r.insertionWorkers),
	}
	// Only include selective fields when scope is selective.
	// Prevents stale values leaking after switching back to full.
	if r.isSelective() {
		cmd.Namespaces = r.parseNamespaces()
		cmd.UsersAndRoles = r.usersAndRoles
	}
	return cmd, nil
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

// --- Restore target form (Step 1) ---

// restoreTargetResult holds the user's selections from the restore target form.
// This is Step 1 of the restore wizard — it determines what to restore.
// Step 2 (the restore options form) determines how.
type restoreTargetResult struct {
	restoreType  restoreMode // restoreModeSnapshot or restoreModePITR
	profileName  string      // selected profile filter (snapshot mode)
	backupName   string      // selected backup name (snapshot mode)
	pitrPreset   string      // selected PITR preset (pitr mode)
	pitrTarget   string      // custom target datetime (pitr mode, when preset == "custom")
	pitrBaseName string      // selected base backup name (pitr mode)
	confirmed    bool
}

// effectivePITRTarget returns the PITR target to use from Step 1. If the user
// selected a preset, that value is used; otherwise the custom input is used.
func (r *restoreTargetResult) effectivePITRTarget() string {
	if r.pitrPreset != pitrPresetCustom {
		return r.pitrPreset
	}
	return r.pitrTarget
}

// newRestoreTargetForm creates the restore target form (Step 1 of the wizard).
// The user picks the restore type (Snapshot/PITR) and the specific target.
// Groups are built dynamically based on type — the form is rebuilt when type
// or PITR preset changes (see restoreTargetOverlay).
func newRestoreTargetForm(formTheme *huh.Theme, backups []sdk.Backup, timelines []sdk.Timeline, initial *restoreTargetResult) (*huh.Form, *restoreTargetResult) {
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
				result.pitrTarget = result.pitrPreset
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
				groups = append(groups, huh.NewGroup(
					huh.NewInput().
						Title("Custom target").
						Placeholder(pitrTargetFormat).
						Value(&result.pitrTarget).
						Validate(func(s string) error {
							_, err := parsePITRTarget(s)
							return err
						}),
				))
			}

			// Base backup selector: filter backups valid for the
			// effective PITR target using the same criteria as the SDK.
			baseOpts := pitrBaseOptions(result.effectivePITRTarget(), backups, timelines)
			if len(baseOpts) > 0 {
				if !hasOptionValue(baseOpts, result.pitrBaseName) {
					result.pitrBaseName = baseOpts[0].Value
				}
				groups = append(groups, huh.NewGroup(
					huh.NewSelect[string]().
						Title("Base backup").
						Options(baseOpts...).
						Value(&result.pitrBaseName),
				))
			} else {
				// No valid base — show a note and leave pitrBaseName empty.
				result.pitrBaseName = ""
				groups = append(groups, huh.NewGroup(
					huh.NewNote().
						Title("Base backup").
						Description("No valid base backup for this target"),
				))
			}
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

// completedBackupProfiles returns huh.Option entries for the distinct
// profile names found among completed backups. "Main" is always first.
func completedBackupProfiles(backups []sdk.Backup) []huh.Option[string] {
	seen := make(map[string]bool)
	var named []string
	for i := range backups {
		bk := &backups[i]
		if !bk.Status.Equal(sdk.StatusDone) {
			continue
		}
		cn := bk.ConfigName.String()
		if cn == "" || cn == defaultConfigName {
			seen[defaultConfigName] = true
			continue
		}
		if !seen[cn] {
			seen[cn] = true
			named = append(named, cn)
		}
	}

	var opts []huh.Option[string]
	if seen[defaultConfigName] {
		opts = append(opts, huh.NewOption("Main", defaultConfigName))
	}
	for _, n := range named {
		opts = append(opts, huh.NewOption(n, n))
	}
	return opts
}

// completedBackupOptions returns huh.Option entries for completed backups
// matching the given profile filter. Each option label shows the backup name
// with type and size; the value is the backup name.
func completedBackupOptions(backups []sdk.Backup, profile string) []huh.Option[string] {
	var opts []huh.Option[string]
	for i := range backups {
		bk := &backups[i]
		if !bk.Status.Equal(sdk.StatusDone) {
			continue
		}
		if bk.ConfigName.String() != profile {
			continue
		}
		label := fmt.Sprintf("%s  %s", bk.Name, bk.Type)
		if bk.Size > 0 {
			label += "  " + humanBytes(bk.Size)
		}
		opts = append(opts, huh.NewOption(label, bk.Name))
	}
	return opts
}

// hasOptionValue reports whether any option in the slice has the given value.
func hasOptionValue(opts []huh.Option[string], value string) bool {
	for _, o := range opts {
		if o.Value == value {
			return true
		}
	}
	return false
}

// latestTimeline returns the timeline with the most recent End timestamp,
// or nil if the slice is empty.
func latestTimeline(timelines []sdk.Timeline) *sdk.Timeline {
	if len(timelines) == 0 {
		return nil
	}
	best := &timelines[0]
	for i := 1; i < len(timelines); i++ {
		if timelines[i].End.T > best.End.T {
			best = &timelines[i]
		}
	}
	return best
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
func newSnapshotRestoreForm(formTheme *huh.Theme, bk *sdk.Backup, initial *restoreFormResult) (*huh.Form, *restoreFormResult) {
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
// Groups are built dynamically based on pitrPreset, base backup type, and
// scope — the form is rebuilt when any of these change (see restoreFormOverlay).
// initial carries values from a previous form state during rebuild (nil for
// first open).
//
// When the selected base backup is physical or incremental, scope and tuning
// knobs are omitted — PBM uses the physical restore path where they don't
// apply. This mirrors the snapshot form behavior for physical backups.
func newPITRRestoreForm(formTheme *huh.Theme, timeline *sdk.Timeline, backups []sdk.Backup, timelines []sdk.Timeline, initial *restoreFormResult) (*huh.Form, *restoreFormResult) {
	result := &restoreFormResult{
		scope:     restoreScopeFull,
		confirmed: true,
	}

	// Default to latest.
	result.pitrPreset = timeline.End.Time().UTC().Format(pitrTargetFormat)
	result.pitrTarget = result.pitrPreset

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
		groups = append(groups, huh.NewGroup(
			huh.NewInput().
				Title("Custom target").
				Placeholder(pitrTargetFormat).
				Value(&result.pitrTarget).
				Validate(func(s string) error {
					_, err := parsePITRTarget(s)
					return err
				}),
		))
	}

	// Base backup selector for PITR.
	baseOpts := pitrBaseOptions(result.effectivePITRTarget(), backups, timelines)
	if len(baseOpts) > 0 {
		if !hasOptionValue(baseOpts, result.pitrBaseName) {
			result.pitrBaseName = baseOpts[0].Value
		}
		groups = append(groups, huh.NewGroup(
			huh.NewSelect[string]().
				Title("Base backup").
				Options(baseOpts...).
				Value(&result.pitrBaseName),
		))
	} else {
		result.pitrBaseName = ""
		groups = append(groups, huh.NewGroup(
			huh.NewNote().
				Title("Base backup").
				Description("No valid base backup for this target"),
		))
	}

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

// --- PITR base backup selection ---

// pitrBaseOptions returns huh.Option entries for backups that are valid PITR
// base snapshots for the given target time string. Uses [sdk.FilterPITRBases]
// to apply the full validation criteria (status, config, timeline coverage).
// Returns nil if the target cannot be parsed or no valid bases exist.
func pitrBaseOptions(targetStr string, backups []sdk.Backup, timelines []sdk.Timeline) []huh.Option[string] {
	target, err := parsePITRTarget(targetStr)
	if err != nil {
		return nil
	}

	bases := sdk.FilterPITRBases(target, backups, timelines)
	if len(bases) == 0 {
		return nil
	}

	var opts []huh.Option[string]
	for i := range bases {
		bk := &bases[i]
		label := fmt.Sprintf("%s  %s", bk.Name, bk.Type)
		if bk.Size > 0 {
			label += "  " + humanBytes(bk.Size)
		}
		opts = append(opts, huh.NewOption(label, bk.Name))
	}
	return opts
}
