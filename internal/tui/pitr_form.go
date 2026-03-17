package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/huh"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// pitrTargetFormat is the datetime format used to display PITR preset values.
const pitrTargetFormat = "2006-01-02T15:04:05"

// pitrPresetCustom is the sentinel value for the "Custom..." PITR preset.
const pitrPresetCustom = "custom"

// resolvePITRTarget returns the effective PITR target as an sdk.Timestamp.
// If the preset is not "custom", the preset string (a formatted timestamp) is
// parsed and returned; otherwise the custom time.Time value is used.
// Returns zero Timestamp when the preset is empty.
func resolvePITRTarget(preset string, customTarget time.Time) sdk.Timestamp {
	if preset == pitrPresetCustom {
		return sdk.Timestamp{T: uint32(customTarget.UTC().Unix())}
	}
	if preset == "" {
		return sdk.Timestamp{}
	}
	// Preset values are formatted timestamps (pitrTargetFormat).
	if t, err := time.Parse(pitrTargetFormat, preset); err == nil {
		return sdk.Timestamp{T: uint32(t.UTC().Unix())}
	}
	return sdk.Timestamp{}
}

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

// pitrBaseGroup builds a form group for the base backup selector and updates
// the baseName pointer to a valid selection. Returns the group to append and
// the (possibly modified) baseName. Used by both newRestoreTargetForm and
// newPITRRestoreForm to avoid duplicating the filter-options-or-note logic.
func pitrBaseGroup(target sdk.Timestamp, backups []sdk.Backup, timelines []sdk.Timeline, baseName *string) *huh.Group {
	baseOpts := pitrBaseOptions(target, backups, timelines)
	if len(baseOpts) > 0 {
		if !hasOptionValue(baseOpts, *baseName) {
			*baseName = baseOpts[0].Value
		}
		return huh.NewGroup(
			huh.NewSelect[string]().
				Title("Base backup").
				Options(baseOpts...).
				Value(baseName),
		)
	}
	// No valid base — show a note and clear the selection.
	*baseName = ""
	return huh.NewGroup(
		huh.NewNote().
			Title("Base backup").
			Description("No valid base backup for this target"),
	)
}

// pitrBaseOptions returns huh.Option entries for backups that are valid PITR
// base snapshots for the given target timestamp. Uses [sdk.FilterPITRBases]
// to apply the full validation criteria (status, config, timeline coverage).
// Returns nil if no valid bases exist.
func pitrBaseOptions(target sdk.Timestamp, backups []sdk.Backup, timelines []sdk.Timeline) []huh.Option[string] {
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
		if timelines[i].End.After(best.End) {
			best = &timelines[i]
		}
	}
	return best
}

// splitNamespaces splits a comma-separated namespace string into a slice,
// trimming whitespace and filtering empty entries (e.g. trailing commas,
// whitespace-only entries, or empty input which yields [""] from Split).
// Returns nil when no non-empty entries remain.
func splitNamespaces(s string) []string {
	parts := strings.Split(s, ",")
	var nss []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			nss = append(nss, p)
		}
	}
	if len(nss) == 0 {
		return nil
	}
	return nss
}
