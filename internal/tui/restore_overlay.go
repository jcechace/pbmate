package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// =============================================================================
// Restore target overlay (Step 1 — what to restore)
// =============================================================================

// restoreTargetOverlay wraps the restore target form (Step 1 of the wizard).
// On completion it transitions to a restoreFormOverlay (Step 2) for the
// selected backup or PITR target.
type restoreTargetOverlay struct {
	form        *huh.Form
	result      *restoreTargetResult
	lastType    restoreMode // tracks type for dynamic rebuild
	lastProfile string      // tracks profileName for dynamic rebuild (snapshot mode)
	lastPreset  string      // tracks pitrPreset for dynamic rebuild (pitr mode)
	backups     []sdk.Backup
	timelines   []sdk.Timeline
	formTheme   *huh.Theme
	ctx         context.Context
	client      *sdk.Client
}

func newRestoreTargetOverlay(ctx context.Context, client *sdk.Client, formTheme *huh.Theme, backups []sdk.Backup, timelines []sdk.Timeline) (*restoreTargetOverlay, tea.Cmd) {
	form, result := newRestoreTargetForm(formTheme, backups, timelines, nil)
	o := &restoreTargetOverlay{
		form:        form,
		result:      result,
		lastType:    result.restoreType,
		lastProfile: result.profileName,
		lastPreset:  result.pitrPreset,
		backups:     backups,
		timelines:   timelines,
		formTheme:   formTheme,
		ctx:         ctx,
		client:      client,
	}
	return o, o.form.Init()
}

func (o *restoreTargetOverlay) Update(msg tea.Msg, back, quit key.Binding) (formOverlay, tea.Cmd) {
	if dismissOverlay(msg, back, quit) {
		return nil, nil
	}

	cmd := updateFormModel(&o.form, msg)

	if o.form.State == huh.StateCompleted {
		if !o.result.confirmed {
			return nil, nil
		}
		return o.transitionToOptions()
	}

	if o.form.State == huh.StateAborted {
		return nil, nil
	}

	// Rebuild when type, profile (snapshot), or PITR preset changes.
	typeChanged := o.result.restoreType != o.lastType
	profileChanged := o.result.restoreType == restoreModeSnapshot && o.result.profileName != o.lastProfile
	presetChanged := o.result.restoreType == restoreModePITR && o.result.pitrPreset != o.lastPreset
	if typeChanged || profileChanged || presetChanged {
		return o.rebuildForm(!typeChanged && (profileChanged || presetChanged))
	}

	return o, cmd
}

// transitionToOptions creates the appropriate Step 2 overlay based on
// the selected restore type and target.
func (o *restoreTargetOverlay) transitionToOptions() (formOverlay, tea.Cmd) {
	switch o.result.restoreType {
	case restoreModeSnapshot:
		bk := o.findBackup(o.result.backupName)
		if bk == nil {
			return nil, nil // should not happen
		}
		return newSnapshotRestoreOverlay(o.ctx, o.client, o.formTheme, bk)

	case restoreModePITR:
		timeline := latestTimeline(o.timelines)
		if timeline == nil {
			return nil, nil // should not happen
		}
		// Pre-populate the PITR target from Step 1 selections.
		initial := &restoreFormResult{
			scope:      restoreScopeFull,
			pitrPreset: o.result.pitrPreset,
			pitrTarget: o.result.pitrTarget,
			confirmed:  true,
		}
		return newPITRRestoreOverlayWithInitial(o.ctx, o.client, o.formTheme, timeline, o.backups, initial)

	default:
		return nil, nil
	}
}

// findBackup looks up a backup by name from the cached backup list.
func (o *restoreTargetOverlay) findBackup(name string) *sdk.Backup {
	for i := range o.backups {
		if o.backups[i].Name == name {
			return &o.backups[i]
		}
	}
	return nil
}

// rebuildForm reconstructs the restore target form when type or PITR preset
// changes, preserving current field values. When presetOnly is true, focus
// is advanced past Type to the preset selector.
func (o *restoreTargetOverlay) rebuildForm(presetOnly bool) (formOverlay, tea.Cmd) {
	form, result := newRestoreTargetForm(o.formTheme, o.backups, o.timelines, o.result)
	o.form = form
	o.result = result
	o.lastType = result.restoreType
	o.lastProfile = result.profileName
	o.lastPreset = result.pitrPreset
	return o, initFormWithAdvance(o.form, presetOnly)
}

func (o *restoreTargetOverlay) View(styles *Styles, contentW, contentH int) string {
	return renderFormOverlay(o.form, "Restore", styles, contentW, contentH)
}

// =============================================================================
// Restore options overlay (Step 2 — how to restore)
// =============================================================================

// restoreFormOverlay wraps the restore form. The mode determines whether
// this is a snapshot restore (from a selected backup) or a PITR restore
// (from a selected timeline, with auto-selected base backup).
type restoreFormOverlay struct {
	form       *huh.Form
	result     *restoreFormResult
	mode       restoreMode
	backupName string        // set for snapshot mode
	backup     *sdk.Backup   // set for snapshot mode (for rebuild + incremental check)
	backups    []sdk.Backup  // set for PITR mode (for base backup auto-selection)
	timeline   *sdk.Timeline // set for PITR mode (for rebuild)
	lastScope  string        // tracks scope for dynamic rebuild
	lastPreset string        // tracks pitrPreset for dynamic rebuild (PITR mode)
	formTheme  *huh.Theme
	ctx        context.Context
	client     *sdk.Client
}

func newSnapshotRestoreOverlay(ctx context.Context, client *sdk.Client, formTheme *huh.Theme, bk *sdk.Backup) (*restoreFormOverlay, tea.Cmd) {
	form, result := newSnapshotRestoreForm(formTheme, bk, nil)
	o := &restoreFormOverlay{
		form:       form,
		result:     result,
		mode:       restoreModeSnapshot,
		backupName: bk.Name,
		backup:     bk,
		lastScope:  result.scope,
		formTheme:  formTheme,
		ctx:        ctx,
		client:     client,
	}
	return o, o.form.Init()
}

func newPITRRestoreOverlay(ctx context.Context, client *sdk.Client, formTheme *huh.Theme, timeline *sdk.Timeline, backups []sdk.Backup) (*restoreFormOverlay, tea.Cmd) {
	return newPITRRestoreOverlayWithInitial(ctx, client, formTheme, timeline, backups, nil)
}

func newPITRRestoreOverlayWithInitial(ctx context.Context, client *sdk.Client, formTheme *huh.Theme, timeline *sdk.Timeline, backups []sdk.Backup, initial *restoreFormResult) (*restoreFormOverlay, tea.Cmd) {
	form, result := newPITRRestoreForm(formTheme, timeline, initial)
	o := &restoreFormOverlay{
		form:       form,
		result:     result,
		mode:       restoreModePITR,
		backups:    backups,
		timeline:   timeline,
		lastScope:  result.scope,
		lastPreset: result.pitrPreset,
		formTheme:  formTheme,
		ctx:        ctx,
		client:     client,
	}
	return o, o.form.Init()
}

func (o *restoreFormOverlay) Update(msg tea.Msg, back, quit key.Binding) (formOverlay, tea.Cmd) {
	if dismissOverlay(msg, back, quit) {
		return nil, nil
	}

	cmd := updateFormModel(&o.form, msg)

	if o.form.State == huh.StateCompleted {
		if !o.result.confirmed {
			return nil, nil
		}
		return nil, o.dispatchRestore()
	}

	if o.form.State == huh.StateAborted {
		return nil, nil
	}

	// Rebuild form when tracked values change so only relevant fields are shown.
	switch o.mode {
	case restoreModeSnapshot:
		if o.result.scope != o.lastScope {
			return o.rebuildSnapshotForm()
		}
	case restoreModePITR:
		scopeChanged := o.result.scope != o.lastScope
		presetChanged := o.result.pitrPreset != o.lastPreset
		if scopeChanged || presetChanged {
			return o.rebuildPITRForm(scopeChanged)
		}
	}

	return o, cmd
}

// rebuildSnapshotForm reconstructs the snapshot restore form when scope
// changes, preserving current field values. This swaps conditional groups
// (namespaces + users/roles for selective vs nothing for full).
func (o *restoreFormOverlay) rebuildSnapshotForm() (formOverlay, tea.Cmd) {
	form, result := newSnapshotRestoreForm(o.formTheme, o.backup, o.result)
	o.form = form
	o.result = result
	o.lastScope = result.scope
	return o, o.form.Init()
}

// rebuildPITRForm reconstructs the PITR restore form when scope or preset
// changes, preserving current field values. This swaps conditional groups
// (custom target for "Custom..." preset, namespaces + users/roles for selective).
// When scopeChanged is true, focus is advanced past "Restore to" to the Scope
// field; otherwise Init naturally focuses "Restore to" (the first interactive
// field after the skipped Note).
func (o *restoreFormOverlay) rebuildPITRForm(scopeChanged bool) (formOverlay, tea.Cmd) {
	form, result := newPITRRestoreForm(o.formTheme, o.timeline, o.result)
	o.form = form
	o.result = result
	o.lastScope = result.scope
	o.lastPreset = result.pitrPreset
	return o, initFormWithAdvance(o.form, scopeChanged)
}

// dispatchRestore builds the SDK command and dispatches it. For PITR mode
// this includes auto-selecting the base backup from the cached backup list.
//
// When the target backup (or PITR base) is physical or incremental, the
// dispatch is deferred: a physicalRestoreConfirmRequest is emitted instead,
// prompting the root model to show a final warning before dispatching.
func (o *restoreFormOverlay) dispatchRestore() tea.Cmd {
	switch o.mode {
	case restoreModeSnapshot:
		cmd := o.result.toSnapshotCommand(o.backupName)
		if o.backup != nil && (o.backup.IsPhysical() || o.backup.IsIncremental()) {
			return physicalRestoreConfirmCmd(cmd, o.backup.Name, o.backup.Type.String(), false)
		}
		return startRestoreCmd(o.ctx, o.client, cmd)

	case restoreModePITR:
		target, err := parsePITRTarget(o.result.effectivePITRTarget())
		if err != nil {
			return restoreErrorCmd(err)
		}
		baseName, err := findBaseBackup(target, o.backups)
		if err != nil {
			return restoreErrorCmd(err)
		}
		pitrCmd, err := o.result.toPITRCommand(baseName)
		if err != nil {
			return restoreErrorCmd(err)
		}
		// Check if the auto-selected base backup is physical/incremental.
		if base := findBackupByName(o.backups, baseName); base != nil && (base.IsPhysical() || base.IsIncremental()) {
			return physicalRestoreConfirmCmd(pitrCmd, base.Name, base.Type.String(), true)
		}
		return startRestoreCmd(o.ctx, o.client, pitrCmd)

	default:
		panic("unreachable: unknown restoreMode")
	}
}

// findBackupByName looks up a backup by name from a slice.
func findBackupByName(backups []sdk.Backup, name string) *sdk.Backup {
	for i := range backups {
		if backups[i].Name == name {
			return &backups[i]
		}
	}
	return nil
}

// physicalRestoreConfirmCmd emits a physicalRestoreConfirmRequest message.
func physicalRestoreConfirmCmd(cmd sdk.StartRestoreCommand, backupName, backupType string, isPITR bool) tea.Cmd {
	return func() tea.Msg {
		return physicalRestoreConfirmRequest{
			cmd:        cmd,
			backupName: backupName,
			backupType: backupType,
			isPITR:     isPITR,
		}
	}
}

func (o *restoreFormOverlay) View(styles *Styles, contentW, contentH int) string {
	title := "Restore"
	if o.mode == restoreModePITR {
		title = "PITR Restore"
	}
	return renderFormOverlay(o.form, title, styles, contentW, contentH)
}

// restoreErrorCmd wraps an error as an actionResultMsg command.
func restoreErrorCmd(err error) tea.Cmd {
	return func() tea.Msg {
		return actionResultMsg{action: "restore", err: err}
	}
}

// physicalRestoreWarning builds the warning description shown in the
// confirmation overlay before dispatching a physical/incremental restore.
func physicalRestoreWarning(req physicalRestoreConfirmRequest) string {
	if req.isPITR {
		return fmt.Sprintf(
			"The base backup for this PITR restore is %s.\n"+
				"This will shut down mongod on all nodes\n"+
				"in the cluster.\n\n"+
				"The TUI will exit after dispatching the command.\n"+
				"Monitor progress with: pbm status\n\n"+
				"Base: %s (%s)",
			req.backupType, req.backupName, req.backupType)
	}

	return fmt.Sprintf(
		"This is a %s restore that will shut down mongod\n"+
			"on all nodes in the cluster.\n\n"+
			"The TUI will exit after dispatching the command.\n"+
			"Monitor progress with: pbm status\n\n"+
			"Base: %s (%s)",
		req.backupType, req.backupName, req.backupType)
}
