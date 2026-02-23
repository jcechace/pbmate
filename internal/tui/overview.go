package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

const (
	maxBackupNameOverview = 16  // max backup name length in the overview status panel
	statusLabelWidth      = 10  // fixed label column width in the status panel
	maxLogEntries         = 200 // max log entries in follow buffer; balances memory vs scroll depth
)

// overviewFocus identifies which quadrant has focus in the overview layout.
type overviewFocus int

const (
	focusCluster       overviewFocus = iota // top-left
	focusDetail                             // top-right
	focusStatus                             // bottom-left
	focusLog                                // bottom-right
	overviewFocusCount                      // sentinel for cycling
)

// overviewModel is the sub-model for the Overview tab.
type overviewModel struct {
	ctx    context.Context // root context for SDK calls; set after connect
	client *sdk.Client
	focus  overviewFocus
	styles *Styles
	data   overviewData

	// Log follow state (reference types survive model copying).
	logFollowCancel  context.CancelFunc
	logFollowCtx     context.Context
	logFollowCh      <-chan sdk.LogEntry
	logFollowErrs    <-chan error
	logFollowSession uint64 // monotonic counter to identify the current session

	// Sub-panels.
	cluster  clusterPanel
	statusVP viewport.Model
	logs     logPanel
}

// newOverviewModel creates a new overview sub-model.
func newOverviewModel(styles *Styles) overviewModel {
	cp := newClusterPanel(styles)
	cp.focused = true // cluster panel starts focused
	return overviewModel{
		styles:   styles,
		focus:    focusCluster,
		cluster:  cp,
		statusVP: newPanelViewport(),
		logs:     newLogPanel(styles),
	}
}

// isFollowing reports whether log follow mode is active.
func (m *overviewModel) isFollowing() bool {
	return m.logs.following
}

// HasRunningOps reports whether any operations are currently running.
func (m *overviewModel) HasRunningOps() bool {
	return len(m.data.operations) > 0
}

// PITRStatusText returns a short PITR status string for the status bar.
func (m *overviewModel) PITRStatusText() string {
	if m.data.pitr == nil {
		return "PITR:--"
	}
	if !m.data.pitr.Enabled {
		return "PITR:off"
	}
	if m.data.pitr.Running {
		return "PITR:on"
	}
	return "PITR:paused"
}

// RunningOpText returns a short running operation summary for the status bar.
func (m *overviewModel) RunningOpText() string {
	if len(m.data.operations) == 0 {
		return "Op:none"
	}
	op := m.data.operations[0]
	text := fmt.Sprintf("Op:%s", op.Type)
	if len(m.data.operations) > 1 {
		text += fmt.Sprintf("(+%d)", len(m.data.operations)-1)
	}
	return text
}

// ClusterTimeText returns the cluster time for the status bar.
func (m *overviewModel) ClusterTimeText() string {
	if m.data.clusterTime.IsZero() {
		return "--:--"
	}
	return m.data.clusterTime.Time().UTC().Format("15:04")
}

// toggleFollow starts or stops the log follow mode and returns a command
// to begin listening for log entries. Follow errors arrive asynchronously
// via the error channel and are surfaced through logFollowDoneMsg.
func (m *overviewModel) toggleFollow() tea.Cmd {
	if m.logs.following {
		m.stopFollow()
		return nil
	}

	// Start following — pin to bottom so new entries auto-scroll.
	ctx, cancel := context.WithCancel(m.ctx)
	entries, errs := m.client.Logs.Follow(ctx, sdk.FollowOptions{})
	m.logFollowSession++
	m.logFollowCancel = cancel
	m.logFollowCtx = ctx
	m.logFollowCh = entries
	m.logFollowErrs = errs
	m.logs.setFollowing(true)

	return waitForLogEntry(ctx, m.logFollowSession, entries, errs)
}

// stopFollow cancels the follow goroutine and resets follow state.
// Safe to call when not following.
func (m *overviewModel) stopFollow() {
	if m.logFollowCancel != nil {
		m.logFollowCancel()
	}
	m.logFollowCancel = nil
	m.logFollowCtx = nil
	m.logFollowCh = nil
	m.logFollowErrs = nil
	m.logs.setFollowing(false)
}

// appendLogEntries adds streamed log entries from follow mode, trims to
// maxLogEntries, and updates the log panel.
func (m *overviewModel) appendLogEntries(entries []sdk.LogEntry) {
	m.data.logEntries = append(m.data.logEntries, entries...)
	if len(m.data.logEntries) > maxLogEntries {
		m.data.logEntries = m.data.logEntries[len(m.data.logEntries)-maxLogEntries:]
	}
	m.logs.setEntries(m.data.logEntries)
}

// nextLogCmd returns a command that waits for the next follow log batch.
func (m *overviewModel) nextLogCmd() tea.Cmd {
	return waitForLogEntry(m.logFollowCtx, m.logFollowSession, m.logFollowCh, m.logFollowErrs)
}

// setData rebuilds all panels from fresh overview data.
// During follow mode the poll skips log fetching (logEntries will be nil),
// so we preserve the existing follow-accumulated entries.
func (m *overviewModel) setData(d overviewData) {
	if m.isFollowing() && d.logEntries == nil {
		d.logEntries = m.data.logEntries
	}
	m.data = d
	m.cluster.setAgents(d.agents)
	m.rebuildStatusContent()
	m.logs.setEntries(d.logEntries)
}

// update handles key messages for the overview tab.
// Returns a tea.Cmd if an action was triggered, nil otherwise.
func (m *overviewModel) update(msg tea.KeyMsg, keys globalKeyMap) tea.Cmd {
	switch {
	case key.Matches(msg, keys.NextPanel):
		m.cyclePanel(1)
	case key.Matches(msg, keys.PrevPanel):
		m.cyclePanel(-1)
	case key.Matches(msg, keys.Down):
		m.handleVertical(1)
	case key.Matches(msg, keys.Up):
		m.handleVertical(-1)
	case key.Matches(msg, overviewKeys.Toggle) && m.focus == focusCluster:
		m.cluster.toggleCollapse()
	case key.Matches(msg, overviewKeys.Follow):
		return m.toggleFollow()
	case key.Matches(msg, overviewKeys.Wrap):
		m.logs.toggleWrap()
	}
	return nil
}

// cyclePanel moves focus to the next or previous panel in Z-order
// (Cluster → Detail → Status → Log).
func (m *overviewModel) cyclePanel(delta int) {
	old := m.focus
	m.focus = overviewFocus((int(m.focus) + delta + int(overviewFocusCount)) % int(overviewFocusCount))
	if m.focus != old {
		m.cluster.focused = (m.focus == focusCluster)
		m.cluster.rebuildClusterContent() // update cursor ▶ visibility
	}
}

// handleVertical dispatches Up/Down to the focused panel.
func (m *overviewModel) handleVertical(delta int) {
	switch m.focus {
	case focusCluster:
		m.cluster.moveCursor(delta)
	case focusDetail:
		m.cluster.scrollDetail(delta)
	case focusLog:
		m.logs.scroll(delta)
	case focusStatus:
		// Status panel has few static lines; scrolling is not useful.
	}
}

// statusContent builds the bottom-left status panel content string.
func (m *overviewModel) statusContent() string {
	var b strings.Builder
	label := m.styles.Bold.Width(statusLabelWidth)

	// PITR status.
	pitrVal := m.styles.StatusMuted.Render("--")
	if m.data.pitr != nil {
		switch {
		case !m.data.pitr.Enabled:
			pitrVal = m.styles.StatusMuted.Render("off")
		case m.data.pitr.Running:
			pitrVal = m.styles.StatusOK.Render("on (running)")
		default:
			pitrVal = m.styles.StatusWarning.Render("enabled (paused)")
		}
	}
	fmt.Fprintf(&b, " %s %s\n", label.Render("PITR"), pitrVal)
	if m.data.pitr != nil && m.data.pitr.Error != "" {
		fmt.Fprintf(&b, " %s %s\n", label.Render(""), m.styles.StatusError.Render(m.data.pitr.Error))
	}

	// PITR range (latest timeline).
	if len(m.data.timelines) > 0 {
		latest := m.data.timelines[len(m.data.timelines)-1]
		start := latest.Start.Time().UTC().Format("Jan 02 15:04")
		end := latest.End.Time().UTC().Format("Jan 02 15:04")
		rangeVal := fmt.Sprintf("%s → %s", start, end)
		fmt.Fprintf(&b, " %s %s\n", label.Render(""), rangeVal)
	}

	// Running operation.
	opVal := m.styles.StatusMuted.Render("none")
	if len(m.data.operations) > 0 {
		op := m.data.operations[0]
		opVal = m.styles.StatusWarning.Render(fmt.Sprintf("%s %s", op.Type, m.styles.StatusWarning.Render("●")))
		if len(m.data.operations) > 1 {
			opVal += m.styles.StatusMuted.Render(fmt.Sprintf(" (+%d)", len(m.data.operations)-1))
		}
	}
	fmt.Fprintf(&b, " %s %s\n", label.Render("Op"), opVal)

	// Latest backup.
	latestVal := m.styles.StatusMuted.Render("none")
	if len(m.data.recentBackups) > 0 {
		latest := m.data.recentBackups[0]
		ind := statusIndicator(latest.Status, m.styles)
		name := latest.Name
		if len(name) > maxBackupNameOverview {
			name = name[:maxBackupNameOverview]
		}
		age := ""
		if !latest.StartTS.IsZero() {
			age = " (" + relativeTime(latest.StartTS) + ")"
		}
		latestVal = fmt.Sprintf("%s %s%s", ind, name, m.styles.StatusMuted.Render(age))
	}
	fmt.Fprintf(&b, " %s %s\n", label.Render("Latest"), latestVal)

	// Storage info (will be populated when config data is fetched).
	storageVal := m.styles.StatusMuted.Render("--")
	if m.data.storageName != "" {
		storageVal = m.data.storageName
	}
	fmt.Fprintf(&b, " %s %s\n", label.Render("Storage"), storageVal)

	return b.String()
}

// rebuildStatusContent reconstructs the status viewport content.
func (m *overviewModel) rebuildStatusContent() {
	m.statusVP.SetContent(m.statusContent())
}

// borderColor returns the border color for the given quadrant, highlighting
// the focused panel.
func (m *overviewModel) borderColor(f overviewFocus) lipgloss.TerminalColor {
	if m.focus == f {
		return m.styles.FocusedBorderColor
	}
	return m.styles.UnfocusedBorderColor
}

// view renders the Overview tab with 4-quadrant layout:
// top-left (Cluster), top-right (Detail), bottom-left (Status), bottom-right (Logs).
func (m *overviewModel) view(totalW, totalH int) string {
	panelLeftW, panelRightW, contentLeftW, contentRightW := horizontalSplit(totalW)

	topHeight := totalH * topPanelPct / 100
	bottomHeight := totalH - topHeight
	innerTopH := innerHeight(topHeight)
	innerBotH := innerHeight(bottomHeight)

	// Set viewport dimensions (known only at View time).
	m.cluster.resize(contentLeftW, innerTopH, contentRightW, innerTopH)
	m.statusVP.Width = contentLeftW
	m.statusVP.Height = innerBotH
	m.logs.vp.Width = contentRightW
	m.logs.vp.Height = innerBotH
	if m.logs.pinned {
		m.logs.vp.GotoBottom()
	}

	// Render titled panels with focus-highlighted borders.
	border := m.styles.PanelBorder
	topRow := lipgloss.JoinHorizontal(lipgloss.Top,
		renderTitledPanel("Cluster", m.cluster.clusterView(),
			m.styles.LeftPanel, panelLeftW, innerTopH, border, m.borderColor(focusCluster)),
		renderTitledPanel("Detail", m.cluster.detailView(),
			m.styles.RightPanel, panelRightW, innerTopH, border, m.borderColor(focusDetail)),
	)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top,
		renderTitledPanel("Status", m.statusVP.View(),
			m.styles.LeftPanel, panelLeftW, innerBotH, border, m.borderColor(focusStatus)),
		renderTitledPanel("Logs", m.logs.view(),
			m.styles.RightPanel, panelRightW, innerBotH, border, m.borderColor(focusLog)),
	)

	return lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)
}

// resize precomputes viewport dimensions so Update-time operations (scrolling,
// GotoBottom) use correct bounds. View-time dimension setting operates on a
// value copy and doesn't persist.
func (m *overviewModel) resize(totalW, totalH int) {
	_, _, contentLeftW, contentRightW := horizontalSplit(totalW)

	topH := totalH * topPanelPct / 100
	bottomH := totalH - topH

	m.cluster.resize(contentLeftW, innerHeight(topH), contentRightW, innerHeight(topH))
	m.statusVP.Width = contentLeftW
	m.statusVP.Height = innerHeight(bottomH)
	m.logs.vp.Width = contentRightW
	m.logs.vp.Height = innerHeight(bottomH)
}
