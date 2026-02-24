package tui

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// resyncScope identifies the resync target category.
type resyncScope string

const (
	resyncScopeMain    resyncScope = "main"
	resyncScopeProfile resyncScope = "profile"
)

// resyncProfileAll is the sentinel value for "All profiles" in the profile selector.
const resyncProfileAll = "*"

// resyncFormResult holds the user's selections from the resync form.
type resyncFormResult struct {
	scope           resyncScope
	profileName     string // resyncProfileAll or a specific profile name
	includeRestores bool   // Main scope: also resync restore metadata
	clear           bool   // Profile scope: clear local metadata first
	confirmed       bool
}

// toCommand converts the form result into a sealed SDK ResyncCommand.
func (r *resyncFormResult) toCommand() sdk.ResyncCommand {
	switch r.scope {
	case resyncScopeProfile:
		if r.profileName == resyncProfileAll {
			return sdk.ResyncAllProfiles{Clear: r.clear}
		}
		return sdk.ResyncProfile{Name: r.profileName, Clear: r.clear}
	default:
		return sdk.ResyncMain{IncludeRestores: r.includeRestores}
	}
}

// newResyncForm creates a single-screen form for configuring a resync operation.
// Groups are built dynamically based on scope — the form is rebuilt when scope
// or profile changes (see resyncFormOverlay). initial carries values from a
// previous form state during rebuild (nil for first open).
func newResyncForm(formTheme *huh.Theme, profiles []sdk.StorageProfile, initial *resyncFormResult) (*huh.Form, *resyncFormResult) {
	result := &resyncFormResult{
		scope:       resyncScopeMain,
		profileName: resyncProfileAll,
		confirmed:   true,
	}
	if initial != nil {
		result.scope = initial.scope
		result.profileName = initial.profileName
		result.includeRestores = initial.includeRestores
		result.clear = initial.clear
	}

	// Target options: always Main; Profile only when profiles exist.
	targetOpts := []huh.Option[resyncScope]{
		huh.NewOption("Main", resyncScopeMain),
	}
	if len(profiles) > 0 {
		targetOpts = append(targetOpts, huh.NewOption("Profile", resyncScopeProfile))
	}

	// Build groups dynamically based on scope.
	groups := []*huh.Group{
		huh.NewGroup(
			huh.NewSelect[resyncScope]().
				Title("Target").
				Options(targetOpts...).
				Inline(true).
				Value(&result.scope),
		),
	}

	switch result.scope {
	case resyncScopeMain:
		groups = append(groups, huh.NewGroup(
			huh.NewConfirm().
				Title("Include restores?").
				Inline(true).
				Affirmative("Yes").
				Negative("No").
				Value(&result.includeRestores),
		))

	case resyncScopeProfile:
		// Profile options: "All" synthetic first, then individual profiles.
		profileOpts := []huh.Option[string]{
			huh.NewOption("All", resyncProfileAll),
		}
		for _, p := range profiles {
			profileOpts = append(profileOpts, huh.NewOption(p.Name.String(), p.Name.String()))
		}

		groups = append(groups, huh.NewGroup(
			huh.NewSelect[string]().
				Title("Profile").
				Options(profileOpts...).
				Inline(true).
				Value(&result.profileName),

			huh.NewConfirm().
				Title("Clear metadata?").
				Inline(true).
				Affirmative("Yes").
				Negative("No").
				Value(&result.clear),
		))
	}

	// Confirm title based on current scope and profile.
	confirmTitle := "Resync main storage?"
	if result.scope == resyncScopeProfile {
		if result.profileName == resyncProfileAll {
			confirmTitle = "Resync all profiles?"
		} else {
			confirmTitle = fmt.Sprintf("Resync profile %q?", result.profileName)
		}
	}

	groups = append(groups, huh.NewGroup(
		huh.NewConfirm().
			Title(confirmTitle).
			WithButtonAlignment(lipgloss.Left).
			Affirmative("Resync").
			Negative("Cancel").
			Value(&result.confirmed),
	))

	form := huh.NewForm(groups...).
		WithTheme(formTheme).
		WithLayout(huh.LayoutStack).
		WithWidth(formOverlayDefaultWidth).
		WithShowHelp(false).
		WithShowErrors(false).
		WithKeyMap(formKeyMap())

	return form, result
}
