package tui

import "github.com/charmbracelet/lipgloss"

// Styles holds all lipgloss styles derived from a Theme.
type Styles struct {
	// Tab bar.
	ActiveTab   lipgloss.Style
	InactiveTab lipgloss.Style
	Header      lipgloss.Style

	// Panel layout.
	PanelBorder        lipgloss.Border
	LeftPanel          lipgloss.Style
	RightPanel         lipgloss.Style
	FocusedBorderColor lipgloss.TerminalColor

	// Bottom bar (merged status + help).
	BottomBar lipgloss.Style

	// Status indicator styles.
	StatusOK      lipgloss.Style
	StatusError   lipgloss.Style
	StatusWarning lipgloss.Style
	StatusMuted   lipgloss.Style
}

// NewStyles creates a Styles set from the given Theme.
func NewStyles(t Theme) Styles {
	tab := lipgloss.NewStyle().
		Padding(0, 2)

	return Styles{
		ActiveTab: tab.
			Bold(true).
			Foreground(t.Primary),
		InactiveTab: tab.
			Foreground(t.Subtle),
		Header: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(t.Subtle),

		PanelBorder: lipgloss.RoundedBorder(),
		LeftPanel: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(t.Subtle).
			Padding(0, 1),
		RightPanel: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(t.Subtle).
			Padding(0, 1),
		FocusedBorderColor: t.Primary,

		BottomBar: lipgloss.NewStyle().
			Foreground(t.Highlight).
			Background(t.StatusBarBg),

		StatusOK:      lipgloss.NewStyle().Foreground(t.OK),
		StatusError:   lipgloss.NewStyle().Foreground(t.Error),
		StatusWarning: lipgloss.NewStyle().Foreground(t.Warning),
		StatusMuted:   lipgloss.NewStyle().Foreground(t.Muted),
	}
}
