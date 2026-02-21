package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// profileNameResult holds the user's input from the profile name form.
type profileNameResult struct {
	name      string
	confirmed bool
}

// newProfileNameForm creates a compact form that asks only for a profile name.
// Used as the first step of new profile creation (followed by a file picker).
func newProfileNameForm() (*huh.Form, *profileNameResult) {
	result := &profileNameResult{confirmed: true}

	theme := huh.ThemeCatppuccin()
	theme.Focused.Base = theme.Focused.Base.BorderStyle(lipgloss.HiddenBorder())

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Profile name").
				Placeholder("my-profile").
				Value(&result.name),

			huh.NewConfirm().
				Affirmative("Next").
				Negative("Cancel").
				Value(&result.confirmed),
		),
	).
		WithTheme(theme).
		WithWidth(backupFormInnerWidth).
		WithShowHelp(false).
		WithShowErrors(false).
		WithKeyMap(backupFormKeyMap())

	return form, result
}
