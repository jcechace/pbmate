package tui

import (
	"fmt"
	"strings"

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

	activeTab tab
	width     int
	height    int

	keys globalKeyMap
	help help.Model
}

// New creates a new root model with the given theme.
func New(client *sdk.Client, theme Theme) Model {
	h := help.New()
	h.ShortSeparator = "  "
	return Model{
		client:    client,
		styles:    NewStyles(theme),
		activeTab: tabOverview,
		keys:      globalKeys,
		help:      h,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.WindowSize()
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		return m, nil

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
	style := lipgloss.NewStyle().
		Width(m.width).
		Height(height)

	var content string
	switch m.activeTab {
	case tabOverview:
		content = "Overview - cluster agents, recent backups, PITR status"
	case tabBackups:
		content = "Backups - list and manage backups"
	case tabRestores:
		content = "Restores - list restores"
	case tabConfig:
		content = "Config - PBM configuration and profiles"
	case tabLogs:
		content = "Logs - streaming PBM log entries"
	}

	return style.Render(content)
}

// statusBarView renders the bottom status bar.
func (m Model) statusBarView() string {
	left := "PITR: --"
	mid := "Op: none"
	right := "Cluster: --"

	bar := fmt.Sprintf("  %s  |  %s  |  %s", left, mid, right)
	return m.styles.StatusBar.Width(m.width).Render(bar)
}

// helpBarView renders the keybinding help at the bottom.
func (m Model) helpBarView() string {
	return m.styles.HelpBar.Width(m.width).Render(m.help.ShortHelpView(m.keys.ShortHelp()))
}
