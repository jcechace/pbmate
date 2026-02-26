package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// bulkDeleteOverlay wraps the bulk delete form. It tracks field values
// to trigger form rebuilds when target or preset changes.
type bulkDeleteOverlay struct {
	form       *huh.Form
	result     *bulkDeleteFormResult
	lastTarget bulkDeleteTarget
	lastPreset bulkDeletePreset
	profiles   []sdk.StorageProfile
	formTheme  *huh.Theme
	ctx        context.Context
	client     *sdk.Client
}

// bulkDeleteFormReadyMsg carries fetched profiles so the bulk delete form
// can be created with the profile selector populated. The initial field
// pre-selects the target when opening from a context-sensitive action
// (e.g. d on a PITR timeline).
type bulkDeleteFormReadyMsg struct {
	profiles []sdk.StorageProfile
	initial  *bulkDeleteFormResult // nil for default
}

// fetchBulkDeleteProfilesCmd fetches storage profiles for the bulk delete form.
func fetchBulkDeleteProfilesCmd(ctx context.Context, client *sdk.Client, initial *bulkDeleteFormResult) tea.Cmd {
	return func() tea.Msg {
		profiles, _ := client.Config.ListProfiles(ctx)
		return bulkDeleteFormReadyMsg{profiles: profiles, initial: initial}
	}
}

func newBulkDeleteOverlay(ctx context.Context, client *sdk.Client, formTheme *huh.Theme, profiles []sdk.StorageProfile, initial *bulkDeleteFormResult) (*bulkDeleteOverlay, tea.Cmd) {
	form, result := newBulkDeleteForm(formTheme, profiles, initial)
	o := &bulkDeleteOverlay{
		form:       form,
		result:     result,
		lastTarget: result.target,
		lastPreset: result.preset,
		profiles:   profiles,
		formTheme:  formTheme,
		ctx:        ctx,
		client:     client,
	}
	return o, o.form.Init()
}

func (o *bulkDeleteOverlay) Update(msg tea.Msg, back, quit key.Binding) (formOverlay, tea.Cmd) {
	if dismissOverlay(msg, back, quit) {
		return nil, nil
	}

	cmd := updateFormModel(&o.form, msg)

	if o.form.State == huh.StateCompleted {
		if !o.result.confirmed {
			return nil, nil
		}
		return nil, o.dispatch()
	}

	if o.form.State == huh.StateAborted {
		return nil, nil
	}

	// Rebuild when target or preset changes so the correct fields are shown.
	targetChanged := o.result.target != o.lastTarget
	presetChanged := o.result.preset != o.lastPreset
	if targetChanged || presetChanged {
		return o.rebuildForm()
	}

	return o, cmd
}

// rebuildForm reconstructs the bulk delete form when target or preset changes,
// preserving current field values. Focus is advanced past the Target selector
// so the user doesn't have to re-navigate through unchanged fields.
func (o *bulkDeleteOverlay) rebuildForm() (formOverlay, tea.Cmd) {
	form, result := newBulkDeleteForm(o.formTheme, o.profiles, o.result)
	o.form = form
	o.result = result
	o.lastTarget = result.target
	o.lastPreset = result.preset
	// Advance past Target to Preset (1 field), so focus lands on the
	// field the user was interacting with or just past it.
	return o, initFormWithAdvance(o.form, true)
}

// dispatch creates the appropriate tea.Cmd based on the form result.
func (o *bulkDeleteOverlay) dispatch() tea.Cmd {
	switch o.result.target {
	case bulkDeletePITR:
		cmd, err := o.result.toPITRCommand()
		if err != nil {
			return func() tea.Msg {
				return actionResultMsg{action: "bulk delete", err: err}
			}
		}
		return deletePITRCmd(o.ctx, o.client, cmd)
	default:
		cmd, err := o.result.toBackupCommand()
		if err != nil {
			return func() tea.Msg {
				return actionResultMsg{action: "bulk delete", err: err}
			}
		}
		return deleteBackupsBulkCmd(o.ctx, o.client, cmd)
	}
}

func (o *bulkDeleteOverlay) View(styles *Styles, contentW, contentH int) string {
	return renderFormOverlay(o.form, "Bulk Delete", styles, contentW, contentH)
}
