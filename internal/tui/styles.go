package tui

import (
	"image/color"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// Styles holds all lipgloss styles derived from a Theme.
type Styles struct {
	// Tab bar.
	ActiveTab   lipgloss.Style
	InactiveTab lipgloss.Style
	Header      lipgloss.Style

	// Panel layout.
	PanelBorder          lipgloss.Border
	LeftPanel            lipgloss.Style
	RightPanel           lipgloss.Style
	FocusedBorderColor   color.Color
	UnfocusedBorderColor color.Color

	// Bottom bar (merged status + help).
	BottomBar lipgloss.Style
	HintKey   lipgloss.Style // bold key name in bottom bar hints
	HintDesc  lipgloss.Style // description text in bottom bar hints

	// Text styles.
	SectionHeader lipgloss.Style // bold + primary color, for detail panel headings
	Bold          lipgloss.Style // bold text, for selected items

	// Status indicator styles.
	StatusOK      lipgloss.Style
	StatusError   lipgloss.Style
	StatusWarning lipgloss.Style
	StatusMuted   lipgloss.Style

	// ChromaStyle is the Chroma syntax highlighting style name for YAML rendering.
	ChromaStyle string

	// FormTheme is the huh form theme matching the active TUI theme.
	FormTheme huh.Theme
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
			Padding(1, 1),
		RightPanel: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(t.Subtle).
			Padding(1, 1),
		FocusedBorderColor:   t.Primary,
		UnfocusedBorderColor: t.Subtle,

		BottomBar: lipgloss.NewStyle().
			Foreground(t.Highlight).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(t.Subtle),
		HintKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Highlight),
		HintDesc: lipgloss.NewStyle().
			Foreground(t.Subtle),

		SectionHeader: lipgloss.NewStyle().Bold(true).Foreground(t.Primary),
		Bold:          lipgloss.NewStyle().Bold(true),

		StatusOK:      lipgloss.NewStyle().Foreground(t.OK),
		StatusError:   lipgloss.NewStyle().Foreground(t.Error),
		StatusWarning: lipgloss.NewStyle().Foreground(t.Warning),
		StatusMuted:   lipgloss.NewStyle().Foreground(t.Muted),

		ChromaStyle: t.ChromaStyle,
		FormTheme:   t.HuhTheme(),
	}
}
