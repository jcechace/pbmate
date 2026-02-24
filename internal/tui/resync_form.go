package tui

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// resyncScope identifies which storage(s) to resync.
type resyncScope string

const (
	resyncScopeMain    resyncScope = "main"
	resyncScopeProfile resyncScope = "profile"
	resyncScopeAll     resyncScope = "all"
)

// resyncFormResult holds the user's selections from the resync form.
type resyncFormResult struct {
	scope           resyncScope
	profileName     string // set when scope == resyncScopeProfile
	includeRestores bool   // Main scope: also resync restore metadata
	clear           bool   // Profile/All scope: clear local metadata first
	confirmed       bool
}

// toCommand converts the form result into a sealed SDK ResyncCommand.
func (r *resyncFormResult) toCommand() sdk.ResyncCommand {
	switch r.scope {
	case resyncScopeProfile:
		return sdk.ResyncProfile{Name: r.profileName, Clear: r.clear}
	case resyncScopeAll:
		return sdk.ResyncAllProfiles{Clear: r.clear}
	default:
		return sdk.ResyncMain{IncludeRestores: r.includeRestores}
	}
}

// newResyncForm creates a single-screen form for configuring a resync operation.
// profiles is the list of named storage profiles (may be empty).
func newResyncForm(formTheme *huh.Theme, profiles []sdk.StorageProfile) (*huh.Form, *resyncFormResult) {
	result := &resyncFormResult{
		scope:     resyncScopeMain,
		confirmed: true,
	}

	// Scope options: always Main and All; Profile only when profiles exist.
	scopeOpts := []huh.Option[resyncScope]{
		huh.NewOption("Main", resyncScopeMain),
	}
	if len(profiles) > 0 {
		scopeOpts = append(scopeOpts, huh.NewOption("Profile", resyncScopeProfile))
	}
	scopeOpts = append(scopeOpts, huh.NewOption("All", resyncScopeAll))

	// Profile name options for the dropdown.
	profileOpts := make([]huh.Option[string], 0, len(profiles))
	for _, p := range profiles {
		profileOpts = append(profileOpts, huh.NewOption(p.Name.String(), p.Name.String()))
	}
	// Set a default so the value is never empty when the group is visible.
	if len(profiles) > 0 {
		result.profileName = profiles[0].Name.String()
	}

	theme := *formTheme
	theme.Focused.Base = theme.Focused.Base.BorderStyle(lipgloss.HiddenBorder())

	form := huh.NewForm(
		// Scope selector.
		huh.NewGroup(
			huh.NewSelect[resyncScope]().
				Title("Scope").
				Options(scopeOpts...).
				Inline(len(scopeOpts) <= 3).
				Value(&result.scope),
		),

		// Profile dropdown — visible only when scope == profile.
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Profile").
				Options(profileOpts...).
				Value(&result.profileName),
		).WithHideFunc(func() bool {
			return result.scope != resyncScopeProfile
		}),

		// Main-specific option: include restores.
		huh.NewGroup(
			huh.NewConfirm().
				Title("Include restores?").
				Description("Also resync restore metadata from storage.").
				Inline(true).
				Affirmative("Yes").
				Negative("No").
				Value(&result.includeRestores),
		).WithHideFunc(func() bool {
			return result.scope != resyncScopeMain
		}),

		// Profile/All option: clear local metadata first.
		huh.NewGroup(
			huh.NewConfirm().
				Title("Clear local metadata?").
				DescriptionFunc(func() string {
					if result.scope == resyncScopeAll {
						return "Remove local metadata for all profiles before re-reading from storage."
					}
					return fmt.Sprintf("Remove local metadata for %q before re-reading from storage.", result.profileName)
				}, result).
				Inline(true).
				Affirmative("Yes").
				Negative("No").
				Value(&result.clear),
		).WithHideFunc(func() bool {
			return result.scope == resyncScopeMain
		}),

		// Confirmation.
		huh.NewGroup(
			huh.NewConfirm().
				TitleFunc(func() string {
					switch result.scope {
					case resyncScopeProfile:
						return fmt.Sprintf("Resync profile %q?", result.profileName)
					case resyncScopeAll:
						return "Resync all profiles?"
					default:
						return "Resync main storage?"
					}
				}, result).
				Affirmative("Resync").
				Negative("Cancel").
				Value(&result.confirmed),
		),
	).
		WithTheme(&theme).
		WithWidth(formOverlayDefaultWidth).
		WithShowHelp(false).
		WithShowErrors(false).
		WithKeyMap(formKeyMap())

	return form, result
}
