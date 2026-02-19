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
)

// overviewItem is a single row in the overview left panel.
type overviewItem struct {
	kind       itemKind
	label      string     // rendered display text (without selection highlight)
	agent      *sdk.Agent // set when kind == itemAgent
	rsName     string     // set when kind == itemRSHeader
	selectable bool       // whether the cursor can land here
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

	// Build cluster agent tree grouped by replica set.
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

// detailView renders the detail panel content for the selected agent.
func (m *overviewModel) detailView() string {
	sel := m.selectedItem()
	if sel == nil {
		return m.styles.StatusMuted.Render("No selection")
	}
	if sel.kind == itemAgent {
		var b strings.Builder
		m.renderAgentDetail(&b, sel.agent)
		return b.String()
	}
	return m.styles.StatusMuted.Render("No selection")
}

// statusView renders the bottom-left status panel with PITR, ops, latest
// backup, and storage info.
func (m *overviewModel) statusView() string {
	var b strings.Builder
	label := lipgloss.NewStyle().Bold(true).Width(10)

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
		if len(name) > 16 {
			name = name[:16]
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

// logsView renders the bottom-right log panel content.
func (m *overviewModel) logsView() string {
	if len(m.data.logEntries) == 0 {
		return m.styles.StatusMuted.Render(" No log entries")
	}

	var b strings.Builder
	for i, entry := range m.data.logEntries {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(m.formatLogEntry(entry))
	}
	return b.String()
}

// formatLogEntry formats a single log entry for the log panel.
func (m *overviewModel) formatLogEntry(entry sdk.LogEntry) string {
	ts := entry.Timestamp.Format("15:04:05")
	sev := m.severityStyle(entry.Severity).Render(entry.Severity.String()[:1])
	return fmt.Sprintf(" %s %s %s", m.styles.StatusMuted.Render(ts), sev, entry.Message)
}

// severityStyle returns the style for a log severity level.
func (m *overviewModel) severityStyle(sev sdk.LogSeverity) lipgloss.Style {
	switch {
	case sev.Equal(sdk.LogSeverityError), sev.Equal(sdk.LogSeverityFatal):
		return m.styles.StatusError
	case sev.Equal(sdk.LogSeverityWarning):
		return m.styles.StatusWarning
	case sev.Equal(sdk.LogSeverityDebug):
		return m.styles.StatusMuted
	default:
		return lipgloss.NewStyle()
	}
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

// clusterView renders the left panel content (cluster agents only).
func (m *overviewModel) clusterView(width, height int) string {
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
	case itemRSHeader:
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(m.styles.FocusedBorderColor).
			Render(fmt.Sprintf("▾ %s", item.rsName))

	case itemAgent:
		a := item.agent
		indicator := agentIndicator(a, m.styles)
		role := a.Role.String()
		ver := a.Version
		if len(ver) > 5 {
			ver = ver[:5]
		}
		return fmt.Sprintf("  %s %s  %s  %s", indicator, a.Node, role, ver)
	}
	return ""
}

// relativeTime returns a human-readable relative time string.
func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
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
