package tui

import (
	"bytes"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m Model) handleActionResult(msg actionResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		return m, m.setActionFlash(msg.err)
	}
	m.setActionFlash(nil)
	// Action-specific side effects on success.
	switch msg.action {
	case "restore":
		m.backups.mode = listRestores
		m.backups.rebuildListContent()
		m.backups.rebuildDetailContent()
	case "apply config", "set profile", "create profile", "remove profile", "edit config":
		// Clear cached profile YAMLs so they are re-fetched.
		m.config.profileYAMLs = make(map[string][]byte)
		return m, tea.Batch(tickCmd(0), fetchConfigCmd(m.ctx, m.client))
	default:
		// Match "edit profile <name>" actions.
		if strings.HasPrefix(msg.action, "edit profile ") {
			m.config.profileYAMLs = make(map[string][]byte)
			return m, tea.Batch(tickCmd(0), fetchConfigCmd(m.ctx, m.client))
		}
	}
	return m, tickCmd(0)
}

func (m Model) handleEditorDone(msg editorDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		// Pre-editor errors (temp file creation, editor exit) — no
		// temp file to preserve (already cleaned in openEditorCmd).
		return m, m.setActionFlash(msg.err)
	}
	if bytes.Equal(msg.original, msg.edited) {
		// No changes — clean up temp file.
		_ = os.Remove(msg.tmpPath)
		m.setActionFlash(nil)
		return m, nil
	}
	// Apply the edited config. applyEditedConfigCmd handles temp
	// file cleanup: deletes on success, preserves on failure.
	return m, applyEditedConfigCmd(m.ctx, m.client, msg.edited, msg.profileName, msg.tmpPath)
}

func (m Model) handlePhysicalRestoreResult(msg physicalRestoreResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		return m, m.setActionFlash(msg.err)
	}
	m.exitMessage = "Physical restore dispatched. Monitor progress with: pbm status"
	m.overview.stopFollow()
	return m, tea.Quit
}
