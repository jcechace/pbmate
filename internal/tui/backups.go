package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

const backupTimeFormat = "2006-01-02 15:04" // display format for backup timestamps
const backupTypeColWidth = 11               // pad type to widest value ("incremental")

// listMode selects which list is shown in the Backups tab.
type listMode int

const (
	listBackups listMode = iota
	listRestores
)

// backupItemKind identifies the type of item in the backup tree.
type backupItemKind int

const (
	itemPITR          backupItemKind = iota // PITR timeline range
	itemProfileHeader                       // collapsible profile header
	itemBackup                              // individual backup (or incremental base)
	itemIncrChild                           // incremental chain child (indented under base)
)

// backupItem is a single row in the backup list tree.
type backupItem struct {
	kind     backupItemKind
	timeline *sdk.Timeline // set when kind == itemPITR
	profile  string        // set for itemProfileHeader and itemBackup
	count    int           // set for itemProfileHeader: number of backups
	backup   *sdk.Backup   // set when kind == itemBackup
}

// backupsModel is the sub-model for the Backups tab.
type backupsModel struct {
	// Backup tree view.
	items     []backupItem
	timelines []sdk.Timeline
	grouped   map[string][]sdk.Backup // profile name -> backups
	profiles  []string                // sorted profile names (main first)
	collapsed map[string]bool         // profile name -> collapsed state

	// Restore flat list.
	restores []sdk.Restore

	mode          listMode
	backupCursor  int
	restoreCursor int
	focus         panel
	styles        *Styles

	// Panel viewports — each produces exactly its allocated height.
	listVP   viewport.Model
	detailVP viewport.Model
}

// newBackupsModel creates a new backups sub-model.
func newBackupsModel(styles *Styles) backupsModel {
	return backupsModel{
		styles:    styles,
		focus:     panelLeft,
		collapsed: make(map[string]bool),
		listVP:    newPanelViewport(),
		detailVP:  newPanelViewport(),
	}
}

// setBackupData updates the backup list from a fresh poll.
func (m *backupsModel) setBackupData(d backupsData) {
	m.timelines = d.timelines
	m.grouped = groupBackupsByProfile(d.backups)
	m.profiles = sortedProfileNames(m.grouped)
	m.rebuildItems()
}

// setRestoreData updates the restore list from a fresh poll.
func (m *backupsModel) setRestoreData(d restoresData) {
	m.restores = d.restores
	if m.restoreCursor >= len(m.restores) {
		m.restoreCursor = max(0, len(m.restores)-1)
	}
	if m.mode == listRestores {
		m.rebuildListContent()
		m.rebuildDetailContent()
	}
}

// selectedItem returns the currently selected backup-tree item, or nil.
func (m *backupsModel) selectedItem() *backupItem {
	if m.mode != listBackups {
		return nil
	}
	if m.backupCursor >= 0 && m.backupCursor < len(m.items) {
		return &m.items[m.backupCursor]
	}
	return nil
}

// selectedBackup returns the backup under the cursor, or nil if the cursor
// is on a non-backup item or in restore mode.
func (m *backupsModel) selectedBackup() *sdk.Backup {
	item := m.selectedItem()
	if item != nil && (item.kind == itemBackup || item.kind == itemIncrChild) {
		return item.backup
	}
	return nil
}

// selectedRestore returns the currently selected restore, or nil.
func (m *backupsModel) selectedRestore() *sdk.Restore {
	if m.restoreCursor >= 0 && m.restoreCursor < len(m.restores) {
		return &m.restores[m.restoreCursor]
	}
	return nil
}

// update handles key messages for the Backups tab.
// Returns a tea.Cmd if an action was triggered, nil otherwise.
func (m *backupsModel) update(msg tea.KeyMsg, keys globalKeyMap) tea.Cmd {
	switch {
	case key.Matches(msg, backupKeys.Toggle) && m.focus == panelLeft:
		if m.mode == listBackups {
			m.mode = listRestores
		} else {
			m.mode = listBackups
		}
		m.rebuildListContent()
		m.rebuildDetailContent()
	case key.Matches(msg, keys.NextPanel):
		m.cyclePanel(1)
	case key.Matches(msg, keys.PrevPanel):
		m.cyclePanel(-1)
	case key.Matches(msg, keys.Down):
		m.handleVertical(1)
	case key.Matches(msg, keys.Up):
		m.handleVertical(-1)
	case key.Matches(msg, keys.Delete):
		if m.mode == listBackups {
			if sel := m.selectedBackup(); sel != nil {
				baseName, title, desc := m.resolveDeleteTarget(sel)
				return requestDeleteConfirm(baseName, title, desc)
			}
		}
	}

	// Toggle collapse on space/enter when on a profile header.
	if m.mode == listBackups && m.focus == panelLeft {
		if msg.String() == " " || msg.String() == "enter" {
			if item := m.selectedItem(); item != nil && item.kind == itemProfileHeader {
				m.collapsed[item.profile] = !m.collapsed[item.profile]
				m.rebuildItems()
			}
		}
	}

	return nil
}

// cyclePanel moves focus to the next or previous panel.
func (m *backupsModel) cyclePanel(delta int) {
	old := m.focus
	m.focus = panel((int(m.focus) + delta + int(panelCount)) % int(panelCount))
	if m.focus != old {
		m.rebuildListContent() // update cursor ▶ visibility
	}
}

// handleVertical dispatches Up/Down to the focused panel.
func (m *backupsModel) handleVertical(delta int) {
	switch m.focus {
	case panelLeft:
		if m.mode == listBackups {
			m.moveCursor(delta)
		} else {
			m.moveRestoreCursor(delta)
		}
	case panelRight:
		if delta > 0 {
			m.detailVP.ScrollDown(delta)
		} else {
			m.detailVP.ScrollUp(-delta)
		}
	}
}

// moveCursor moves the backup tree cursor by delta, always landing on a
// selectable item. Rebuilds content after moving.
func (m *backupsModel) moveCursor(delta int) {
	if len(m.items) == 0 {
		return
	}
	m.backupCursor += delta
	if m.backupCursor < 0 {
		m.backupCursor = 0
	}
	if m.backupCursor >= len(m.items) {
		m.backupCursor = len(m.items) - 1
	}
	m.rebuildListContent()
	m.rebuildDetailContent()
}

// moveRestoreCursor moves the restore list cursor by delta.
func (m *backupsModel) moveRestoreCursor(delta int) {
	n := len(m.restores)
	if n == 0 {
		return
	}
	m.restoreCursor += delta
	if m.restoreCursor < 0 {
		m.restoreCursor = 0
	}
	if m.restoreCursor >= n {
		m.restoreCursor = n - 1
	}
	m.rebuildListContent()
	m.rebuildDetailContent()
}

// --- Backup tree item management ---

// rebuildItems reconstructs the flat backup item list from grouped data,
// respecting collapsed state. Preserves cursor position by item identity.
func (m *backupsModel) rebuildItems() {
	defer m.rebuildListContent()
	defer m.rebuildDetailContent()

	// Remember currently selected item identity for cursor stability.
	var selectedBackupName string
	var selectedProfile string
	selectedTimelineIdx := -1
	if sel := m.selectedItem(); sel != nil {
		switch sel.kind {
		case itemBackup, itemIncrChild:
			selectedBackupName = sel.backup.Name
		case itemProfileHeader:
			selectedProfile = sel.profile
		case itemPITR:
			// Find the index of this timeline by matching start/end values.
			// Cannot compare pointers because m.timelines is replaced on each
			// data refresh, so old item pointers become stale.
			for i := range m.timelines {
				if m.timelines[i].Start == sel.timeline.Start && m.timelines[i].End == sel.timeline.End {
					selectedTimelineIdx = i
					break
				}
			}
		}
	}

	var items []backupItem

	// PITR timelines at the top.
	for i := range m.timelines {
		items = append(items, backupItem{
			kind:     itemPITR,
			timeline: &m.timelines[i],
		})
	}

	// Profile sections.
	for _, profile := range m.profiles {
		backups := m.grouped[profile]
		items = append(items, backupItem{
			kind:    itemProfileHeader,
			profile: profile,
			count:   len(backups),
		})
		if !m.collapsed[profile] {
			items = append(items, chainOrderedItems(profile, backups)...)
		}
	}

	m.items = items

	// Restore cursor to the same item if possible.
	m.backupCursor = 0
	if selectedBackupName != "" {
		for i, item := range m.items {
			if (item.kind == itemBackup || item.kind == itemIncrChild) && item.backup.Name == selectedBackupName {
				m.backupCursor = i
				return
			}
		}
	}
	if selectedProfile != "" {
		for i, item := range m.items {
			if item.kind == itemProfileHeader && item.profile == selectedProfile {
				m.backupCursor = i
				return
			}
		}
	}
	if selectedTimelineIdx >= 0 && selectedTimelineIdx < len(m.timelines) {
		// Timeline items are emitted in order, so the Nth PITR item
		// corresponds to the Nth timeline.
		pitrCount := 0
		for i, item := range m.items {
			if item.kind == itemPITR {
				if pitrCount == selectedTimelineIdx {
					m.backupCursor = i
					return
				}
				pitrCount++
			}
		}
	}
}

// --- List content rendering ---

// listContent builds the list content string for the current mode.
func (m *backupsModel) listContent() string {
	if m.mode == listRestores {
		return m.restoreListContent()
	}
	return m.backupTreeContent()
}

// backupTreeContent builds the backup tree content string.
func (m *backupsModel) backupTreeContent() string {
	if len(m.items) == 0 {
		return m.styles.StatusMuted.Render("No backups")
	}

	lines := make([]string, len(m.items))
	for i, item := range m.items {
		lines[i] = m.renderBackupItem(item)
	}
	return renderCursorList(lines, m.backupCursor, m.focus == panelLeft, m.styles)
}

// restoreListContent builds the restore list content string.
func (m *backupsModel) restoreListContent() string {
	if len(m.restores) == 0 {
		return m.styles.StatusMuted.Render("No restores")
	}

	lines := make([]string, len(m.restores))
	for i := range m.restores {
		lines[i] = m.renderRestoreLine(&m.restores[i])
	}
	return renderCursorList(lines, m.restoreCursor, m.focus == panelLeft, m.styles)
}

// --- Item rendering ---

// renderBackupItem renders a single item line for the backup tree.
func (m *backupsModel) renderBackupItem(item backupItem) string {
	switch item.kind {
	case itemPITR:
		return m.renderPITRLine(item.timeline)
	case itemProfileHeader:
		return m.renderProfileHeader(item.profile, item.count)
	case itemBackup:
		return "  " + m.renderBackupLine(item.backup)
	case itemIncrChild:
		return "    " + m.renderBackupLine(item.backup)
	}
	return ""
}

// renderPITRLine renders a PITR timeline range.
func (m *backupsModel) renderPITRLine(tl *sdk.Timeline) string {
	start := tl.Start.Time().UTC().Format(backupTimeFormat)
	end := tl.End.Time().UTC().Format(backupTimeFormat)
	return fmt.Sprintf("⧖ PITR  %s → %s", start, end)
}

// renderProfileHeader renders a collapsible profile header.
func (m *backupsModel) renderProfileHeader(profile string, count int) string {
	headerStyle := m.styles.SectionHeader
	label := profileDisplayName(profile)
	if m.collapsed[profile] {
		return fmt.Sprintf("%s (%d)", headerStyle.Render("▸ "+label), count)
	}
	return headerStyle.Render("▾ " + label)
}

// renderBackupLine renders a single backup line for the list.
// Layout: status dot, timestamp, padded type, flag column.
// Flags: ⌂ = incremental base, ◇ = selective (namespace-filtered).
func (m *backupsModel) renderBackupLine(bk *sdk.Backup) string {
	ind := statusIndicator(bk.Status, m.styles)

	ts := bk.LastWriteTS.Time().UTC().Format(backupTimeFormat)
	if bk.LastWriteTS.IsZero() {
		ts = bk.StartTS.UTC().Format(backupTimeFormat)
	}

	flag := ""
	if bk.IsIncrementalBase() {
		flag = m.styles.StatusWarning.Render("⌂")
	} else if bk.IsSelective() {
		flag = m.styles.StatusWarning.Render("◇")
	}

	return fmt.Sprintf("%s %s  %-*s %s", ind, ts, backupTypeColWidth, bk.Type, flag)
}

// renderRestoreLine renders a single restore line for the list.
// Shows the start time, source backup type, and status.
func (m *backupsModel) renderRestoreLine(rs *sdk.Restore) string {
	ind := statusIndicator(rs.Status, m.styles)
	ts := rs.StartTS.UTC().Format(backupTimeFormat)
	return fmt.Sprintf("%s %s  %s", ind, ts, rs.Type)
}

// --- Detail content ---

// detailContent builds the detail content string for the current mode.
func (m *backupsModel) detailContent() string {
	if m.mode == listRestores {
		sel := m.selectedRestore()
		if sel == nil {
			return m.styles.StatusMuted.Render("No selection")
		}
		var b strings.Builder
		renderRestoreDetail(&b, sel, m.styles)
		return b.String()
	}

	item := m.selectedItem()
	if item == nil {
		return m.styles.StatusMuted.Render("No selection")
	}

	var b strings.Builder
	switch item.kind {
	case itemPITR:
		m.renderPITRDetail(&b, item.timeline)
	case itemProfileHeader:
		m.renderProfileDetail(&b, item.profile)
	case itemBackup, itemIncrChild:
		renderBackupDetail(&b, item.backup, m.styles)
	}
	return b.String()
}

// renderPITRDetail writes PITR timeline detail to the builder.
func (m *backupsModel) renderPITRDetail(b *strings.Builder, tl *sdk.Timeline) {
	b.WriteString(m.styles.SectionHeader.Render("PITR Timeline"))
	b.WriteByte('\n')

	start := tl.Start.Time().UTC()
	end := tl.End.Time().UTC()
	fmt.Fprintf(b, "  Start:    %s\n", start.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(b, "  End:      %s\n", end.Format("2006-01-02 15:04:05"))

	dur := end.Sub(start)
	if dur > 0 {
		fmt.Fprintf(b, "  Duration: %s\n", dur.Truncate(time.Second).String())
	}
}

// renderProfileDetail writes profile summary detail to the builder.
func (m *backupsModel) renderProfileDetail(b *strings.Builder, profile string) {
	b.WriteString(m.styles.SectionHeader.Render("Storage Profile"))
	b.WriteByte('\n')

	fmt.Fprintf(b, "  Name:    %s\n", profileDisplayName(profile))
	fmt.Fprintf(b, "  Backups: %d\n", len(m.grouped[profile]))
}

// --- View ---

// borderColor returns the border color for the given panel, highlighting
// the focused panel.
func (m *backupsModel) borderColor(p panel) lipgloss.TerminalColor {
	if m.focus == p {
		return m.styles.FocusedBorderColor
	}
	return m.styles.UnfocusedBorderColor
}

// segmentedTitle renders the toggle title: [Backups] Restores or Backups [Restores].
// The active label is bold with brackets, the inactive one is muted. Toggle
// with tab is shown in the bottom bar hints.
func (m *backupsModel) segmentedTitle(borderColor lipgloss.TerminalColor) string {
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(borderColor)
	inactiveStyle := m.styles.StatusMuted
	bracketStyle := lipgloss.NewStyle().Bold(true).Foreground(borderColor)

	renderLabel := func(label string, active bool) string {
		if active {
			return bracketStyle.Render("[") + activeStyle.Render(label) + bracketStyle.Render("]")
		}
		return inactiveStyle.Render(label)
	}

	return renderLabel("Backups", m.mode == listBackups) +
		" " +
		renderLabel("Restores", m.mode == listRestores)
}

// view renders the Backups tab with left list + right detail panels.
func (m *backupsModel) view(totalW, totalH int) string {
	panelLeftW, panelRightW, contentLeftW, contentRightW := horizontalSplit(totalW)
	innerH := innerHeight(totalH)

	// Set viewport dimensions (known only at View time) and render.
	m.listVP.Width = contentLeftW
	m.listVP.Height = innerH
	m.detailVP.Width = contentRightW
	m.detailVP.Height = innerH

	border := m.styles.PanelBorder
	leftColor := m.borderColor(panelLeft)

	// Build the left panel without a title, then apply the segmented title border.
	left := renderTitledPanel("", m.listVP.View(),
		m.styles.LeftPanel, panelLeftW, innerH, border, leftColor)
	left = replaceStyledTitleBorder(left, m.segmentedTitle(leftColor),
		panelLeftW+panelBorderH, border, leftColor)

	right := renderTitledPanel("Detail", m.detailVP.View(),
		m.styles.RightPanel, panelRightW, innerH, border, m.borderColor(panelRight))

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

// resize precomputes viewport dimensions so Update-time operations (scrolling)
// use correct bounds. View-time dimension setting operates on a value copy.
func (m *backupsModel) resize(totalW, totalH int) {
	_, _, contentLeftW, contentRightW := horizontalSplit(totalW)
	innerH := innerHeight(totalH)

	m.listVP.Width = contentLeftW
	m.listVP.Height = innerH
	m.detailVP.Width = contentRightW
	m.detailVP.Height = innerH
}

// --- Viewport content rebuilders ---

func (m *backupsModel) rebuildListContent() {
	m.listVP.SetContent(m.listContent())
}

func (m *backupsModel) rebuildDetailContent() {
	m.detailVP.SetContent(m.detailContent())
}

// --- Delete helpers ---

// resolveDeleteTarget determines what to delete when the user presses 'd' on a
// backup. For non-incremental backups it returns the backup itself. For any
// incremental chain member it walks to the base and counts the chain, since PBM
// only supports deleting from the base (which removes the entire chain).
func (m *backupsModel) resolveDeleteTarget(bk *sdk.Backup) (baseName, title, description string) {
	profile := profileDisplayName(bk.ConfigName.String())

	if !bk.IsIncremental() {
		return bk.Name, "Delete Backup",
			fmt.Sprintf("%s\n%s · %s\nProfile: %s", bk.Name, bk.Type, bk.Status, profile)
	}

	// Find the profile's backup list for chain resolution.
	// ConfigName is always normalized — never zero (see sdk.Backup.ConfigName).
	backups := m.grouped[bk.ConfigName.String()]

	baseName, count := resolveIncrChain(bk, backups)

	return baseName, "Delete Incremental Chain",
		fmt.Sprintf("⌂ %s\nand its increments (%d total)\nProfile: %s", baseName, count, profile)
}
