package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
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
	help help.Model
}

// New creates a new root model with the given theme.
func New(client *sdk.Client, theme Theme) Model {
	h := help.New()
	h.ShortSeparator = "  "
	s := NewStyles(theme)
	return Model{
		client:       client,
		styles:       s,
		activeTab:    tabOverview,
		pollInterval: idleInterval,
		overview:     newOverviewModel(&s),
		backups:      newBackupsModel(client, &s),
		keys:         globalKeys,
		help:         h,
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
		m.help.Width = msg.Width
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
		m.overview.setData(m.data)
		m.overview.logFollowing = m.logFollowing
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
		m.data.logEntries = append(m.data.logEntries, msg.entry)
		// Keep a reasonable buffer (last 200 entries).
		if len(m.data.logEntries) > 200 {
			m.data.logEntries = m.data.logEntries[len(m.data.logEntries)-200:]
		}
		m.overview.setLogEntries(m.data.logEntries)
		// Read next entry from the follow channel.
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

	content := m.contentView(contentHeight)

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

// contentView renders the active tab's content.
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
	}
	return ""
}

// overviewContentView renders the Overview tab with 4-quadrant layout:
// top-left (Cluster), top-right (Detail), bottom-left (Status), bottom-right (Logs).
func (m Model) overviewContentView(height int) string {
	leftWidth := m.width * 30 / 100
	if leftWidth < 28 {
		leftWidth = 28
	}
	rightWidth := m.width - leftWidth

	// Account for panel border + padding (2 border + 2 padding = 4 per panel).
	const panelChrome = 4
	innerLeftWidth := leftWidth - panelChrome
	innerRightWidth := rightWidth - panelChrome

	// Vertical split: top row ~60%, bottom row ~40%.
	topHeight := height * 60 / 100
	bottomHeight := height - topHeight

	innerTopLeftHeight := topHeight - 2    // border
	innerTopRightHeight := topHeight - 2   // border
	innerBotLeftHeight := bottomHeight - 2 // border
	innerBotRightHeight := bottomHeight - 2

	// Clamp to zero.
	if innerLeftWidth < 0 {
		innerLeftWidth = 0
	}
	if innerRightWidth < 0 {
		innerRightWidth = 0
	}
	if innerTopLeftHeight < 0 {
		innerTopLeftHeight = 0
	}
	if innerTopRightHeight < 0 {
		innerTopRightHeight = 0
	}
	if innerBotLeftHeight < 0 {
		innerBotLeftHeight = 0
	}
	if innerBotRightHeight < 0 {
		innerBotRightHeight = 0
	}

	// Render panel contents.
	clusterContent := m.overview.clusterView(innerLeftWidth, innerTopLeftHeight)
	detailContent := m.overview.detailView()
	statusContent := m.overview.statusView()
	logsContent := m.overview.logsView(innerBotRightHeight)

	// Apply panel styles with titled borders.
	clusterStyle := m.styles.LeftPanel.Width(innerLeftWidth).Height(innerTopLeftHeight)
	detailStyle := m.styles.RightPanel.Width(innerRightWidth).Height(innerTopRightHeight)
	statusStyle := m.styles.LeftPanel.Width(innerLeftWidth).Height(innerBotLeftHeight)
	logsStyle := m.styles.RightPanel.Width(innerRightWidth).Height(innerBotRightHeight)

	// Focus highlighting on the cluster panel.
	if m.overview.focus == panelLeft {
		clusterStyle = clusterStyle.BorderForeground(m.styles.FocusedBorderColor)
	}
	if m.overview.focus == panelRight {
		detailStyle = detailStyle.BorderForeground(m.styles.FocusedBorderColor)
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
	leftWidth := m.width * 30 / 100
	if leftWidth < 28 {
		leftWidth = 28
	}
	rightWidth := m.width - leftWidth

	const panelChrome = 4
	innerLeftWidth := leftWidth - panelChrome
	innerRightWidth := rightWidth - panelChrome
	innerHeight := height - 2

	if innerLeftWidth < 0 {
		innerLeftWidth = 0
	}
	if innerRightWidth < 0 {
		innerRightWidth = 0
	}
	if innerHeight < 0 {
		innerHeight = 0
	}

	leftContent := m.backups.leftView(innerLeftWidth, innerHeight)
	rightContent := m.backups.detailView()

	leftStyle := m.styles.LeftPanel.Width(innerLeftWidth).Height(innerHeight)
	rightStyle := m.styles.RightPanel.Width(innerRightWidth).Height(innerHeight)

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
		statusParts = append(statusParts, m.pitrStatusText())
		statusParts = append(statusParts, m.runningOpText())
		statusParts = append(statusParts, m.clusterTimeText())
	}
	leftZone := strings.Join(statusParts, "  ")

	// Right zone: context-sensitive keybinding hints.
	bindings := m.contextBindings()
	rightZone := m.help.ShortHelpView(bindings)

	// Compose: status left, hints right, separated by a divider.
	divider := m.styles.StatusMuted.Render(" │ ")
	leftWidth := lipgloss.Width(leftZone)
	rightWidth := lipgloss.Width(rightZone)
	dividerWidth := lipgloss.Width(divider)
	gap := m.width - leftWidth - rightWidth - dividerWidth - 1 // 1 for left padding
	if gap < 0 {
		gap = 0
	}

	bar := " " + leftZone + strings.Repeat(" ", gap) + divider + rightZone

	return m.styles.BottomBar.Width(m.width).Render(bar)
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
	// Always include navigation essentials.
	bindings := []key.Binding{m.keys.Up, m.keys.Down}

	switch m.activeTab {
	case tabOverview:
		bindings = append(bindings, overviewKeys.Toggle, overviewKeys.Follow)
		bindings = append(bindings, backupKeys.Start)
	case tabBackups:
		bindings = append(bindings, backupKeys.Start, backupKeys.Cancel, backupKeys.Delete)
	}

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
		return m, nil
	}

	// Start following.
	ctx, cancel := context.WithCancel(context.Background())
	entries, _ := m.client.Logs.Follow(ctx)
	m.logFollowing = true
	m.logFollowCancel = cancel
	m.logFollowCh = entries
	m.overview.logFollowing = true

	return m, waitForLogEntry(entries)
}
