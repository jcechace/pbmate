package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// tab identifies which tab is active.
type tab int

const (
	tabOverview tab = iota
	tabBackups
	tabRestores
	tabConfig
	tabCount // sentinel for cycling
)

var tabNames = [tabCount]string{
	"Overview",
	"Backups",
	"Restores",
	"Config",
}

// Layout constants.
const (
	leftPanelPct  = 30  // left panel width as percentage of terminal width
	minLeftPanelW = 28  // minimum left panel width in characters
	topPanelPct   = 60  // top row height as percentage of content area
	maxLogEntries = 200 // max log entries kept in the follow buffer

	panelBorderH  = 2 // horizontal border: left + right
	panelPaddingH = 2 // horizontal padding: left + right (from Padding(0,1))
	panelBorderV  = 2 // vertical border: top + bottom
)

// Model is the root BubbleTea model for PBMate.
type Model struct {
	client *sdk.Client
	styles Styles

	activeTab    tab
	width        int
	height       int
	pollInterval time.Duration
	flashErr     string // transient error message for the status bar

	// Fetched data.
	data    overviewData
	bkpData backupsData

	// Log follow state (reference types survive model copying).
	logFollowing    bool
	logFollowCancel context.CancelFunc
	logFollowCh     <-chan sdk.LogEntry

	// Sub-models.
	overview overviewModel
	backups  backupsModel

	keys globalKeyMap
}

// New creates a new root model with the given theme.
func New(client *sdk.Client, theme Theme) Model {
	s := NewStyles(theme)
	return Model{
		client:       client,
		styles:       s,
		activeTab:    tabOverview,
		pollInterval: idleInterval,
		overview:     newOverviewModel(&s),
		backups:      newBackupsModel(client, &s),
		keys:         globalKeys,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(tea.WindowSize(), tickCmd(0))
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateViewportDims()
		return m, nil

	case tickMsg:
		// Always fetch overview data (needed for status bar).
		// Additionally fetch tab-specific data.
		cmds := []tea.Cmd{fetchOverviewCmd(m.client, m.logFollowing)}
		if m.activeTab == tabBackups {
			cmds = append(cmds, fetchBackupsCmd(m.client))
		}
		return m, tea.Batch(cmds...)

	case overviewDataMsg:
		// Preserve follow-mode log entries; the poll doesn't fetch logs
		// during follow, so msg.logEntries would be nil.
		if m.logFollowing {
			msg.logEntries = m.data.logEntries
		}
		m.data = msg.overviewData
		// Set logFollowing before setData so rebuildLogContent uses the
		// current state for the mode indicator.
		m.overview.logFollowing = m.logFollowing
		m.overview.setData(m.data)
		m.flashErr = "" // clear flash on successful poll
		// Adaptive polling: faster when operations are running.
		if len(m.data.operations) > 0 {
			m.pollInterval = activeInterval
		} else {
			m.pollInterval = idleInterval
		}
		return m, tickCmd(m.pollInterval)

	case backupsDataMsg:
		m.bkpData = msg.backupsData
		m.backups.setData(m.bkpData)
		return m, nil

	case backupActionMsg:
		if msg.err != nil {
			m.flashErr = fmt.Sprintf("%s failed: %v", msg.action, msg.err)
		} else {
			m.flashErr = ""
		}
		// Trigger immediate re-fetch to pick up the change.
		return m, tickCmd(0)

	case logFollowMsg:
		if msg.err != nil {
			// Follow channel errored; stop following.
			m.logFollowing = false
			return m, nil
		}
		m.data.logEntries = append(m.data.logEntries, msg.entries...)
		if len(m.data.logEntries) > maxLogEntries {
			m.data.logEntries = m.data.logEntries[len(m.data.logEntries)-maxLogEntries:]
		}
		m.overview.setLogEntries(m.data.logEntries)
		// Wait for the next batch from the follow channel.
		return m, waitForLogEntry(m.logFollowCh)

	case logFollowDoneMsg:
		m.logFollowing = false
		return m, nil

	case tea.KeyMsg:
		var newTab tab = -1
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Tab1):
			newTab = tabOverview
		case key.Matches(msg, m.keys.Tab2):
			newTab = tabBackups
		case key.Matches(msg, m.keys.Tab3):
			newTab = tabRestores
		case key.Matches(msg, m.keys.Tab4):
			newTab = tabConfig
		case key.Matches(msg, m.keys.NextTab):
			newTab = (m.activeTab + 1) % tabCount
		case key.Matches(msg, m.keys.PrevTab):
			newTab = (m.activeTab - 1 + tabCount) % tabCount
		case key.Matches(msg, overviewKeys.Follow) && m.activeTab == tabOverview:
			return m.toggleLogFollow()
		case key.Matches(msg, backupKeys.Start):
			return m, startBackupCmd(m.client)
		case key.Matches(msg, backupKeys.Cancel):
			return m, cancelBackupCmd(m.client)
		default:
			// Forward to active tab sub-model.
			switch m.activeTab {
			case tabOverview:
				m.overview.update(msg, m.keys)
			case tabBackups:
				if cmd := m.backups.update(msg, m.keys); cmd != nil {
					return m, cmd
				}
			}
		}
		// Handle tab switch with immediate data fetch.
		if newTab >= 0 && newTab != m.activeTab {
			m.activeTab = newTab
			return m, tickCmd(0)
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	header := m.headerView()
	bottomBar := m.bottomBarView()

	// Calculate remaining height for content.
	chromeHeight := lipgloss.Height(header) + lipgloss.Height(bottomBar)
	contentHeight := m.height - chromeHeight
	if contentHeight < 0 {
		contentHeight = 0
	}

	content := lipgloss.NewStyle().
		MaxHeight(contentHeight).
		Render(m.contentView(contentHeight))

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		content,
		bottomBar,
	)
}

// headerView renders the tab bar.
func (m Model) headerView() string {
	var tabs []string
	for i := 0; i < int(tabCount); i++ {
		label := fmt.Sprintf("%d:%s", i+1, tabNames[i])
		if tab(i) == m.activeTab {
			tabs = append(tabs, m.styles.ActiveTab.Render(label))
		} else {
			tabs = append(tabs, m.styles.InactiveTab.Render(label))
		}
	}

	title := lipgloss.NewStyle().Bold(true).Padding(0, 1).Render("PBMate")
	row := lipgloss.JoinHorizontal(lipgloss.Bottom,
		title,
		strings.Join(tabs, ""),
	)

	return m.styles.Header.Width(m.width).Render(row)
}

// contentView renders the active tab's content. Panels use viewports that
// produce their allocated height; MaxHeight is a safety net against overflow.
func (m Model) contentView(height int) string {
	switch m.activeTab {
	case tabOverview:
		return m.overviewContentView(height)
	case tabBackups:
		return m.backupsContentView(height)
	case tabRestores:
		return m.placeholderContent("Restores - list restores", height)
	case tabConfig:
		return m.placeholderContent("Config - PBM configuration and profiles", height)
	default:
		return ""
	}
}

// overviewContentView renders the Overview tab with 4-quadrant layout:
// top-left (Cluster), top-right (Detail), bottom-left (Status), bottom-right (Logs).
func (m Model) overviewContentView(height int) string {
	panelLeftW, panelRightW, contentLeftW, contentRightW := horizontalSplit(m.width)

	topHeight := height * topPanelPct / 100
	bottomHeight := height - topHeight
	innerTopH := innerHeight(topHeight)
	innerBotH := innerHeight(bottomHeight)

	// Set viewport dimensions (known only at View time) and render.
	m.overview.setClusterSize(contentLeftW, innerTopH)
	m.overview.setDetailSize(contentRightW, innerTopH)
	m.overview.setStatusSize(contentLeftW, innerBotH)
	m.overview.setLogSize(contentRightW, innerBotH)

	clusterContent := m.overview.clusterView()
	detailContent := m.overview.detailView()
	statusContent := m.overview.statusView()
	logsContent := m.overview.logsView()

	// Apply panel styles with titled borders.
	clusterStyle := m.styles.LeftPanel.Width(panelLeftW).Height(innerTopH)
	detailStyle := m.styles.RightPanel.Width(panelRightW).Height(innerTopH)
	statusStyle := m.styles.LeftPanel.Width(panelLeftW).Height(innerBotH)
	logsStyle := m.styles.RightPanel.Width(panelRightW).Height(innerBotH)

	// Highlight the focused panel's border.
	switch m.overview.focus {
	case focusCluster:
		clusterStyle = clusterStyle.BorderForeground(m.styles.FocusedBorderColor)
	case focusDetail:
		detailStyle = detailStyle.BorderForeground(m.styles.FocusedBorderColor)
	case focusStatus:
		statusStyle = statusStyle.BorderForeground(m.styles.FocusedBorderColor)
	case focusLog:
		logsStyle = logsStyle.BorderForeground(m.styles.FocusedBorderColor)
	}

	cluster := clusterStyle.Render(clusterContent)
	detail := detailStyle.Render(detailContent)
	status := statusStyle.Render(statusContent)
	logs := logsStyle.Render(logsContent)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, cluster, detail)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, status, logs)

	return lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)
}

// backupsContentView renders the Backups tab with left list + right detail.
func (m Model) backupsContentView(height int) string {
	panelLeftW, panelRightW, contentLeftW, contentRightW := horizontalSplit(m.width)
	innerH := innerHeight(height)

	// Set viewport dimensions (known only at View time) and render.
	m.backups.setListSize(contentLeftW, innerH)
	m.backups.setDetailSize(contentRightW, innerH)

	leftContent := m.backups.listView()
	rightContent := m.backups.detailView()

	leftStyle := m.styles.LeftPanel.Width(panelLeftW).Height(innerH)
	rightStyle := m.styles.RightPanel.Width(panelRightW).Height(innerH)

	if m.backups.focus == panelLeft {
		leftStyle = leftStyle.BorderForeground(m.styles.FocusedBorderColor)
	}
	if m.backups.focus == panelRight {
		rightStyle = rightStyle.BorderForeground(m.styles.FocusedBorderColor)
	}

	left := leftStyle.Render(leftContent)
	right := rightStyle.Render(rightContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

// placeholderContent renders a simple placeholder for unimplemented tabs.
func (m Model) placeholderContent(text string, height int) string {
	return lipgloss.NewStyle().
		Width(m.width).
		Height(height).
		Render(text)
}

// bottomBarView renders the single merged bottom bar with status HUD on the
// left and context-sensitive keybinding hints on the right.
func (m Model) bottomBarView() string {
	// Left zone: operational status HUD.
	var statusParts []string
	if m.flashErr != "" {
		statusParts = append(statusParts, m.styles.StatusError.Render(m.flashErr))
	} else {
		statusParts = append(statusParts, m.clusterTimeText())
		statusParts = append(statusParts, m.pitrStatusText())
		statusParts = append(statusParts, m.runningOpText())
	}
	leftZone := " " + strings.Join(statusParts, "  ")

	// Right zone: context-sensitive keybinding hints, truncated to fit.
	bindings := m.contextBindings()
	const hintPadding = 2 // 1 char padding on each side
	availWidth := m.width - lipgloss.Width(leftZone) - hintPadding
	rightZone := m.renderHints(bindings, availWidth) + " "

	// Compose: left-aligned status, gap, right-aligned hints.
	gap := m.width - lipgloss.Width(leftZone) - lipgloss.Width(rightZone)
	if gap < 0 {
		gap = 0
	}
	bar := leftZone + strings.Repeat(" ", gap) + rightZone

	return m.styles.BottomBar.Width(m.width).Render(bar)
}

// renderHints formats keybinding hints for the bottom bar using
// foreground-only styles. Bindings that exceed maxWidth are dropped.
func (m Model) renderHints(bindings []key.Binding, maxWidth int) string {
	const hintSep = "  "
	var parts []string
	totalWidth := 0

	for _, b := range bindings {
		if !b.Enabled() {
			continue
		}
		keys := b.Help().Key
		desc := b.Help().Desc
		if keys == "" || desc == "" {
			continue
		}
		hint := m.styles.HintKey.Render(keys) + " " + m.styles.HintDesc.Render(desc)
		hintWidth := lipgloss.Width(hint)

		// Account for separator before this hint (if not the first).
		sepWidth := 0
		if len(parts) > 0 {
			sepWidth = lipgloss.Width(hintSep)
		}
		if totalWidth+sepWidth+hintWidth > maxWidth {
			break
		}
		totalWidth += sepWidth + hintWidth
		parts = append(parts, hint)
	}
	return strings.Join(parts, hintSep)
}

// pitrStatusText returns a short PITR status string for the status bar.
func (m Model) pitrStatusText() string {
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

// runningOpText returns a short running operation string for the status bar.
func (m Model) runningOpText() string {
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

// contextBindings returns the keybinding hints appropriate for the current
// tab and selection state.
func (m Model) contextBindings() []key.Binding {
	// Navigation: panel cycling + vertical.
	bindings := []key.Binding{m.keys.NextPanel, m.keys.PrevPanel, m.keys.Up, m.keys.Down}

	// Tab-specific hints.
	switch m.activeTab {
	case tabOverview:
		bindings = append(bindings, overviewKeys.Toggle, overviewKeys.Follow, overviewKeys.Wrap)
	case tabBackups:
		bindings = append(bindings, backupKeys.Delete)
	}

	// Global actions + help.
	bindings = append(bindings, backupKeys.Start, backupKeys.Cancel)
	bindings = append(bindings, m.keys.Help, m.keys.Quit)
	return bindings
}

// clusterTimeText returns the cluster time for the status bar.
func (m Model) clusterTimeText() string {
	if m.data.clusterTime.IsZero() {
		return "--:--"
	}
	return m.data.clusterTime.Time().Format("15:04")
}

// updateViewportDims precomputes all viewport dimensions from the current
// terminal size. This allows Update-time operations (scrolling, GotoBottom)
// to use correct bounds, since View-time dimension setting operates on a
// value copy and doesn't persist.
func (m *Model) updateViewportDims() {
	if m.width == 0 || m.height == 0 {
		return
	}
	chromeH := lipgloss.Height(m.headerView()) + lipgloss.Height(m.bottomBarView())
	contentH := max(m.height-chromeH, 0)

	_, _, contentLeftW, contentRightW := horizontalSplit(m.width)

	// Overview: 4-quadrant layout.
	topH := contentH * topPanelPct / 100
	bottomH := contentH - topH

	m.overview.clusterVP.Width = contentLeftW
	m.overview.clusterVP.Height = innerHeight(topH)
	m.overview.detailVP.Width = contentRightW
	m.overview.detailVP.Height = innerHeight(topH)
	m.overview.statusVP.Width = contentLeftW
	m.overview.statusVP.Height = innerHeight(bottomH)
	m.overview.logVP.Width = contentRightW
	m.overview.logVP.Height = innerHeight(bottomH)

	// Backups: 2-panel full-height layout.
	m.backups.listVP.Width = contentLeftW
	m.backups.listVP.Height = innerHeight(contentH)
	m.backups.detailVP.Width = contentRightW
	m.backups.detailVP.Height = innerHeight(contentH)
}

// toggleLogFollow starts or stops the log follow mode.
func (m Model) toggleLogFollow() (tea.Model, tea.Cmd) {
	if m.logFollowing {
		// Stop following.
		if m.logFollowCancel != nil {
			m.logFollowCancel()
		}
		m.logFollowing = false
		m.logFollowCancel = nil
		m.logFollowCh = nil
		m.overview.logFollowing = false
		m.overview.rebuildLogContent()
		return m, nil
	}

	// Start following — pin to bottom so new entries auto-scroll.
	ctx, cancel := context.WithCancel(context.Background())
	entries, _ := m.client.Logs.Follow(ctx)
	m.logFollowing = true
	m.logFollowCancel = cancel
	m.logFollowCh = entries
	m.overview.logFollowing = true
	m.overview.logPinned = true
	m.overview.rebuildLogContent()

	return m, waitForLogEntry(entries)
}
