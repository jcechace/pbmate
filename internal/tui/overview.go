package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// itemKind identifies the type of item in the overview list.
type itemKind int

const (
	itemSectionHeader itemKind = iota
	itemRSHeader
	itemAgent
	itemBackup
)

// overviewItem is a single row in the overview left panel.
type overviewItem struct {
	kind       itemKind
	label      string      // rendered display text (without selection highlight)
	agent      *sdk.Agent  // set when kind == itemAgent
	backup     *sdk.Backup // set when kind == itemBackup
	rsName     string      // set when kind == itemRSHeader
	selectable bool        // whether the cursor can land here
}

// panel identifies which panel has focus.
type panel int

const (
	panelLeft panel = iota
	panelRight
)

// overviewModel is the sub-model for the Overview tab.
type overviewModel struct {
	items  []overviewItem
	cursor int
	focus  panel
	styles *Styles
	data   overviewData
}

// newOverviewModel creates a new overview sub-model.
func newOverviewModel(styles *Styles) overviewModel {
	return overviewModel{
		styles: styles,
		focus:  panelLeft,
	}
}

// setData rebuilds the item list from fresh overview data.
func (m *overviewModel) setData(d overviewData) {
	m.data = d

	var items []overviewItem

	// Section: Cluster (agents grouped by replica set).
	items = append(items, overviewItem{
		kind:  itemSectionHeader,
		label: "Cluster",
	})

	grouped := groupAgentsByRS(d.agents)
	rsNames := sortedKeys(grouped)

	for _, rs := range rsNames {
		items = append(items, overviewItem{
			kind:   itemRSHeader,
			label:  rs,
			rsName: rs,
		})
		for i := range grouped[rs] {
			a := &grouped[rs][i]
			items = append(items, overviewItem{
				kind:       itemAgent,
				agent:      a,
				selectable: true,
			})
		}
	}

	// Section: Recent Backups.
	items = append(items, overviewItem{
		kind:  itemSectionHeader,
		label: "Recent Backups",
	})

	for i := range d.recentBackups {
		b := &d.recentBackups[i]
		items = append(items, overviewItem{
			kind:       itemBackup,
			backup:     b,
			selectable: true,
		})
	}

	m.items = items

	// Ensure cursor is on a selectable item.
	if m.cursor >= len(m.items) {
		m.cursor = 0
	}
	m.ensureSelectable(1)
}

// update handles key messages for the overview tab.
func (m *overviewModel) update(msg tea.KeyMsg, keys globalKeyMap) {
	switch {
	case key.Matches(msg, keys.Down):
		m.moveCursor(1)
	case key.Matches(msg, keys.Up):
		m.moveCursor(-1)
	case key.Matches(msg, keys.Left):
		m.focus = panelLeft
	case key.Matches(msg, keys.Right):
		m.focus = panelRight
	}
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

// selectedItem returns the currently selected item, or nil if none.
func (m *overviewModel) selectedItem() *overviewItem {
	if m.cursor >= 0 && m.cursor < len(m.items) && m.items[m.cursor].selectable {
		return &m.items[m.cursor]
	}
	return nil
}

// rightView renders the right panel content (detail for selected item).
func (m *overviewModel) rightView(width, height int) string {
	sel := m.selectedItem()
	if sel == nil {
		return m.styles.StatusMuted.Render("No selection")
	}
	switch sel.kind {
	case itemAgent:
		return fmt.Sprintf("Agent: %s\nReplica Set: %s\nRole: %s\nVersion: %s",
			sel.agent.Node, sel.agent.ReplicaSet, sel.agent.Role, sel.agent.Version)
	case itemBackup:
		return fmt.Sprintf("Backup: %s\nType: %s\nStatus: %s",
			sel.backup.Name, sel.backup.Type, sel.backup.Status)
	}
	return ""
}

// leftView renders the left panel content.
func (m *overviewModel) leftView(width, height int) string {
	var b strings.Builder
	for i, item := range m.items {
		if i > 0 {
			b.WriteByte('\n')
		}
		line := m.renderItem(item)
		if i == m.cursor && m.focus == panelLeft {
			line = lipgloss.NewStyle().Reverse(true).Render(line)
		} else if i == m.cursor {
			line = lipgloss.NewStyle().Bold(true).Render(line)
		}
		b.WriteString(line)
	}
	return b.String()
}

// renderItem renders a single item line for the left panel.
func (m *overviewModel) renderItem(item overviewItem) string {
	switch item.kind {
	case itemSectionHeader:
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(m.styles.FocusedBorderColor).
			Render(fmt.Sprintf("── %s ──", item.label))

	case itemRSHeader:
		return fmt.Sprintf("  %s", item.label)

	case itemAgent:
		a := item.agent
		indicator := m.agentIndicator(a)
		role := a.Role.String()
		ver := a.Version
		if len(ver) > 5 {
			ver = ver[:5]
		}
		return fmt.Sprintf("    %s %s  %s  %s", indicator, a.Node, role, ver)

	case itemBackup:
		b := item.backup
		indicator := m.statusIndicator(b.Status)
		name := b.Name
		if len(name) > 20 {
			name = name[:20]
		}
		return fmt.Sprintf("  %s %s  %s  %s", indicator, name, b.Type, b.Status)
	}
	return ""
}

// agentIndicator returns a colored status dot for an agent.
func (m *overviewModel) agentIndicator(a *sdk.Agent) string {
	if a.Stale {
		return m.styles.StatusMuted.Render("○")
	}
	if !a.OK || len(a.Errors) > 0 {
		return m.styles.StatusError.Render("●")
	}
	return m.styles.StatusOK.Render("●")
}

// statusIndicator returns a colored status dot for a PBM status.
func (m *overviewModel) statusIndicator(s sdk.Status) string {
	switch {
	case s.Equal(sdk.StatusDone):
		return m.styles.StatusOK.Render("●")
	case s.Equal(sdk.StatusError), s.Equal(sdk.StatusPartlyDone):
		return m.styles.StatusError.Render("●")
	case s.Equal(sdk.StatusCancelled):
		return m.styles.StatusMuted.Render("●")
	case s.IsTerminal():
		return m.styles.StatusMuted.Render("●")
	default:
		// Running / in-progress states.
		return m.styles.StatusWarning.Render("●")
	}
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
