package tui

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
