package tui

import "github.com/charmbracelet/lipgloss"

// Layout constants controlling panel geometry.
const (
	leftPanelPct  = 30 // left panel width as percentage of terminal width
	minLeftPanelW = 28 // minimum left panel width in characters
	topPanelPct   = 60 // top row height as percentage of content area

	panelBorderH  = 2 // horizontal border: left + right
	panelPaddingH = 2 // horizontal padding: left + right (from Padding(0,1))
	panelBorderV  = 2 // vertical border: top + bottom
)

// panel identifies which panel has focus in a two-panel layout.
type panel int

const (
	panelLeft panel = iota
	panelRight
	panelCount // sentinel for cycling
)

// horizontalSplit computes left/right panel and content widths from the total
// terminal width. Panel widths are for lipgloss .Width() (area inside borders).
// Content widths are for viewports (area inside borders AND padding).
func horizontalSplit(totalW int) (panelLeftW, panelRightW, contentLeftW, contentRightW int) {
	leftW := max(totalW*leftPanelPct/100, minLeftPanelW)
	rightW := totalW - leftW

	panelLeftW = max(leftW-panelBorderH, 0)
	panelRightW = max(rightW-panelBorderH, 0)
	contentLeftW = max(leftW-panelBorderH-panelPaddingH, 0)
	contentRightW = max(rightW-panelBorderH-panelPaddingH, 0)
	return
}

// innerHeight computes the usable height inside a panel's vertical border.
func innerHeight(panelH int) int {
	return max(panelH-panelBorderV, 0)
}

// panelBorderColor returns the focused or unfocused border color depending on
// whether the panel is focused. Used by all sub-models for consistent styling.
func panelBorderColor(focused bool, styles *Styles) lipgloss.TerminalColor {
	if focused {
		return styles.FocusedBorderColor
	}
	return styles.UnfocusedBorderColor
}
