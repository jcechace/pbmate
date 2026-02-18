package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

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

// detailView renders the detail panel content for the selected item.
func (m *overviewModel) detailView() string {
	sel := m.selectedItem()
	if sel == nil {
		return m.styles.StatusMuted.Render("No selection")
	}
	var b strings.Builder
	switch sel.kind {
	case itemAgent:
		m.renderAgentDetail(&b, sel.agent)
	case itemBackup:
		m.renderBackupDetail(&b, sel.backup)
	}
	return b.String()
}

// statusView renders the running operations and PITR status.
func (m *overviewModel) statusView() string {
	var b strings.Builder
	m.renderOperationsSection(&b)
	m.renderPITRSection(&b)
	return b.String()
}

// renderAgentDetail writes agent detail to the builder.
func (m *overviewModel) renderAgentDetail(b *strings.Builder, a *sdk.Agent) {
	header := lipgloss.NewStyle().Bold(true).Foreground(m.styles.FocusedBorderColor)
	b.WriteString(header.Render("Agent"))
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

// renderBackupDetail writes backup detail to the builder.
func (m *overviewModel) renderBackupDetail(b *strings.Builder, bk *sdk.Backup) {
	header := lipgloss.NewStyle().Bold(true).Foreground(m.styles.FocusedBorderColor)
	b.WriteString(header.Render("Backup"))
	b.WriteByte('\n')

	fmt.Fprintf(b, "  Name:        %s\n", bk.Name)
	fmt.Fprintf(b, "  Type:        %s\n", bk.Type)

	indicator := m.statusIndicator(bk.Status)
	fmt.Fprintf(b, "  Status:      %s %s\n", indicator, bk.Status)

	if bk.Size > 0 {
		fmt.Fprintf(b, "  Size:        %s", humanBytes(bk.Size))
		if bk.SizeUncompressed > 0 {
			fmt.Fprintf(b, " (%s uncompressed)", humanBytes(bk.SizeUncompressed))
		}
		b.WriteByte('\n')
	}

	if !bk.Compression.IsZero() {
		fmt.Fprintf(b, "  Compression: %s\n", bk.Compression)
	}
	if !bk.ConfigName.IsZero() {
		fmt.Fprintf(b, "  Config:      %s\n", bk.ConfigName)
	}
	if !bk.StartTS.IsZero() {
		fmt.Fprintf(b, "  Started:     %s\n", bk.StartTS.Format("2006-01-02 15:04:05"))
	}
	if !bk.LastTransitionTS.IsZero() && !bk.StartTS.IsZero() {
		dur := bk.LastTransitionTS.Sub(bk.StartTS).Truncate(time.Second)
		if dur > 0 {
			fmt.Fprintf(b, "  Duration:    %s\n", dur)
		}
	}

	if bk.Error != "" {
		fmt.Fprintf(b, "  Error:       %s\n", m.styles.StatusError.Render(bk.Error))
	}

	if len(bk.Replsets) > 0 {
		b.WriteByte('\n')
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("  Replica Sets"))
		b.WriteByte('\n')
		for _, rs := range bk.Replsets {
			ind := m.statusIndicator(rs.Status)
			node := rs.Node
			if node == "" {
				node = "-"
			}
			fmt.Fprintf(b, "  %s %s: %s  (%s)\n", ind, rs.Name, rs.Status, node)
		}
	}
	b.WriteByte('\n')
}

// renderOperationsSection writes the running operations section.
func (m *overviewModel) renderOperationsSection(b *strings.Builder) {
	header := lipgloss.NewStyle().Bold(true).Foreground(m.styles.FocusedBorderColor)
	b.WriteString(header.Render("Running Operations"))
	b.WriteByte('\n')

	if len(m.data.operations) == 0 {
		b.WriteString(m.styles.StatusMuted.Render("  none"))
		b.WriteByte('\n')
	} else {
		for _, op := range m.data.operations {
			fmt.Fprintf(b, "  %s %s  %s\n", m.styles.StatusWarning.Render("●"), op.Type, op.OPID)
		}
	}
	b.WriteByte('\n')
}

// renderPITRSection writes the PITR status section.
func (m *overviewModel) renderPITRSection(b *strings.Builder) {
	header := lipgloss.NewStyle().Bold(true).Foreground(m.styles.FocusedBorderColor)
	b.WriteString(header.Render("PITR"))
	b.WriteByte('\n')

	pitr := m.data.pitr
	if pitr == nil {
		b.WriteString(m.styles.StatusMuted.Render("  no data"))
		b.WriteByte('\n')
		return
	}

	enabledStr := m.styles.StatusMuted.Render("false")
	if pitr.Enabled {
		enabledStr = m.styles.StatusOK.Render("true")
	}
	fmt.Fprintf(b, "  Enabled: %s\n", enabledStr)

	runningStr := m.styles.StatusMuted.Render("false")
	if pitr.Running {
		runningStr = m.styles.StatusOK.Render("true")
	}
	fmt.Fprintf(b, "  Running: %s\n", runningStr)

	if pitr.Error != "" {
		fmt.Fprintf(b, "  Error:   %s\n", m.styles.StatusError.Render(pitr.Error))
	}

	if len(m.data.timelines) > 0 {
		b.WriteByte('\n')
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("  Timelines"))
		b.WriteByte('\n')
		for _, tl := range m.data.timelines {
			start := tl.Start.Time().Format("2006-01-02 15:04:05")
			end := tl.End.Time().Format("2006-01-02 15:04:05")
			fmt.Fprintf(b, "  %s -> %s\n", start, end)
		}
	}
}

// leftView renders the left panel content.
func (m *overviewModel) leftView(width, height int) string {
	cursor := lipgloss.NewStyle().Foreground(m.styles.FocusedBorderColor)

	var b strings.Builder
	for i, item := range m.items {
		if i > 0 {
			b.WriteByte('\n')
		}
		line := m.renderItem(item)
		if i == m.cursor && item.selectable {
			if m.focus == panelLeft {
				line = cursor.Render("▶ ") + lipgloss.NewStyle().Bold(true).Render(line)
			} else {
				line = "  " + lipgloss.NewStyle().Bold(true).Render(line)
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

// humanBytes formats a byte count into a human-readable string.
func humanBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1fGB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1fMB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1fKB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%dB", b)
	}
}
