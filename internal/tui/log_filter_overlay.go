package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// logFilterOverlay wraps the log filter form.
type logFilterOverlay struct {
	form   *huh.Form
	result *logFilterFormResult
}

// logFilterRequest is emitted by the overview sub-model when the user
// presses l. The agents and current filter are used to populate the form.
type logFilterRequest struct {
	agents []sdk.Agent
	filter sdk.LogFilter
}

// logFilterResultMsg carries the result of the log filter form back to
// the root model, which updates the overview's logFilter and triggers
// a re-fetch.
type logFilterResultMsg struct {
	filter sdk.LogFilter
	reset  bool // true when user chose "Reset"
}

func newLogFilterOverlay(formTheme huh.Theme, agents []sdk.Agent, current sdk.LogFilter) (*logFilterOverlay, tea.Cmd) {
	initial := fromLogFilter(current)
	form, result := newLogFilterForm(formTheme, agents, initial)
	o := &logFilterOverlay{form: form, result: result}
	return o, initThemedForm(o.form)
}

func (o *logFilterOverlay) Update(msg tea.Msg, back, quit key.Binding) (formOverlay, tea.Cmd) {
	if dismissOverlay(msg, back, quit) {
		return nil, nil
	}

	cmd := updateFormModel(&o.form, msg)

	if o.form.State == huh.StateCompleted {
		if o.result.confirmed {
			// Apply — convert form result to LogFilter.
			f := o.result.toLogFilter()
			return nil, func() tea.Msg {
				return logFilterResultMsg{filter: f}
			}
		}
		// Reset — clear all filters.
		return nil, func() tea.Msg {
			return logFilterResultMsg{reset: true}
		}
	}

	if o.form.State == huh.StateAborted {
		return nil, nil
	}

	return o, cmd
}

func (o *logFilterOverlay) View(styles *Styles, contentW, contentH int) string {
	return renderFormOverlay(o.form, "Log Filter", styles, contentW, contentH)
}
