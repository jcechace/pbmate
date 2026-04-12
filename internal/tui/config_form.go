package tui

import (
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// setConfigTarget identifies the config target category.
type setConfigTarget string

const (
	setConfigTargetMain    setConfigTarget = "main"
	setConfigTargetProfile setConfigTarget = "profile"
)

// setConfigProfileNew is the sentinel value for creating a new profile.
const setConfigProfileNew = "*new*"

// setConfigFormResult holds the user's selections from the set config form.
type setConfigFormResult struct {
	target    setConfigTarget
	profile   string // setConfigProfileNew or an existing profile name
	newName   string // only used when profile == setConfigProfileNew
	confirmed bool
}

// isNew reports whether the user chose to create a new profile.
func (r *setConfigFormResult) isNew() bool {
	return r.target == setConfigTargetProfile && r.profile == setConfigProfileNew
}

// effectiveProfile returns the profile name that should be used for the
// file picker and apply commands.
func (r *setConfigFormResult) effectiveProfile() string {
	if r.isNew() {
		return r.newName
	}
	if r.target == setConfigTargetProfile {
		return r.profile
	}
	return ""
}

// newSetConfigForm creates a single-screen form for configuring a set-config
// operation. Groups are built dynamically based on target — the form is rebuilt
// when target or profile selection changes (see setConfigOverlay). initial
// carries values from a previous form state during rebuild (nil for first open).
func newSetConfigForm(formTheme huh.Theme, profiles []sdk.StorageProfile, initial *setConfigFormResult) (*huh.Form, *setConfigFormResult) {
	result := &setConfigFormResult{
		target:    setConfigTargetMain,
		profile:   setConfigProfileNew,
		confirmed: true,
	}
	if initial != nil {
		result.target = initial.target
		result.profile = initial.profile
		result.newName = initial.newName
	}

	// Target options: always Main; Profile only when profiles exist or "New" is viable.
	targetOpts := []huh.Option[setConfigTarget]{
		huh.NewOption("Main", setConfigTargetMain),
		huh.NewOption("Profile", setConfigTargetProfile),
	}

	// Build groups dynamically based on target.
	groups := []*huh.Group{
		huh.NewGroup(
			huh.NewSelect[setConfigTarget]().
				Title("Target").
				Options(targetOpts...).
				Inline(true).
				Value(&result.target),
		),
	}

	if result.target == setConfigTargetProfile {
		// Profile options: "New" first, then existing profiles.
		profileOpts := []huh.Option[string]{
			huh.NewOption("New", setConfigProfileNew),
		}
		for _, p := range profiles {
			profileOpts = append(profileOpts, huh.NewOption(p.Name.String(), p.Name.String()))
		}

		groups = append(groups, huh.NewGroup(
			huh.NewSelect[string]().
				Title("Profile").
				Options(profileOpts...).
				Inline(true).
				Value(&result.profile),
		))

		// Name input only for new profiles.
		if result.profile == setConfigProfileNew {
			groups = append(groups, huh.NewGroup(
				huh.NewInput().
					Title("Profile name").
					Placeholder("my-profile").
					Value(&result.newName),
			))
		}
	}

	groups = append(groups, huh.NewGroup(
		huh.NewConfirm().
			Title("Select YAML file?").
			WithButtonAlignment(lipgloss.Left).
			Affirmative("Next").
			Negative("Cancel").
			Value(&result.confirmed),
	))

	form := newStandardForm(groups, formTheme)

	return form, result
}
