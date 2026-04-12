package tui

import (
	"fmt"
	"time"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/jcechace/pbmate/datefield"
	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// bulkDeleteTarget identifies whether the user wants to delete backups or PITR chunks.
type bulkDeleteTarget string

const (
	bulkDeleteBackups bulkDeleteTarget = "backups"
	bulkDeletePITR    bulkDeleteTarget = "pitr"
)

// bulkDeletePreset represents a preset "older than" duration option.
type bulkDeletePreset string

const (
	presetNow    bulkDeletePreset = "now"
	preset1Day   bulkDeletePreset = "1d"
	preset3Days  bulkDeletePreset = "3d"
	preset1Week  bulkDeletePreset = "1w"
	preset2Weeks bulkDeletePreset = "2w"
	preset1Month bulkDeletePreset = "1m"
	presetCustom bulkDeletePreset = "custom"
)

// bulkDeleteFormResult holds the user's selections from the bulk delete form.
type bulkDeleteFormResult struct {
	target     bulkDeleteTarget
	preset     bulkDeletePreset
	customDate time.Time // user-selected date for presetCustom
	backupType string    // "all", "logical", "physical", "incremental"
	configName string    // profile name, "main" for main config
	confirmed  bool

	// profiles is stored for form rebuilds.
	profiles []sdk.StorageProfile
}

// presetDuration returns the time.Duration for the selected preset.
// Returns 0 for presetNow and -1 for presetCustom (caller must parse customDate).
func (r *bulkDeleteFormResult) presetDuration() time.Duration {
	switch r.preset {
	case presetNow:
		return 0
	case preset1Day:
		return 24 * time.Hour
	case preset3Days:
		return 3 * 24 * time.Hour
	case preset1Week:
		return 7 * 24 * time.Hour
	case preset2Weeks:
		return 14 * 24 * time.Hour
	case preset1Month:
		return 30 * 24 * time.Hour
	default:
		return -1 // custom
	}
}

// toBackupCommand converts the form result into a sealed SDK DeleteBackupCommand.
// Returns the command and nil error on success. Returns an error if the custom
// date cannot be parsed.
func (r *bulkDeleteFormResult) toBackupCommand() (sdk.DeleteBackupCommand, error) {
	configName := sdk.ConfigName{}
	if r.configName != defaultConfigName {
		if cn, err := sdk.NewConfigName(r.configName); err == nil {
			configName = cn
		}
	}

	backupType := sdk.BackupType{}
	switch r.backupType {
	case "logical":
		backupType = sdk.BackupTypeLogical
	case "physical":
		backupType = sdk.BackupTypePhysical
	case "incremental":
		backupType = sdk.BackupTypeIncremental
	}

	d := r.presetDuration()
	if d >= 0 {
		return sdk.DeleteBackupsOlderThan{
			OlderThan:  d,
			Type:       backupType,
			ConfigName: configName,
		}, nil
	}

	// Custom date.
	return sdk.DeleteBackupsBefore{
		OlderThan:  r.customDate,
		Type:       backupType,
		ConfigName: configName,
	}, nil
}

// toPITRCommand converts the form result into a sealed SDK DeletePITRCommand.
// Returns the command and nil error on success. Returns an error if the custom
// date cannot be parsed.
func (r *bulkDeleteFormResult) toPITRCommand() (sdk.DeletePITRCommand, error) {
	d := r.presetDuration()
	if d >= 0 {
		return sdk.DeletePITROlderThan{OlderThan: d}, nil
	}

	// Custom date.
	return sdk.DeletePITRBefore{OlderThan: r.customDate}, nil
}

// confirmTitle returns a descriptive confirm question for the current selections.
func (r *bulkDeleteFormResult) confirmTitle() string {
	target := "backups"
	if r.target == bulkDeletePITR {
		target = "PITR chunks"
	}

	age := r.presetLabel()
	return fmt.Sprintf("Delete %s older than %s?", target, age)
}

// presetLabel returns a human-readable label for the selected preset.
func (r *bulkDeleteFormResult) presetLabel() string {
	switch r.preset {
	case presetNow:
		return "now (all)"
	case preset1Day:
		return "1 day"
	case preset3Days:
		return "3 days"
	case preset1Week:
		return "1 week"
	case preset2Weeks:
		return "2 weeks"
	case preset1Month:
		return "1 month"
	case presetCustom:
		if !r.customDate.IsZero() {
			return r.customDate.UTC().Format("2006-01-02 15:04")
		}
		return "custom date"
	default:
		return string(r.preset)
	}
}

// newBulkDeleteForm creates a single-screen form for configuring a bulk delete
// operation. Groups are built dynamically based on the target — backup-specific
// fields (type, profile) are only shown when target is "backups". The custom
// date input is only shown when the preset is "custom".
func newBulkDeleteForm(formTheme huh.Theme, profiles []sdk.StorageProfile, initial *bulkDeleteFormResult) (*huh.Form, *bulkDeleteFormResult) {
	result := &bulkDeleteFormResult{
		target:     bulkDeleteBackups,
		preset:     preset1Week,
		backupType: "all",
		configName: defaultConfigName,
		confirmed:  true,
		profiles:   profiles,
	}
	if initial != nil {
		result.target = initial.target
		result.preset = initial.preset
		result.customDate = initial.customDate
		result.backupType = initial.backupType
		result.configName = initial.configName
		result.profiles = initial.profiles
	}

	// Target selector: Backups or PITR.
	groups := []*huh.Group{
		huh.NewGroup(
			huh.NewSelect[bulkDeleteTarget]().
				Title("Target").
				Options(
					huh.NewOption("Backups", bulkDeleteBackups),
					huh.NewOption("PITR chunks", bulkDeletePITR),
				).
				Inline(true).
				Value(&result.target),

			huh.NewSelect[bulkDeletePreset]().
				Title("Older than").
				Options(
					huh.NewOption("Now (all)", presetNow),
					huh.NewOption("1 day", preset1Day),
					huh.NewOption("3 days", preset3Days),
					huh.NewOption("1 week", preset1Week),
					huh.NewOption("2 weeks", preset2Weeks),
					huh.NewOption("1 month", preset1Month),
					huh.NewOption("Custom", presetCustom),
				).
				Inline(true).
				Value(&result.preset),
		),
	}

	// Custom date input — only shown when preset is "custom".
	if result.preset == presetCustom {
		initial := result.customDate
		if initial.IsZero() {
			initial = time.Now().UTC()
		}
		groups = append(groups, huh.NewGroup(
			datefield.New(initial).
				Title("Date").
				Mode(datefield.ModeDateTimeSec).
				Value(&result.customDate),
		))
	}

	// Backup-specific fields: type and profile.
	if result.target == bulkDeleteBackups {
		profileOpts := []huh.Option[string]{
			huh.NewOption("Main", defaultConfigName),
		}
		for _, p := range profiles {
			profileOpts = append(profileOpts, huh.NewOption(p.Name.String(), p.Name.String()))
		}

		groups = append(groups, huh.NewGroup(
			huh.NewSelect[string]().
				Title("Backup type").
				Options(
					huh.NewOption("All", "all"),
					huh.NewOption("Logical", "logical"),
					huh.NewOption("Physical", "physical"),
					huh.NewOption("Incremental", "incremental"),
				).
				Inline(true).
				Value(&result.backupType),

			huh.NewSelect[string]().
				Title("Profile").
				Options(profileOpts...).
				Inline(true).
				Value(&result.configName),
		))
	}

	// Confirm group with dynamic title.
	groups = append(groups, huh.NewGroup(
		huh.NewConfirm().
			Title(result.confirmTitle()).
			WithButtonAlignment(lipgloss.Left).
			Affirmative("Delete").
			Negative("Cancel").
			Value(&result.confirmed),
	))

	form := newStandardForm(groups, formTheme)
	return form, result
}
