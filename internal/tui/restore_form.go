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

// restoreFormResult holds the user's selections from the restore form.
type restoreFormResult struct {
	restoreType      string // "snapshot" or "pitr"
	pitrTarget       string // human-readable datetime (pre-filled from timeline)
	namespaces       string // comma-separated, optional
	usersAndRoles    bool
	parallelColls    string // "" = server default
	insertionWorkers string // "" = server default
	confirmed        bool
}

// toCommand converts the form result into a sealed SDK StartRestoreCommand.
// backupName is the source backup name (from the selected backup, not from
// user input).
func (r restoreFormResult) toCommand(backupName string) (sdk.StartRestoreCommand, error) {
	numParallelColls := parseOptionalInt(r.parallelColls)
	numInsertionWorkers := parseOptionalInt(r.insertionWorkers)

	var namespaces []string
	if r.namespaces != "" && r.namespaces != "*.*" {
		nss := strings.Split(r.namespaces, ",")
		for i := range nss {
			nss[i] = strings.TrimSpace(nss[i])
		}
		namespaces = nss
	}

	if r.restoreType == "pitr" {
		target, err := parsePITRTarget(r.pitrTarget)
		if err != nil {
			return nil, err
		}
		return sdk.StartPITRRestore{
			BackupName:          backupName,
			Target:              target,
			Namespaces:          namespaces,
			UsersAndRoles:       r.usersAndRoles,
			NumParallelColls:    numParallelColls,
			NumInsertionWorkers: numInsertionWorkers,
		}, nil
	}

	// Default: snapshot restore.
	return sdk.StartSnapshotRestore{
		BackupName:          backupName,
		Namespaces:          namespaces,
		UsersAndRoles:       r.usersAndRoles,
		NumParallelColls:    numParallelColls,
		NumInsertionWorkers: numInsertionWorkers,
	}, nil
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

// newRestoreForm creates a multi-group wizard form for configuring a restore.
// backupName is displayed in the confirmation step. timelines provides the
// available PITR ranges for pre-filling and display.
func newRestoreForm(formTheme *huh.Theme, backupName string, timelines []sdk.Timeline) (*huh.Form, *restoreFormResult) {
	result := &restoreFormResult{
		restoreType: "snapshot",
		confirmed:   true,
	}

	// Pre-fill PITR target with the latest available timestamp.
	if len(timelines) > 0 {
		latest := timelines[len(timelines)-1]
		result.pitrTarget = latest.End.Time().UTC().Format(pitrTargetFormat)
	}

	// Build PITR range description for display.
	pitrRangeNote := "No PITR timelines available"
	if len(timelines) > 0 {
		latest := timelines[len(timelines)-1]
		start := latest.Start.Time().UTC().Format(pitrTargetFormat)
		end := latest.End.Time().UTC().Format(pitrTargetFormat)
		pitrRangeNote = fmt.Sprintf("Range: **%s** → **%s**", start, end)
		if len(timelines) > 1 {
			pitrRangeNote += fmt.Sprintf("\n(%d timeline segments available)", len(timelines))
		}
	}

	form := huh.NewForm(
		// Group 1: Restore type.
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Restore Type").
				Options(
					huh.NewOption("Snapshot", "snapshot"),
					huh.NewOption("Point-in-Time", "pitr"),
				).
				Value(&result.restoreType),
		),

		// Group 2: PITR target (shown only for PITR restores).
		huh.NewGroup(
			huh.NewNote().
				Title("PITR Target").
				Description(pitrRangeNote),

			huh.NewInput().
				Title("Restore to").
				Placeholder(pitrTargetFormat).
				Value(&result.pitrTarget),
		).WithHideFunc(func() bool {
			return result.restoreType != "pitr"
		}),

		// Group 3: Namespaces (optional).
		huh.NewGroup(
			huh.NewInput().
				Title("Namespaces").
				Placeholder("*.*  (all)").
				Value(&result.namespaces),
		),

		// Group 4: Users and Roles.
		huh.NewGroup(
			huh.NewConfirm().
				Title("Include users and roles?").
				Affirmative("Yes").
				Negative("No").
				Value(&result.usersAndRoles),
		),

		// Group 5: Performance tuning.
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

		// Group 6: Confirmation.
		huh.NewGroup(
			huh.NewConfirm().
				TitleFunc(func() string {
					return fmt.Sprintf("Restore %s from %s?", result.restoreType, backupName)
				}, &result.restoreType).
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
