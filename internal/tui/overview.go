package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// Display truncation limits.
const (
	maxBackupNameOverview = 16 // max backup name length in the overview status panel
	maxAgentVersionLen    = 5  // max agent version length in the cluster panel
	statusLabelWidth      = 10 // fixed label column width in the status panel
)

// itemKind identifies the type of item in the overview list.
type itemKind int

const (
	itemRSHeader itemKind = iota
	itemAgent
)

// overviewItem is a single row in the overview left panel.
type overviewItem struct {
	kind       itemKind
	agent      *sdk.Agent // set when kind == itemAgent
	rsName     string     // set for itemRSHeader and itemAgent
	selectable bool       // whether the cursor can land here
}

// panel identifies which panel has focus in a two-panel layout.
type panel int

const (
	panelLeft panel = iota
	panelRight
	panelCount // sentinel for cycling
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
	items     []overviewItem
	cursor    int
	focus     overviewFocus
	styles    *Styles
	data      overviewData
	collapsed map[string]bool        // RS name -> collapsed state
	grouped   map[string][]sdk.Agent // RS name -> agents (for collapsed indicators)
	rsNames   []string               // sorted RS names

	// Panel viewports — each produces exactly its allocated height.
	clusterVP viewport.Model
	detailVP  viewport.Model
	statusVP  viewport.Model
	logs      logPanel
}

// newOverviewModel creates a new overview sub-model.
func newOverviewModel(styles *Styles) overviewModel {
	return overviewModel{
		styles:    styles,
		focus:     focusCluster,
		collapsed: make(map[string]bool),
		clusterVP: newPanelViewport(),
		detailVP:  newPanelViewport(),
		statusVP:  newPanelViewport(),
		logs:      newLogPanel(styles),
	}
}

// setData rebuilds the item list from fresh overview data.
func (m *overviewModel) setData(d overviewData) {
	m.data = d
	m.grouped = groupAgentsByRS(d.agents)
	m.rsNames = sortedKeys(m.grouped)
	m.rebuildItems()
	m.rebuildStatusContent()
	m.logs.setEntries(d.logEntries)
}

// rebuildItems reconstructs the flat item list from grouped data,
// respecting collapsed state.
func (m *overviewModel) rebuildItems() {
	// Remember currently selected item identity for cursor stability.
	var selectedNode string
	var selectedRS string
	if sel := m.selectedItem(); sel != nil {
		switch sel.kind {
		case itemAgent:
			selectedNode = sel.agent.Node
		case itemRSHeader:
			selectedRS = sel.rsName
		}
	}

	var items []overviewItem
	for _, rs := range m.rsNames {
		items = append(items, overviewItem{
			kind:       itemRSHeader,
			rsName:     rs,
			selectable: true,
		})
		if !m.collapsed[rs] {
			for i := range m.grouped[rs] {
				a := &m.grouped[rs][i]
				items = append(items, overviewItem{
					kind:       itemAgent,
					agent:      a,
					rsName:     rs,
					selectable: true,
				})
			}
		}
	}

	m.items = items

	// Restore cursor to the same item if possible.
	m.cursor = 0
	if selectedNode != "" {
		for i, item := range m.items {
			if item.kind == itemAgent && item.agent.Node == selectedNode {
				m.cursor = i
				return
			}
		}
	}
	if selectedRS != "" {
		for i, item := range m.items {
			if item.kind == itemRSHeader && item.rsName == selectedRS {
				m.cursor = i
				return
			}
		}
	}
	m.ensureSelectable(1)
	m.rebuildClusterContent()
	m.rebuildDetailContent()
}

// update handles key messages for the overview tab.
func (m *overviewModel) update(msg tea.KeyMsg, keys globalKeyMap) {
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
		m.toggleCollapse()
	case key.Matches(msg, overviewKeys.Wrap):
		m.logs.toggleWrap()
	}
}

// cyclePanel moves focus to the next or previous panel in Z-order
// (Cluster → Detail → Status → Log).
func (m *overviewModel) cyclePanel(delta int) {
	old := m.focus
	m.focus = overviewFocus((int(m.focus) + delta + int(overviewFocusCount)) % int(overviewFocusCount))
	if m.focus != old {
		m.rebuildClusterContent() // update cursor ▶ visibility
	}
}

// handleVertical dispatches Up/Down to the focused panel.
func (m *overviewModel) handleVertical(delta int) {
	switch m.focus {
	case focusCluster:
		if delta > 0 {
			m.moveCursor(1)
		} else {
			m.moveCursor(-1)
		}
		m.rebuildClusterContent()
		m.rebuildDetailContent()
	case focusDetail:
		m.scrollDetail(delta)
	case focusLog:
		m.logs.scroll(delta)
	case focusStatus:
		// Status panel has few static lines; scrolling is not useful.
	}
}

// toggleCollapse expands or collapses the RS group under the cursor.
func (m *overviewModel) toggleCollapse() {
	sel := m.selectedItem()
	if sel == nil {
		return
	}
	var rs string
	switch sel.kind {
	case itemRSHeader:
		rs = sel.rsName
	case itemAgent:
		rs = sel.rsName
	default:
		return
	}
	m.collapsed[rs] = !m.collapsed[rs]
	m.rebuildItems()
}

// moveCursor moves the cursor by delta, skipping non-selectable items.
func (m *overviewModel) moveCursor(delta int) {
	if len(m.items) == 0 {
		return
	}
	start := m.cursor
	for {
		m.cursor += delta
		if m.cursor < 0 {
			m.cursor = 0
			return
		}
		if m.cursor >= len(m.items) {
			m.cursor = len(m.items) - 1
			return
		}
		if m.items[m.cursor].selectable {
			return
		}
		// If we've wrapped around without finding a selectable, stop.
		if m.cursor == start {
			return
		}
	}
}

// ensureSelectable moves cursor to the nearest selectable item in the given
// direction. Used after rebuilding the item list.
func (m *overviewModel) ensureSelectable(dir int) {
	if len(m.items) == 0 {
		return
	}
	if m.cursor < len(m.items) && m.items[m.cursor].selectable {
		return
	}
	m.moveCursor(dir)
}

// scrollDetail scrolls the detail viewport by delta lines.
func (m *overviewModel) scrollDetail(delta int) {
	if delta > 0 {
		m.detailVP.ScrollDown(delta)
	} else {
		m.detailVP.ScrollUp(-delta)
	}
}

// selectedItem returns the currently selected item, or nil if none.
func (m *overviewModel) selectedItem() *overviewItem {
	if m.cursor >= 0 && m.cursor < len(m.items) && m.items[m.cursor].selectable {
		return &m.items[m.cursor]
	}
	return nil
}

// detailContent builds the detail panel content string for the selected item.
func (m *overviewModel) detailContent() string {
	sel := m.selectedItem()
	if sel == nil {
		return m.styles.StatusMuted.Render("No selection")
	}
	var b strings.Builder
	switch sel.kind {
	case itemAgent:
		m.renderAgentDetail(&b, sel.agent)
	case itemRSHeader:
		m.renderRSDetail(&b, sel.rsName)
	}
	return b.String()
}

// renderRSDetail writes replica set summary detail to the builder.
func (m *overviewModel) renderRSDetail(b *strings.Builder, rsName string) {
	b.WriteString(m.styles.SectionHeader.Render("Replica Set"))
	b.WriteByte('\n')

	agents := m.grouped[rsName]
	fmt.Fprintf(b, "  Name:    %s\n", rsName)
	fmt.Fprintf(b, "  Agents:  %d\n", len(agents))

	var ok, stale, errCount int
	for i := range agents {
		a := &agents[i]
		switch {
		case a.Stale:
			stale++
		case !a.OK || len(a.Errors) > 0:
			errCount++
		default:
			ok++
		}
	}
	fmt.Fprintf(b, "  Healthy: %s\n", m.styles.StatusOK.Render(fmt.Sprintf("%d", ok)))
	if stale > 0 {
		fmt.Fprintf(b, "  Stale:   %s\n", m.styles.StatusMuted.Render(fmt.Sprintf("%d", stale)))
	}
	if errCount > 0 {
		fmt.Fprintf(b, "  Error:   %s\n", m.styles.StatusError.Render(fmt.Sprintf("%d", errCount)))
	}
	b.WriteByte('\n')

	// List agents in this RS.
	b.WriteString(m.styles.Bold.Render("  Agents"))
	b.WriteByte('\n')
	for i := range agents {
		a := &agents[i]
		ind := agentIndicator(a, m.styles)
		fmt.Fprintf(b, "  %s %s  %s  %s\n", ind, a.Node, a.Role, a.Version)
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

// --- Viewport content rebuilders ---
// Each rebuilds the content string and sets it on the viewport.
// Called during Update when data or selection changes.

func (m *overviewModel) rebuildClusterContent() {
	m.clusterVP.SetContent(m.clusterContent())
}

func (m *overviewModel) rebuildDetailContent() {
	m.detailVP.SetContent(m.detailContent())
}

func (m *overviewModel) rebuildStatusContent() {
	m.statusVP.SetContent(m.statusContent())
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
	m.clusterVP.Width = contentLeftW
	m.clusterVP.Height = innerTopH
	m.detailVP.Width = contentRightW
	m.detailVP.Height = innerTopH
	m.statusVP.Width = contentLeftW
	m.statusVP.Height = innerBotH
	m.logs.vp.Width = contentRightW
	m.logs.vp.Height = innerBotH
	if m.logs.pinned {
		m.logs.vp.GotoBottom()
	}

	// Apply panel styles with focus-highlighted border.
	clusterStyle := m.styles.LeftPanel.Width(panelLeftW).Height(innerTopH)
	detailStyle := m.styles.RightPanel.Width(panelRightW).Height(innerTopH)
	statusStyle := m.styles.LeftPanel.Width(panelLeftW).Height(innerBotH)
	logsStyle := m.styles.RightPanel.Width(panelRightW).Height(innerBotH)

	switch m.focus {
	case focusCluster:
		clusterStyle = clusterStyle.BorderForeground(m.styles.FocusedBorderColor)
	case focusDetail:
		detailStyle = detailStyle.BorderForeground(m.styles.FocusedBorderColor)
	case focusStatus:
		statusStyle = statusStyle.BorderForeground(m.styles.FocusedBorderColor)
	case focusLog:
		logsStyle = logsStyle.BorderForeground(m.styles.FocusedBorderColor)
	}

	topRow := lipgloss.JoinHorizontal(lipgloss.Top,
		clusterStyle.Render(m.clusterVP.View()),
		detailStyle.Render(m.detailVP.View()),
	)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top,
		statusStyle.Render(m.statusVP.View()),
		logsStyle.Render(m.logs.view()),
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

	m.clusterVP.Width = contentLeftW
	m.clusterVP.Height = innerHeight(topH)
	m.detailVP.Width = contentRightW
	m.detailVP.Height = innerHeight(topH)
	m.statusVP.Width = contentLeftW
	m.statusVP.Height = innerHeight(bottomH)
	m.logs.vp.Width = contentRightW
	m.logs.vp.Height = innerHeight(bottomH)
}

// setLogEntries updates the log entries displayed in the log panel.
func (m *overviewModel) setLogEntries(entries []sdk.LogEntry) {
	m.data.logEntries = entries
	m.logs.setEntries(entries)
}

// renderAgentDetail writes agent detail to the builder.
func (m *overviewModel) renderAgentDetail(b *strings.Builder, a *sdk.Agent) {
	b.WriteString(m.styles.SectionHeader.Render("Agent"))
	b.WriteByte('\n')

	fmt.Fprintf(b, "  Node:        %s\n", a.Node)
	fmt.Fprintf(b, "  Replica Set: %s\n", a.ReplicaSet)
	fmt.Fprintf(b, "  Role:        %s\n", a.Role)
	fmt.Fprintf(b, "  Version:     %s\n", a.Version)

	status := m.styles.StatusOK.Render("OK")
	if a.Stale {
		status = m.styles.StatusMuted.Render("Stale")
	} else if !a.OK {
		status = m.styles.StatusError.Render("Error")
	}
	fmt.Fprintf(b, "  Status:      %s\n", status)

	if len(a.Errors) > 0 {
		b.WriteByte('\n')
		b.WriteString(m.styles.StatusError.Render("  Errors:"))
		b.WriteByte('\n')
		for _, e := range a.Errors {
			fmt.Fprintf(b, "    - %s\n", e)
		}
	}
	b.WriteByte('\n')
}

// clusterContent builds the left panel content string (cluster agents).
func (m *overviewModel) clusterContent() string {
	cursor := lipgloss.NewStyle().Foreground(m.styles.FocusedBorderColor)

	var b strings.Builder
	for i, item := range m.items {
		if i > 0 {
			b.WriteByte('\n')
		}
		line := m.renderItem(item)
		if i == m.cursor && item.selectable {
			if m.focus == focusCluster {
				line = cursor.Render("▶ ") + m.styles.Bold.Render(line)
			} else {
				line = "  " + m.styles.Bold.Render(line)
			}
		} else {
			line = "  " + line
		}
		b.WriteString(line)
	}
	return b.String()
}

// renderItem renders a single item line for the left panel.
func (m *overviewModel) renderItem(item overviewItem) string {
	switch item.kind {
	case itemRSHeader:
		rsStyle := m.styles.SectionHeader
		if m.collapsed[item.rsName] {
			// Collapsed: show ▸ with inline agent status dots and count.
			agents := m.grouped[item.rsName]
			dots := m.agentDots(agents)
			return fmt.Sprintf("%s %s (%d)", rsStyle.Render("▸ "+item.rsName), dots, len(agents))
		}
		return rsStyle.Render("▾ " + item.rsName)

	case itemAgent:
		a := item.agent
		indicator := agentIndicator(a, m.styles)
		role := a.Role.String()
		ver := a.Version
		if len(ver) > maxAgentVersionLen {
			ver = ver[:maxAgentVersionLen]
		}
		return fmt.Sprintf("  %s %s  %s  %s", indicator, a.Node, role, ver)
	}
	return ""
}

// agentDots returns a string of status indicator dots for a slice of agents.
func (m *overviewModel) agentDots(agents []sdk.Agent) string {
	var b strings.Builder
	for i := range agents {
		b.WriteString(agentIndicator(&agents[i], m.styles))
	}
	return b.String()
}

// groupAgentsByRS groups agents by their replica set name.
func groupAgentsByRS(agents []sdk.Agent) map[string][]sdk.Agent {
	m := make(map[string][]sdk.Agent)
	for _, a := range agents {
		m[a.ReplicaSet] = append(m[a.ReplicaSet], a)
	}
	return m
}

// sortedKeys returns map keys sorted alphabetically.
func sortedKeys(m map[string][]sdk.Agent) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
