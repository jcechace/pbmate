package tui

import (
	"context"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// resyncFormOverlay wraps the resync scope/options form.
type resyncFormOverlay struct {
	form        *huh.Form
	result      *resyncFormResult
	lastTarget  resyncScope // tracks scope for dynamic rebuild
	lastProfile string      // tracks profile for confirm title rebuild
	profiles    []sdk.StorageProfile
	formTheme   huh.Theme
	ctx         context.Context
	client      *sdk.Client
}

func newResyncFormOverlay(ctx context.Context, client *sdk.Client, formTheme huh.Theme, profiles []sdk.StorageProfile, initial *resyncFormResult) (*resyncFormOverlay, tea.Cmd) {
	form, result := newResyncForm(formTheme, profiles, initial)
	o := &resyncFormOverlay{
		form:        form,
		result:      result,
		lastTarget:  result.scope,
		lastProfile: result.profileName,
		profiles:    profiles,
		formTheme:   formTheme,
		ctx:         ctx,
		client:      client,
	}
	return o, o.form.Init()
}

func (o *resyncFormOverlay) Update(msg tea.Msg, back, quit key.Binding) (formOverlay, tea.Cmd) {
	if dismissOverlay(msg, back, quit) {
		return nil, nil
	}

	cmd := updateFormModel(&o.form, msg)

	if o.form.State == huh.StateCompleted {
		if !o.result.confirmed {
			return nil, nil
		}
		return nil, resyncCmd(o.ctx, o.client, o.result.toCommand())
	}

	if o.form.State == huh.StateAborted {
		return nil, nil
	}

	// Rebuild when target or profile changes so the correct groups and
	// confirm title are shown.
	targetChanged := o.result.scope != o.lastTarget
	profileChanged := o.result.scope == resyncScopeProfile && o.result.profileName != o.lastProfile
	if targetChanged || profileChanged {
		return o.rebuildForm(profileChanged && !targetChanged)
	}

	return o, cmd
}

// rebuildForm reconstructs the resync form when target or profile changes,
// preserving current field values. When profileOnly is true, focus is advanced
// past Target to the Profile selector; otherwise Init focuses Target.
func (o *resyncFormOverlay) rebuildForm(profileOnly bool) (formOverlay, tea.Cmd) {
	form, result := newResyncForm(o.formTheme, o.profiles, o.result)
	o.form = form
	o.result = result
	o.lastTarget = result.scope
	o.lastProfile = result.profileName
	return o, initFormWithAdvance(o.form, profileOnly)
}

func (o *resyncFormOverlay) View(styles *Styles, contentW, contentH int) string {
	return renderFormOverlay(o.form, "Resync Storage", styles, contentW, contentH)
}
