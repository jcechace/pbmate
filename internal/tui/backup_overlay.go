package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// backupFormOverlay wraps the backup start form (quick confirm or full wizard).
type backupFormOverlay struct {
	form           *huh.Form
	result         *backupFormResult
	kind           backupFormKind
	lastBackupType string // tracks backupType for dynamic rebuild
	lastConfigName string // tracks configName for dynamic rebuild
	ctx            context.Context
	client         *sdk.Client
	formTheme      *huh.Theme
}

func newBackupFormOverlay(ctx context.Context, client *sdk.Client, formTheme *huh.Theme, kind backupFormKind, profiles []sdk.StorageProfile, backups []sdk.Backup) (*backupFormOverlay, tea.Cmd) {
	var form *huh.Form
	var result *backupFormResult
	switch kind {
	case backupFormQuick:
		form, result = newQuickBackupForm(formTheme)
	case backupFormFull:
		form, result = newFullBackupForm(formTheme, profiles, backups, nil)
	}
	result.profiles = profiles
	result.backups = backups
	o := &backupFormOverlay{
		form: form, result: result, kind: kind,
		lastBackupType: result.backupType,
		lastConfigName: result.configName,
		ctx:            ctx, client: client, formTheme: formTheme,
	}
	return o, o.form.Init()
}

func (o *backupFormOverlay) Update(msg tea.Msg, back, quit key.Binding) (formOverlay, tea.Cmd) {
	if dismissOverlay(msg, back, quit) {
		return nil, nil
	}
	// 'c' on the quick form transitions to the full wizard.
	if keyMsg, ok := msg.(tea.KeyMsg); ok && o.kind == backupFormQuick && key.Matches(keyMsg, customizeKey) {
		return o.transitionToFull()
	}

	cmd := updateFormModel(&o.form, msg)

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

	// Rebuild the form when backup type or profile changes so that only
	// relevant fields are shown (e.g. namespaces for logical, chain toggle
	// for incremental). Profile changes affect chain existence detection.
	if o.kind == backupFormFull && (o.result.backupType != o.lastBackupType || o.result.configName != o.lastConfigName) {
		return o.rebuildForm()
	}

	return o, cmd
}

func (o *backupFormOverlay) transitionToFull() (formOverlay, tea.Cmd) {
	form, result := newFullBackupForm(o.formTheme, o.result.profiles, o.result.backups, o.result)
	next := &backupFormOverlay{
		form:           form,
		result:         result,
		kind:           backupFormFull,
		lastBackupType: result.backupType,
		lastConfigName: result.configName,
		ctx:            o.ctx,
		client:         o.client,
		formTheme:      o.formTheme,
	}
	return next, next.form.Init()
}

// rebuildForm reconstructs the full backup form when the backup type or
// profile changes, preserving current field values. This swaps conditional
// groups (e.g. namespaces for logical vs chain toggle for incremental).
func (o *backupFormOverlay) rebuildForm() (formOverlay, tea.Cmd) {
	form, result := newFullBackupForm(o.formTheme, o.result.profiles, o.result.backups, o.result)
	o.form = form
	o.result = result
	o.lastBackupType = result.backupType
	o.lastConfigName = result.configName
	return o, o.form.Init()
}

func (o *backupFormOverlay) View(styles *Styles, contentW, contentH int) string {
	title := "Start Backup"
	if o.kind == backupFormFull {
		title = "Configure Backup"
	}
	return renderFormOverlay(o.form, title, styles, contentW, contentH)
}
