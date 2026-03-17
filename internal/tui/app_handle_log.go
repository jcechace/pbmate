package tui

import (
	"context"
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

func (m Model) handleLogFollow(msg logFollowMsg) (tea.Model, tea.Cmd) {
	// Discard messages from a stale follow session.
	if msg.session != m.overview.logFollowSession {
		return m, nil
	}
	if msg.err != nil && !errors.Is(msg.err, context.Canceled) {
		// Follow channel errored; stop following.
		m.overview.stopFollow()
		m.flashErr = fmt.Sprintf("follow: %v", msg.err)
		return m, nil
	}
	m.overview.appendLogEntries(msg.entries)
	// Wait for the next batch from the follow channel.
	return m, m.overview.nextLogCmd()
}

func (m Model) handleLogFollowDone(msg logFollowDoneMsg) (tea.Model, tea.Cmd) {
	// Discard done messages from a stale follow session.
	if msg.session != m.overview.logFollowSession {
		return m, nil
	}
	m.overview.stopFollow()
	if msg.err != nil && !errors.Is(msg.err, context.Canceled) {
		m.flashErr = fmt.Sprintf("follow: %v", msg.err)
	}
	return m, nil
}

func (m Model) handleLogFilterRequest(msg logFilterRequest) (tea.Model, tea.Cmd) {
	overlay, cmd := newLogFilterOverlay(m.styles.FormTheme, msg.agents, msg.filter)
	m.activeOverlay = overlay
	return m, cmd
}

func (m Model) handleLogFilterResult(msg logFilterResultMsg) (tea.Model, tea.Cmd) {
	if msg.reset {
		m.overview.logFilter = sdk.LogFilter{}
	} else {
		m.overview.logFilter = msg.filter
	}
	// If following, restart follow with new filter.
	if m.overview.isFollowing() {
		m.overview.stopFollow()
		cmd := m.overview.toggleFollow()
		return m, tea.Batch(cmd, tickCmd(0))
	}
	return m, tickCmd(0)
}
