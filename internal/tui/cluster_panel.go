package tui

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// max agent version length in the cluster panel tree listing.
const maxAgentVersionLen = 8

// itemKind identifies the type of item in the cluster tree.
type itemKind int

const (
	itemRSHeader itemKind = iota
	itemAgent
)

// clusterItem is a single row in the cluster tree panel.
type clusterItem struct {
	kind       itemKind
	agent      *sdk.Agent // set when kind == itemAgent
	rsName     string     // set for itemRSHeader and itemAgent
	selectable bool       // whether the cursor can land here
}

// clusterPanel manages the cluster agent tree (top-left) and its associated
// detail viewport (top-right). Selecting an item in the tree updates the
// detail panel content.
type clusterPanel struct {
	styles    *Styles
	items     []clusterItem
	cursor    int
	focused   bool                   // cluster tree has keyboard focus (shows ▶ cursor)
	collapsed map[string]bool        // RS name -> collapsed state
	grouped   map[string][]sdk.Agent // RS name -> agents
	rsNames   []string               // sorted RS names

	clusterVP viewport.Model
	detailVP  viewport.Model
}

// newClusterPanel creates a cluster panel with empty data.
func newClusterPanel(styles *Styles) clusterPanel {
	return clusterPanel{
		styles:    styles,
		collapsed: make(map[string]bool),
		clusterVP: newPanelViewport(),
		detailVP:  newPanelViewport(),
	}
}

// setAgents replaces agent data, regroups, and rebuilds the item list.
func (p *clusterPanel) setAgents(agents []sdk.Agent) {
	p.grouped = groupAgentsByRS(agents)
	p.rsNames = sortedKeys(p.grouped)
	p.rebuildItems()
}

// selectedItem returns the currently selected item, or nil if none.
func (p *clusterPanel) selectedItem() *clusterItem {
	if p.cursor >= 0 && p.cursor < len(p.items) && p.items[p.cursor].selectable {
		return &p.items[p.cursor]
	}
	return nil
}

// moveCursor moves the cursor by delta, skipping non-selectable items.
// Rebuilds cluster and detail content after moving.
func (p *clusterPanel) moveCursor(delta int) {
	if len(p.items) == 0 {
		return
	}
	start := p.cursor
	for {
		p.cursor += delta
		if p.cursor < 0 {
			p.cursor = 0
			break
		}
		if p.cursor >= len(p.items) {
			p.cursor = len(p.items) - 1
			break
		}
		if p.items[p.cursor].selectable {
			break
		}
		if p.cursor == start {
			break
		}
	}
	p.rebuildClusterContent()
	p.rebuildDetailContent()
}

// toggleCollapse expands or collapses the RS group under the cursor.
func (p *clusterPanel) toggleCollapse() {
	sel := p.selectedItem()
	if sel == nil {
		return
	}
	switch sel.kind {
	case itemRSHeader, itemAgent:
		p.collapsed[sel.rsName] = !p.collapsed[sel.rsName]
		p.rebuildItems()
	}
}

// scrollDetail scrolls the detail viewport by delta lines.
func (p *clusterPanel) scrollDetail(delta int) {
	if delta > 0 {
		p.detailVP.ScrollDown(delta)
	} else {
		p.detailVP.ScrollUp(-delta)
	}
}

// clusterView returns the cluster tree viewport output.
func (p *clusterPanel) clusterView() string {
	return p.clusterVP.View()
}

// detailView returns the detail viewport output.
func (p *clusterPanel) detailView() string {
	return p.detailVP.View()
}

// resize sets viewport dimensions for both cluster and detail viewports.
// Called at Update time (persists for scrolling) and View time (value copy).
func (p *clusterPanel) resize(clusterW, clusterH, detailW, detailH int) {
	p.clusterVP.Width = clusterW
	p.clusterVP.Height = clusterH
	p.detailVP.Width = detailW
	p.detailVP.Height = detailH
}

// --- Internal methods ---

// rebuildItems reconstructs the flat item list from grouped data,
// respecting collapsed state. Preserves cursor position by item identity.
func (p *clusterPanel) rebuildItems() {
	// Remember currently selected item identity for cursor stability.
	var selectedNode string
	var selectedRS string
	if sel := p.selectedItem(); sel != nil {
		switch sel.kind {
		case itemAgent:
			selectedNode = sel.agent.Node
		case itemRSHeader:
			selectedRS = sel.rsName
		}
	}

	var items []clusterItem
	for _, rs := range p.rsNames {
		items = append(items, clusterItem{
			kind:       itemRSHeader,
			rsName:     rs,
			selectable: true,
		})
		if !p.collapsed[rs] {
			for i := range p.grouped[rs] {
				a := &p.grouped[rs][i]
				items = append(items, clusterItem{
					kind:       itemAgent,
					agent:      a,
					rsName:     rs,
					selectable: true,
				})
			}
		}
	}

	p.items = items

	// Restore cursor to the same item if possible.
	p.cursor = 0
	if selectedNode != "" {
		for i, item := range p.items {
			if item.kind == itemAgent && item.agent.Node == selectedNode {
				p.cursor = i
				p.rebuildClusterContent()
				p.rebuildDetailContent()
				return
			}
		}
	}
	if selectedRS != "" {
		for i, item := range p.items {
			if item.kind == itemRSHeader && item.rsName == selectedRS {
				p.cursor = i
				p.rebuildClusterContent()
				p.rebuildDetailContent()
				return
			}
		}
	}
	p.ensureSelectable(1)
	p.rebuildClusterContent()
	p.rebuildDetailContent()
}

// ensureSelectable moves cursor to the nearest selectable item in the given
// direction if the current position is not selectable.
func (p *clusterPanel) ensureSelectable(dir int) {
	if len(p.items) == 0 {
		return
	}
	if p.cursor < len(p.items) && p.items[p.cursor].selectable {
		return
	}
	// Use a bounded search to find the nearest selectable.
	start := p.cursor
	for {
		p.cursor += dir
		if p.cursor < 0 {
			p.cursor = 0
			return
		}
		if p.cursor >= len(p.items) {
			p.cursor = len(p.items) - 1
			return
		}
		if p.items[p.cursor].selectable {
			return
		}
		if p.cursor == start {
			return
		}
	}
}

// rebuildClusterContent reconstructs the cluster tree viewport content.
func (p *clusterPanel) rebuildClusterContent() {
	p.clusterVP.SetContent(p.clusterContent())
}

// rebuildDetailContent reconstructs the detail viewport content.
func (p *clusterPanel) rebuildDetailContent() {
	p.detailVP.SetContent(p.detailContent())
}

// clusterContent builds the cluster tree content string.
func (p *clusterPanel) clusterContent() string {
	cursorStyle := lipgloss.NewStyle().Foreground(p.styles.FocusedBorderColor)

	var b strings.Builder
	for i, item := range p.items {
		if i > 0 {
			b.WriteByte('\n')
		}
		line := p.renderItem(item)
		if i == p.cursor && item.selectable {
			if p.focused {
				line = cursorStyle.Render("▶ ") + p.styles.Bold.Render(line)
			} else {
				line = "  " + p.styles.Bold.Render(line)
			}
		} else {
			line = "  " + line
		}
		b.WriteString(line)
	}
	return b.String()
}

// detailContent builds the detail panel content for the selected item.
func (p *clusterPanel) detailContent() string {
	sel := p.selectedItem()
	if sel == nil {
		return p.styles.StatusMuted.Render("No selection")
	}
	var b strings.Builder
	switch sel.kind {
	case itemAgent:
		p.renderAgentDetail(&b, sel.agent)
	case itemRSHeader:
		p.renderRSDetail(&b, sel.rsName)
	}
	return b.String()
}

// renderItem renders a single item line for the cluster tree.
func (p *clusterPanel) renderItem(item clusterItem) string {
	switch item.kind {
	case itemRSHeader:
		rsStyle := p.styles.SectionHeader
		if p.collapsed[item.rsName] {
			agents := p.grouped[item.rsName]
			dots := p.agentDots(agents)
			return fmt.Sprintf("%s %s (%d)", rsStyle.Render("▸ "+item.rsName), dots, len(agents))
		}
		return rsStyle.Render("▾ " + item.rsName)

	case itemAgent:
		a := item.agent
		indicator := agentIndicator(a, p.styles)
		role := a.Role.String()
		ver := a.Version
		if len(ver) > maxAgentVersionLen {
			ver = ver[:maxAgentVersionLen]
		}
		return fmt.Sprintf("  %s %s  %s  %s", indicator, a.Node, role, ver)
	}
	return ""
}

// renderAgentDetail writes agent detail to the builder.
func (p *clusterPanel) renderAgentDetail(b *strings.Builder, a *sdk.Agent) {
	b.WriteString(p.styles.SectionHeader.Render("Agent"))
	b.WriteByte('\n')

	fmt.Fprintf(b, "  Node:        %s\n", a.Node)
	fmt.Fprintf(b, "  Replica Set: %s\n", a.ReplicaSet)
	fmt.Fprintf(b, "  Role:        %s\n", a.Role)
	fmt.Fprintf(b, "  Version:     %s\n", a.Version)

	status := p.styles.StatusOK.Render("OK")
	if a.Stale {
		status = p.styles.StatusMuted.Render("Stale")
	} else if !a.OK {
		status = p.styles.StatusError.Render("Error")
	}
	fmt.Fprintf(b, "  Status:      %s\n", status)

	if len(a.Errors) > 0 {
		b.WriteByte('\n')
		b.WriteString(p.styles.StatusError.Render("  Errors:"))
		b.WriteByte('\n')
		for _, e := range a.Errors {
			fmt.Fprintf(b, "    - %s\n", e)
		}
	}
	b.WriteByte('\n')
}

// renderRSDetail writes replica set summary detail to the builder.
func (p *clusterPanel) renderRSDetail(b *strings.Builder, rsName string) {
	b.WriteString(p.styles.SectionHeader.Render("Replica Set"))
	b.WriteByte('\n')

	agents := p.grouped[rsName]
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
	fmt.Fprintf(b, "  Healthy: %s\n", p.styles.StatusOK.Render(fmt.Sprintf("%d", ok)))
	if stale > 0 {
		fmt.Fprintf(b, "  Stale:   %s\n", p.styles.StatusMuted.Render(fmt.Sprintf("%d", stale)))
	}
	if errCount > 0 {
		fmt.Fprintf(b, "  Error:   %s\n", p.styles.StatusError.Render(fmt.Sprintf("%d", errCount)))
	}
	b.WriteByte('\n')

	b.WriteString(p.styles.Bold.Render("  Agents"))
	b.WriteByte('\n')
	for i := range agents {
		a := &agents[i]
		ind := agentIndicator(a, p.styles)
		fmt.Fprintf(b, "  %s %s  %s  %s\n", ind, a.Node, a.Role, a.Version)
	}
}

// agentDots returns a string of status indicator dots for a slice of agents.
func (p *clusterPanel) agentDots(agents []sdk.Agent) string {
	var b strings.Builder
	for i := range agents {
		b.WriteString(agentIndicator(&agents[i], p.styles))
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
	keys := slices.Sorted(maps.Keys(m))
	return keys
}
