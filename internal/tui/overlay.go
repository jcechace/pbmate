package tui

import (
	"context"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// formOverlay is a modal overlay that captures all input while active.
// Update returns the next overlay state: self to continue, nil to dismiss,
// or a different overlay to transition (e.g. profile name → file picker).
type formOverlay interface {
	Update(msg tea.Msg, back, quit key.Binding) (formOverlay, tea.Cmd)
	View(styles *Styles, contentW, contentH int) string
}

// =============================================================================
// Backup form overlay
// =============================================================================

// backupFormOverlay wraps the backup start form (quick confirm or full wizard).
type backupFormOverlay struct {
	form           *huh.Form
	result         *backupFormResult
	kind           backupFormKind
	lastBackupType string // tracks backupType for dynamic rebuild
	ctx            context.Context
	client         *sdk.Client
	formTheme      *huh.Theme
}

func newBackupFormOverlay(ctx context.Context, client *sdk.Client, formTheme *huh.Theme, kind backupFormKind, profiles []sdk.StorageProfile) (*backupFormOverlay, tea.Cmd) {
	var form *huh.Form
	var result *backupFormResult
	switch kind {
	case backupFormQuick:
		form, result = newQuickBackupForm(formTheme)
	case backupFormFull:
		form, result = newFullBackupForm(formTheme, profiles, nil)
	}
	result.profiles = profiles
	o := &backupFormOverlay{form: form, result: result, kind: kind, lastBackupType: result.backupType, ctx: ctx, client: client, formTheme: formTheme}
	return o, o.form.Init()
}

func (o *backupFormOverlay) Update(msg tea.Msg, back, quit key.Binding) (formOverlay, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, back) || key.Matches(keyMsg, quit) {
			return nil, nil
		}
		// 'c' on the quick form transitions to the full wizard.
		if o.kind == backupFormQuick && keyMsg.String() == "c" {
			return o.transitionToFull()
		}
	}

	formModel, cmd := o.form.Update(msg)
	if f, ok := formModel.(*huh.Form); ok {
		o.form = f
	}

	if o.form.State == huh.StateCompleted {
		// Quick form: "Customize" was selected (confirmed == false).
		if o.kind == backupFormQuick && !o.result.confirmed {
			return o.transitionToFull()
		}
		if !o.result.confirmed {
			return nil, nil
		}
		return nil, startBackupCmd(o.ctx, o.client, o.result.toCommand())
	}

	if o.form.State == huh.StateAborted {
		return nil, nil
	}

	// Rebuild the form when backup type changes so that only relevant
	// fields are shown (e.g. namespaces for logical, chain toggle for incremental).
	if o.kind == backupFormFull && o.result.backupType != o.lastBackupType {
		return o.rebuildForm()
	}

	return o, cmd
}

func (o *backupFormOverlay) transitionToFull() (formOverlay, tea.Cmd) {
	form, result := newFullBackupForm(o.formTheme, o.result.profiles, o.result)
	next := &backupFormOverlay{
		form:           form,
		result:         result,
		kind:           backupFormFull,
		lastBackupType: result.backupType,
		ctx:            o.ctx,
		client:         o.client,
		formTheme:      o.formTheme,
	}
	return next, next.form.Init()
}

// rebuildForm reconstructs the full backup form when the backup type changes,
// preserving current field values. This swaps conditional groups (e.g.
// namespaces for logical vs chain toggle for incremental).
func (o *backupFormOverlay) rebuildForm() (formOverlay, tea.Cmd) {
	form, result := newFullBackupForm(o.formTheme, o.result.profiles, o.result)
	o.form = form
	o.result = result
	o.lastBackupType = result.backupType
	return o, o.form.Init()
}

func (o *backupFormOverlay) View(styles *Styles, contentW, contentH int) string {
	title := "Start Backup"
	if o.kind == backupFormFull {
		title = "Configure Backup"
	}
	return renderFormOverlay(o.form, title, styles, contentW, contentH)
}

// =============================================================================
// Restore form overlay
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
	form, result := newPITRRestoreForm(formTheme, timeline, nil)
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
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, back) || key.Matches(keyMsg, quit) {
			return nil, nil
		}
	}

	formModel, cmd := o.form.Update(msg)
	if f, ok := formModel.(*huh.Form); ok {
		o.form = f
	}

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

	initCmd := o.form.Init()
	if scopeChanged {
		// Init focuses "Restore to" (first non-skip field). Advance to Scope.
		advanceCmd := o.form.NextField()
		return o, tea.Batch(initCmd, advanceCmd)
	}
	return o, initCmd
}

// dispatchRestore builds the SDK command and dispatches it. For PITR mode
// this includes auto-selecting the base backup from the cached backup list.
func (o *restoreFormOverlay) dispatchRestore() tea.Cmd {
	switch o.mode {
	case restoreModeSnapshot:
		cmd := o.result.toSnapshotCommand(o.backupName)
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
		return startRestoreCmd(o.ctx, o.client, pitrCmd)

	default:
		panic("unreachable: unknown restoreMode")
	}
}

func (o *restoreFormOverlay) View(styles *Styles, contentW, contentH int) string {
	title := "Restore"
	if o.mode == restoreModePITR {
		title = "PITR Restore"
	}
	return renderFormOverlay(o.form, title, styles, contentW, contentH)
}

// restoreErrorCmd wraps an error as a restoreActionMsg command.
func restoreErrorCmd(err error) tea.Cmd {
	return func() tea.Msg {
		return restoreActionMsg{action: "restore", err: err}
	}
}

// =============================================================================
// Resync form overlay
// =============================================================================

// resyncFormOverlay wraps the resync scope/options form.
type resyncFormOverlay struct {
	form   *huh.Form
	result *resyncFormResult
	ctx    context.Context
	client *sdk.Client
}

func newResyncFormOverlay(ctx context.Context, client *sdk.Client, formTheme *huh.Theme, profiles []sdk.StorageProfile) (*resyncFormOverlay, tea.Cmd) {
	form, result := newResyncForm(formTheme, profiles)
	o := &resyncFormOverlay{form: form, result: result, ctx: ctx, client: client}
	return o, o.form.Init()
}

func (o *resyncFormOverlay) Update(msg tea.Msg, back, quit key.Binding) (formOverlay, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, back) || key.Matches(keyMsg, quit) {
			return nil, nil
		}
	}

	formModel, cmd := o.form.Update(msg)
	if f, ok := formModel.(*huh.Form); ok {
		o.form = f
	}

	if o.form.State == huh.StateCompleted {
		if !o.result.confirmed {
			return nil, nil
		}
		return nil, resyncCmd(o.ctx, o.client, o.result.toCommand())
	}

	if o.form.State == huh.StateAborted {
		return nil, nil
	}

	return o, cmd
}

func (o *resyncFormOverlay) View(styles *Styles, contentW, contentH int) string {
	return renderFormOverlay(o.form, "Resync Storage", styles, contentW, contentH)
}

// =============================================================================
// File picker overlay
// =============================================================================

// filePickerAllowedTypes restricts the file picker to YAML files.
var filePickerAllowedTypes = []string{".yaml", ".yml"}

// filePickerHeight is the number of visible rows in the file picker.
// Fits comfortably in typical 24-row terminals with room for chrome.
const filePickerHeight = 18

// filePickerOverlay wraps a file picker for selecting YAML config files.
type filePickerOverlay struct {
	picker  filepicker.Model
	title   string
	profile string // target profile ("" = main config)
	isNew   bool   // creating new vs overwriting existing
	ctx     context.Context
	client  *sdk.Client
}

func newFilePickerOverlay(ctx context.Context, client *sdk.Client, profile string, isNew bool, title string) (*filePickerOverlay, tea.Cmd) {
	fp := filepicker.New()
	fp.AllowedTypes = filePickerAllowedTypes
	fp.AutoHeight = false
	fp.SetHeight(filePickerHeight)
	fp.ShowHidden = false
	fp.ShowPermissions = false
	fp.ShowSize = true
	fp.DirAllowed = false
	fp.FileAllowed = true

	// Start from an absolute path so Back (filepath.Dir) can navigate up.
	if wd, err := os.Getwd(); err == nil {
		fp.CurrentDirectory = wd
	}

	// Customize keybindings: remove esc from Back (used for dismiss),
	// and use h/backspace/left for parent directory navigation.
	km := filepicker.DefaultKeyMap()
	km.Back = key.NewBinding(
		key.WithKeys("h", "backspace", "left"),
		key.WithHelp("h", "back"),
	)
	fp.KeyMap = km

	o := &filePickerOverlay{
		picker:  fp,
		title:   title,
		profile: profile,
		isNew:   isNew,
		ctx:     ctx,
		client:  client,
	}
	return o, o.picker.Init()
}

func (o *filePickerOverlay) Update(msg tea.Msg, back, quit key.Binding) (formOverlay, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, back) || key.Matches(keyMsg, quit) {
			return nil, nil
		}
	}

	fp, cmd := o.picker.Update(msg)
	o.picker = fp

	if didSelect, path := o.picker.DidSelectFile(msg); didSelect {
		if o.isNew {
			return nil, applyProfileCmd(o.ctx, o.client, o.profile, path, "create profile")
		}
		if o.profile == "" {
			return nil, applyConfigCmd(o.ctx, o.client, path)
		}
		return nil, applyProfileCmd(o.ctx, o.client, o.profile, path, "set profile")
	}

	return o, cmd
}

func (o *filePickerOverlay) View(styles *Styles, contentW, contentH int) string {
	return renderFilePickerOverlay(&o.picker, o.title, styles, contentW, contentH)
}

// =============================================================================
// Profile name overlay (transitions to file picker)
// =============================================================================

// profileNameOverlay wraps the profile name form. On completion it transitions
// to a filePickerOverlay for the newly named profile.
type profileNameOverlay struct {
	form   *huh.Form
	result *profileNameResult
	ctx    context.Context
	client *sdk.Client
}

func newProfileNameOverlay(ctx context.Context, client *sdk.Client, formTheme *huh.Theme) (*profileNameOverlay, tea.Cmd) {
	form, result := newProfileNameForm(formTheme)
	o := &profileNameOverlay{form: form, result: result, ctx: ctx, client: client}
	return o, o.form.Init()
}

func (o *profileNameOverlay) Update(msg tea.Msg, back, quit key.Binding) (formOverlay, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, back) || key.Matches(keyMsg, quit) {
			return nil, nil
		}
	}

	formModel, cmd := o.form.Update(msg)
	if f, ok := formModel.(*huh.Form); ok {
		o.form = f
	}

	if o.form.State == huh.StateCompleted {
		if !o.result.confirmed || o.result.name == "" {
			return nil, nil
		}
		// Transition to file picker for the new profile.
		fp, fpCmd := newFilePickerOverlay(o.ctx, o.client, o.result.name, true, "Select YAML \u2500 "+o.result.name)
		return fp, fpCmd
	}

	if o.form.State == huh.StateAborted {
		return nil, nil
	}

	return o, cmd
}

func (o *profileNameOverlay) View(styles *Styles, contentW, contentH int) string {
	return renderFormOverlay(o.form, "New Profile", styles, contentW, contentH)
}

// =============================================================================
// Confirm overlay
// =============================================================================

// confirmOverlay wraps a confirmation form with a stored action command
// that executes only if the user confirms.
type confirmOverlay struct {
	form      *huh.Form
	result    *confirmFormResult
	title     string
	actionCmd tea.Cmd // executed if confirmed
}

func newConfirmOverlay(formTheme *huh.Theme, title, description, affirmative, negative string, actionCmd tea.Cmd) (*confirmOverlay, tea.Cmd) {
	form, result := newConfirmForm(formTheme, description, affirmative, negative)
	o := &confirmOverlay{form: form, result: result, title: title, actionCmd: actionCmd}
	return o, o.form.Init()
}

func (o *confirmOverlay) Update(msg tea.Msg, back, quit key.Binding) (formOverlay, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, back) || key.Matches(keyMsg, quit) {
			return nil, nil
		}
	}

	formModel, cmd := o.form.Update(msg)
	if f, ok := formModel.(*huh.Form); ok {
		o.form = f
	}

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

// =============================================================================
// File picker rendering
// =============================================================================

// filePickerInnerWidth is the content width inside the file picker panel.
const filePickerInnerWidth = 60

// renderFilePickerOverlay renders the file picker centered over the content
// area inside a bordered panel with a title and current path breadcrumb.
func renderFilePickerOverlay(fp *filepicker.Model, title string, styles *Styles, contentW, contentH int) string {
	// Current directory path — truncate from the left if too wide.
	dir := fp.CurrentDirectory
	maxPathW := filePickerInnerWidth - 2 // leave room for "…/" prefix
	if len(dir) > maxPathW {
		dir = "\u2026" + dir[len(dir)-maxPathW:]
	}
	pathLine := styles.StatusMuted.Render(dir)

	// Hint line for navigation.
	hintLine := styles.StatusMuted.Render("h:back  l:open  enter:select  esc:cancel")

	body := pathLine + "\n" +
		styles.StatusMuted.Render(strings.Repeat("\u2500", filePickerInnerWidth)) + "\n" +
		fp.View() + "\n" +
		styles.StatusMuted.Render(strings.Repeat("\u2500", filePickerInnerWidth)) + "\n" +
		hintLine

	border := lipgloss.RoundedBorder()
	borderColor := styles.FocusedBorderColor

	panelWidth := filePickerInnerWidth + panelPaddingH

	panel := lipgloss.NewStyle().
		Border(border).
		BorderForeground(borderColor).
		Padding(1, 1).
		Width(panelWidth).
		Render(body)

	outerW := panelWidth + panelBorderH
	panel = replaceTitleBorder(panel, title, outerW, border, borderColor)

	return lipgloss.Place(contentW, contentH,
		lipgloss.Center, lipgloss.Center,
		panel)
}
