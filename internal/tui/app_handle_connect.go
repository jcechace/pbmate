package tui

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

func (m Model) handleConnect(msg connectMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.connectAttempt++
		delay := connectBackoff(m.connectAttempt)
		m.flashErr = fmt.Sprintf("Connection failed (retry in %s)", delay.Truncate(time.Second))
		return m, reconnectCmd(delay)
	}
	m.connecting = false
	m.connectAttempt = 0
	m.flashErr = ""
	m.flashFromAction = false
	m.client = msg.client
	m.overview.ctx = m.ctx
	m.overview.client = msg.client
	return m, tickCmd(0)
}

func (m Model) handleReconnect(_ reconnectMsg) (tea.Model, tea.Cmd) {
	m.flashErr = ""
	m.flashFromAction = false
	return m, connectCmd(m.mongoURI)
}

func (m Model) handleSpinnerTick(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	// Keep spinning while connecting or running operations.
	if m.connecting || m.overview.HasRunningOps() {
		m.overview.rebuildStatusContent(m.spinner.View())
		return m, cmd
	}
	return m, nil // stop the tick chain
}
