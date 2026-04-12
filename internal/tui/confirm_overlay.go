package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

// confirmOverlay wraps a confirmation form with a stored action command
// that executes only if the user confirms.
type confirmOverlay struct {
	form      *huh.Form
	result    *confirmFormResult
	title     string
	actionCmd tea.Cmd // executed if confirmed
}

func newConfirmOverlay(formTheme huh.Theme, title, description, affirmative, negative string, actionCmd tea.Cmd) (*confirmOverlay, tea.Cmd) {
	form, result := newConfirmForm(formTheme, description, affirmative, negative)
	o := &confirmOverlay{form: form, result: result, title: title, actionCmd: actionCmd}
	return o, initThemedForm(o.form)
}

func (o *confirmOverlay) Update(msg tea.Msg, back, quit key.Binding) (formOverlay, tea.Cmd) {
	if dismissOverlay(msg, back, quit) {
		return nil, nil
	}

	cmd := updateFormModel(&o.form, msg)

	if o.form.State == huh.StateCompleted {
		if o.result.confirmed {
			return nil, o.actionCmd
		}
		return nil, nil
	}

	if o.form.State == huh.StateAborted {
		return nil, nil
	}

	return o, cmd
}

func (o *confirmOverlay) View(styles *Styles, contentW, contentH int) string {
	return renderFormOverlay(o.form, o.title, styles, contentW, contentH)
}
