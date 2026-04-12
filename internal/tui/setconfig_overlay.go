package tui

import (
	"context"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// setConfigOverlay wraps the set-config form. On completion it transitions
// to a filePickerOverlay for the selected target. When overriding an existing
// config/profile the file picker will additionally transition to a confirm.
type setConfigOverlay struct {
	form        *huh.Form
	result      *setConfigFormResult
	lastTarget  setConfigTarget // tracks target for dynamic rebuild
	lastProfile string          // tracks profile for dynamic rebuild
	profiles    []sdk.StorageProfile
	mainExists  bool
	formTheme   huh.Theme
	ctx         context.Context
	client      *sdk.Client
}

func newSetConfigOverlay(ctx context.Context, client *sdk.Client, formTheme huh.Theme, profiles []sdk.StorageProfile, mainExists bool, initial *setConfigFormResult) (*setConfigOverlay, tea.Cmd) {
	form, result := newSetConfigForm(formTheme, profiles, initial)
	o := &setConfigOverlay{
		form:        form,
		result:      result,
		lastTarget:  result.target,
		lastProfile: result.profile,
		profiles:    profiles,
		mainExists:  mainExists,
		formTheme:   formTheme,
		ctx:         ctx,
		client:      client,
	}
	return o, o.form.Init()
}

func (o *setConfigOverlay) Update(msg tea.Msg, back, quit key.Binding) (formOverlay, tea.Cmd) {
	if dismissOverlay(msg, back, quit) {
		return nil, nil
	}

	cmd := updateFormModel(&o.form, msg)

	if o.form.State == huh.StateCompleted {
		if !o.result.confirmed {
			return nil, nil
		}
		return o.transitionToFilePicker()
	}

	if o.form.State == huh.StateAborted {
		return nil, nil
	}

	// Rebuild when target or profile selection changes.
	targetChanged := o.result.target != o.lastTarget
	profileChanged := o.result.target == setConfigTargetProfile && o.result.profile != o.lastProfile
	if targetChanged || profileChanged {
		return o.rebuildForm(profileChanged && !targetChanged)
	}

	return o, cmd
}

// transitionToFilePicker creates a filePickerOverlay for the selected target.
// Sets needsConfirm when overriding an existing config or profile.
func (o *setConfigOverlay) transitionToFilePicker() (formOverlay, tea.Cmd) {
	profile := o.result.effectiveProfile()
	isNew := o.result.isNew()

	var title string
	if profile == "" {
		title = "Select YAML \u2500 Main"
	} else {
		title = "Select YAML \u2500 " + profile
	}

	// Confirm when overriding: main config exists, or profile exists (not new).
	needsConfirm := false
	if o.result.target == setConfigTargetMain && o.mainExists {
		needsConfirm = true
	} else if o.result.target == setConfigTargetProfile && !isNew {
		needsConfirm = true
	}

	fp, fpCmd := newFilePickerOverlay(o.ctx, o.client, profile, isNew, needsConfirm, o.formTheme, title)
	return fp, fpCmd
}

// rebuildForm reconstructs the set-config form when target or profile changes,
// preserving current field values. When profileOnly is true, focus is advanced
// past Target to the Profile selector.
func (o *setConfigOverlay) rebuildForm(profileOnly bool) (formOverlay, tea.Cmd) {
	form, result := newSetConfigForm(o.formTheme, o.profiles, o.result)
	o.form = form
	o.result = result
	o.lastTarget = result.target
	o.lastProfile = result.profile
	return o, initFormWithAdvance(o.form, profileOnly)
}

func (o *setConfigOverlay) View(styles *Styles, contentW, contentH int) string {
	return renderFormOverlay(o.form, "Set Config", styles, contentW, contentH)
}
