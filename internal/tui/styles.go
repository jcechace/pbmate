package tui

import "github.com/charmbracelet/lipgloss"

// Colors — adaptive for light/dark terminals.
var (
	colorPrimary   = lipgloss.AdaptiveColor{Light: "62", Dark: "62"}
	colorSubtle    = lipgloss.AdaptiveColor{Light: "245", Dark: "241"}
	colorHighlight = lipgloss.AdaptiveColor{Light: "236", Dark: "252"}

	ColorOK      = lipgloss.AdaptiveColor{Light: "34", Dark: "42"}
	ColorError   = lipgloss.AdaptiveColor{Light: "160", Dark: "196"}
	ColorWarning = lipgloss.AdaptiveColor{Light: "172", Dark: "214"}
	ColorMuted   = lipgloss.AdaptiveColor{Light: "245", Dark: "241"}
)

// Tab bar styles.
var (
	tabStyle = lipgloss.NewStyle().
			Padding(0, 2)

	activeTabStyle = tabStyle.
			Bold(true).
			Foreground(colorPrimary)

	inactiveTabStyle = tabStyle.
				Foreground(colorSubtle)
)

// Header renders the top bar with app name and tabs.
var headerStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderBottom(true).
	BorderForeground(colorSubtle)

// Panel styles for the master-detail layout.
var (
	PanelBorder = lipgloss.RoundedBorder()

	LeftPanelStyle = lipgloss.NewStyle().
			BorderStyle(PanelBorder).
			BorderForeground(colorSubtle).
			Padding(0, 1)

	RightPanelStyle = lipgloss.NewStyle().
			BorderStyle(PanelBorder).
			BorderForeground(colorSubtle).
			Padding(0, 1)

	FocusedBorderColor = colorPrimary
)

// Status bar at the bottom.
var statusBarStyle = lipgloss.NewStyle().
	Foreground(colorHighlight).
	Background(lipgloss.AdaptiveColor{Light: "254", Dark: "235"}).
	Padding(0, 1)

// Help bar at the very bottom.
var helpBarStyle = lipgloss.NewStyle().
	Foreground(colorSubtle).
	Padding(0, 1)

// Status indicator styles.
var (
	StatusOK      = lipgloss.NewStyle().Foreground(ColorOK)
	StatusError   = lipgloss.NewStyle().Foreground(ColorError)
	StatusWarning = lipgloss.NewStyle().Foreground(ColorWarning)
	StatusMuted   = lipgloss.NewStyle().Foreground(ColorMuted)
)
