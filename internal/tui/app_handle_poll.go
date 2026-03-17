package tui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) handleTick(_ tickMsg) (tea.Model, tea.Cmd) {
	if m.client == nil {
		return m, nil
	}
	// Always fetch overview data (needed for status bar).
	// Additionally fetch tab-specific data.
	cmds := []tea.Cmd{fetchOverviewCmd(m.ctx, m.client, m.overview.skipLogFetch(), m.overview.logFilter)}
	if m.activeTab == tabBackups {
		cmds = append(cmds, fetchBackupsCmd(m.ctx, m.client), fetchRestoresCmd(m.ctx, m.client))
	}
	if m.activeTab == tabConfig {
		cmds = append(cmds, fetchConfigCmd(m.ctx, m.client))
	}
	return m, tea.Batch(cmds...)
}

func (m Model) handleOverviewData(msg overviewDataMsg) (tea.Model, tea.Cmd) {
	hadOps := m.overview.HasRunningOps()
	m.overview.setData(msg.overviewData, m.spinner.View())
	m.setFlash("fetch", msg.err)
	// Adaptive polling: faster when operations are running.
	cmds := []tea.Cmd{tickCmd(idleInterval)}
	if m.overview.HasRunningOps() {
		cmds[0] = tickCmd(activeInterval)
		// Restart spinner if operations just appeared.
		if !hadOps && !m.connecting {
			cmds = append(cmds, m.spinner.Tick)
		}
	}
	return m, tea.Batch(cmds...)
}

func (m Model) handleBackupsData(msg backupsDataMsg) (tea.Model, tea.Cmd) {
	m.backups.setBackupData(msg.backupsData)
	m.setFlash("fetch", msg.err)
	return m, nil
}

func (m Model) handleRestoresData(msg restoresDataMsg) (tea.Model, tea.Cmd) {
	m.backups.setRestoreData(msg.restoresData)
	m.setFlash("fetch", msg.err)
	return m, nil
}

func (m Model) handleConfigData(msg configDataMsg) (tea.Model, tea.Cmd) {
	m.config.setData(msg.configData)
	m.setFlash("fetch", msg.err)
	// Trigger lazy profile YAML fetch if the selected profile is uncached.
	if name := m.config.needsProfileYAML(); name != "" {
		return m, fetchProfileYAMLCmd(m.ctx, m.client, name)
	}
	return m, nil
}

func (m Model) handleProfileYAML(msg profileYAMLMsg) (tea.Model, tea.Cmd) {
	m.setFlash("fetch", msg.err)
	if msg.err == nil {
		m.config.setProfileYAML(msg.name, msg.yaml)
	}
	return m, nil
}

func (m Model) handleFetchProfileYAMLRequest(msg fetchProfileYAMLRequest) (tea.Model, tea.Cmd) {
	if m.client != nil {
		return m, fetchProfileYAMLCmd(m.ctx, m.client, msg.name)
	}
	return m, nil
}
