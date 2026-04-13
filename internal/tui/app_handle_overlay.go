package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

func (m Model) handleBulkDeleteRequest(msg bulkDeleteRequest) (tea.Model, tea.Cmd) {
	if m.client != nil {
		return m, fetchBulkDeleteProfilesCmd(m.ctx, m.client, msg.initial)
	}
	return m, nil
}

func (m Model) handleBulkDeleteFormReady(msg bulkDeleteFormReadyMsg) (tea.Model, tea.Cmd) {
	overlay, cmd := newBulkDeleteOverlay(m.ctx, m.client, m.formTheme(), msg.profiles, msg.initial)
	m.activeOverlay = overlay
	return m, cmd
}

func (m Model) handleBackupFormReady(msg backupFormReadyMsg) (tea.Model, tea.Cmd) {
	overlay, cmd := newBackupFormOverlay(m.ctx, m.client, m.formTheme(), msg.kind, msg.profiles, msg.backups)
	m.activeOverlay = overlay
	return m, cmd
}

func (m Model) handleResyncFormRequest(msg resyncFormRequest) (tea.Model, tea.Cmd) {
	overlay, cmd := newResyncFormOverlay(m.ctx, m.client, m.formTheme(), msg.profiles, msg.initial)
	m.activeOverlay = overlay
	return m, cmd
}

func (m Model) handleSetConfigRequest(msg setConfigRequest) (tea.Model, tea.Cmd) {
	if m.client == nil {
		return m, nil
	}
	overlay, cmd := newSetConfigOverlay(m.ctx, m.client, m.formTheme(), msg.profiles, msg.mainExists, msg.initial)
	m.activeOverlay = overlay
	return m, cmd
}

func (m Model) handleRemoveProfileRequest(msg removeProfileRequest) (tea.Model, tea.Cmd) {
	if m.client != nil {
		title := fmt.Sprintf("Delete Profile: %s", msg.name)
		description := fmt.Sprintf("Remove storage profile %q?\nThis will clear associated backup metadata.", msg.name)
		overlay, cmd := newConfirmOverlay(m.formTheme(), title, description, "Delete", "Cancel",
			removeProfileCmd(m.ctx, m.client, msg.name))
		m.activeOverlay = overlay
		return m, cmd
	}
	return m, nil
}

func (m Model) handleEditConfigRequest(msg editConfigRequest) (tea.Model, tea.Cmd) {
	if m.client != nil {
		return m, fetchEditYAMLCmd(m.ctx, m.client, msg.profileName)
	}
	return m, nil
}

func (m Model) handleEditConfigReady(msg editConfigReadyMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		return m, m.setActionFlash(msg.err)
	}
	return m, openEditorCmd(m.editor, msg.yaml, msg.profileName)
}

func (m Model) handleDeleteCheckRequest(msg deleteCheckRequest) (tea.Model, tea.Cmd) {
	if m.client != nil {
		return m, canDeleteCmd(m.ctx, m.client, msg.baseName, msg.title, msg.description)
	}
	return m, nil
}

func (m Model) handleCanDelete(msg canDeleteMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		return m, m.setActionFlash(msg.err)
	}
	overlay, cmd := newConfirmOverlay(m.formTheme(), msg.title, msg.description, "Delete", "Cancel",
		deleteBackupCmd(m.ctx, m.client, msg.baseName))
	m.activeOverlay = overlay
	return m, cmd
}

func (m Model) handleRestoreTargetRequest(msg restoreTargetRequest) (tea.Model, tea.Cmd) {
	if m.client == nil {
		return m, nil
	}
	overlay, cmd := newRestoreTargetOverlay(m.ctx, m.client, m.formTheme(), msg.backups, msg.timelines)
	m.activeOverlay = overlay
	return m, cmd
}

func (m Model) handleRestoreRequest(msg restoreRequest) (tea.Model, tea.Cmd) {
	if m.client == nil {
		return m, nil
	}
	var overlay *restoreFormOverlay
	var cmd tea.Cmd
	switch msg.mode {
	case restoreModeSnapshot:
		overlay, cmd = newSnapshotRestoreOverlay(m.ctx, m.client, m.formTheme(), msg.backup)
	case restoreModePITR:
		overlay, cmd = newPITRRestoreOverlay(m.ctx, m.client, m.formTheme(), msg.timeline, msg.backups, msg.timelines)
	}
	m.activeOverlay = overlay
	return m, cmd
}

func (m Model) handlePhysicalRestoreConfirmRequest(msg physicalRestoreConfirmRequest) (tea.Model, tea.Cmd) {
	if m.client == nil {
		return m, nil
	}
	desc := physicalRestoreWarning(msg)
	overlay, cmd := newConfirmOverlay(m.formTheme(),
		"Physical Restore", desc, "Restore", "Cancel",
		startPhysicalRestoreCmd(m.ctx, m.client, msg.cmd))
	m.activeOverlay = overlay
	return m, cmd
}
