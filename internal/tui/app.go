package tui

import (
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
	tabLogs
	tabCount // sentinel for cycling
)

var tabNames = [tabCount]string{
	"Overview",
	"Backups",
	"Restores",
	"Config",
	"Logs",
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
		cmds := []tea.Cmd{fetchOverviewCmd(m.client)}
		if m.activeTab == tabBackups {
			cmds = append(cmds, fetchBackupsCmd(m.client))
		}
		return m, tea.Batch(cmds...)

	case overviewDataMsg:
		m.data = msg.overviewData
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

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Tab1):
			m.activeTab = tabOverview
		case key.Matches(msg, m.keys.Tab2):
			m.activeTab = tabBackups
		case key.Matches(msg, m.keys.Tab3):
			m.activeTab = tabRestores
		case key.Matches(msg, m.keys.Tab4):
			m.activeTab = tabConfig
		case key.Matches(msg, m.keys.Tab5):
			m.activeTab = tabLogs
		case key.Matches(msg, m.keys.NextTab):
			m.activeTab = (m.activeTab + 1) % tabCount
		case key.Matches(msg, m.keys.PrevTab):
			m.activeTab = (m.activeTab - 1 + tabCount) % tabCount
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
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	header := m.headerView()
	statusBar := m.statusBarView()
	helpBar := m.helpBarView()

	// Calculate remaining height for content.
	chromeHeight := lipgloss.Height(header) +
		lipgloss.Height(statusBar) +
		lipgloss.Height(helpBar)
	contentHeight := m.height - chromeHeight
	if contentHeight < 0 {
		contentHeight = 0
	}

	content := m.contentView(contentHeight)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		content,
		statusBar,
		helpBar,
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
	case tabLogs:
		return m.placeholderContent("Logs - streaming PBM log entries", height)
	}
	return ""
}

// overviewContentView renders the Overview tab with left/right panels.
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

	// Right side: two panels stacked vertically.
	// Top detail panel gets 70%, bottom status panel gets 30%.
	topHeight := (height * 70 / 100)
	bottomHeight := height - topHeight
	innerTopHeight := topHeight - 2       // border
	innerBottomHeight := bottomHeight - 2 // border
	innerLeftHeight := height - 2         // border

	// Clamp to zero.
	if innerLeftWidth < 0 {
		innerLeftWidth = 0
	}
	if innerRightWidth < 0 {
		innerRightWidth = 0
	}
	if innerLeftHeight < 0 {
		innerLeftHeight = 0
	}
	if innerTopHeight < 0 {
		innerTopHeight = 0
	}
	if innerBottomHeight < 0 {
		innerBottomHeight = 0
	}

	// Render panel contents.
	leftContent := m.overview.leftView(innerLeftWidth, innerLeftHeight)
	detailContent := m.overview.detailView()
	statusContent := m.overview.statusView()

	// Apply panel styles.
	leftStyle := m.styles.LeftPanel.Width(innerLeftWidth).Height(innerLeftHeight)
	detailStyle := m.styles.RightPanel.Width(innerRightWidth).Height(innerTopHeight)
	statusStyle := m.styles.RightPanel.Width(innerRightWidth).Height(innerBottomHeight)

	// Focus highlighting.
	if m.overview.focus == panelLeft {
		leftStyle = leftStyle.BorderForeground(m.styles.FocusedBorderColor)
	}
	if m.overview.focus == panelRight {
		detailStyle = detailStyle.BorderForeground(m.styles.FocusedBorderColor)
	}

	left := leftStyle.Render(leftContent)
	detail := detailStyle.Render(detailContent)
	status := statusStyle.Render(statusContent)

	rightColumn := lipgloss.JoinVertical(lipgloss.Left, detail, status)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, rightColumn)
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

// statusBarView renders the bottom status bar with live cluster info.
func (m Model) statusBarView() string {
	if m.flashErr != "" {
		return m.styles.StatusBar.Width(m.width).
			Render("  " + m.styles.StatusError.Render(m.flashErr))
	}

	pitr := m.pitrStatusText()
	op := m.runningOpText()
	cluster := m.clusterTimeText()

	bar := fmt.Sprintf("  %s  |  %s  |  %s", pitr, op, cluster)
	return m.styles.StatusBar.Width(m.width).Render(bar)
}

// pitrStatusText returns a short PITR status string for the status bar.
func (m Model) pitrStatusText() string {
	if m.data.pitr == nil {
		return "PITR: --"
	}
	if !m.data.pitr.Enabled {
		return "PITR: off"
	}
	if m.data.pitr.Running {
		return "PITR: on"
	}
	return "PITR: enabled (not running)"
}

// runningOpText returns a short running operation string for the status bar.
func (m Model) runningOpText() string {
	if len(m.data.operations) == 0 {
		return "Op: none"
	}
	op := m.data.operations[0]
	text := fmt.Sprintf("Op: %s", op.Type)
	if len(m.data.operations) > 1 {
		text += fmt.Sprintf(" (+%d)", len(m.data.operations)-1)
	}
	return text
}

// clusterTimeText returns the cluster time for the status bar.
func (m Model) clusterTimeText() string {
	if m.data.clusterTime.IsZero() {
		return "Cluster: --"
	}
	return fmt.Sprintf("Cluster: %s",
		m.data.clusterTime.Time().Format("15:04:05"))
}

// helpBarView renders the keybinding help at the bottom.
func (m Model) helpBarView() string {
	bindings := m.keys.ShortHelp()
	if m.activeTab == tabBackups {
		bindings = append(bindings, backupKeys.Start, backupKeys.Cancel, backupKeys.Delete)
	}
	return m.styles.HelpBar.Width(m.width).Render(m.help.ShortHelpView(bindings))
}
