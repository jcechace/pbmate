package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

const backupTimeFormat = "2006-01-02 15:04" // display format for backup timestamps

// listMode selects which list is shown in the Backups tab.
type listMode int

const (
	listBackups listMode = iota
	listRestores
)

// backupsModel is the sub-model for the Backups tab.
type backupsModel struct {
	backups  []sdk.Backup
	restores []sdk.Restore
	mode     listMode

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
		styles:   styles,
		focus:    panelLeft,
		listVP:   newPanelViewport(),
		detailVP: newPanelViewport(),
	}
}

// setBackupData updates the backup list from a fresh poll.
func (m *backupsModel) setBackupData(d backupsData) {
	m.backups = d.backups
	if m.backupCursor >= len(m.backups) {
		m.backupCursor = max(0, len(m.backups)-1)
	}
	if m.mode == listBackups {
		m.rebuildListContent()
		m.rebuildDetailContent()
	}
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

// cursor returns a pointer to the active cursor for the current mode.
func (m *backupsModel) cursor() *int {
	if m.mode == listRestores {
		return &m.restoreCursor
	}
	return &m.backupCursor
}

// listLen returns the length of the currently active list.
func (m *backupsModel) listLen() int {
	if m.mode == listRestores {
		return len(m.restores)
	}
	return len(m.backups)
}

// update handles key messages for the Backups tab.
// Returns a tea.Cmd if an action was triggered, nil otherwise.
func (m *backupsModel) update(msg tea.KeyMsg, keys globalKeyMap) tea.Cmd {
	switch {
	case key.Matches(msg, backupKeys.ShowBackups):
		if m.mode != listBackups {
			m.mode = listBackups
			m.rebuildListContent()
			m.rebuildDetailContent()
		}
	case key.Matches(msg, backupKeys.ShowRestores):
		if m.mode != listRestores {
			m.mode = listRestores
			m.rebuildListContent()
			m.rebuildDetailContent()
		}
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
				return requestConfirmDelete(sel.Name)
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
		cur := m.cursor()
		n := m.listLen()
		if delta > 0 && *cur < n-1 {
			*cur++
		} else if delta < 0 && *cur > 0 {
			*cur--
		}
		m.rebuildListContent()
		m.rebuildDetailContent()
	case panelRight:
		if delta > 0 {
			m.detailVP.ScrollDown(delta)
		} else {
			m.detailVP.ScrollUp(-delta)
		}
	}
}

// selectedBackup returns the currently selected backup, or nil.
func (m *backupsModel) selectedBackup() *sdk.Backup {
	if m.backupCursor >= 0 && m.backupCursor < len(m.backups) {
		return &m.backups[m.backupCursor]
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

// listContent builds the list content string for the current mode.
func (m *backupsModel) listContent() string {
	if m.mode == listRestores {
		return m.restoreListContent()
	}
	return m.backupListContent()
}

// backupListContent builds the backup list content string.
func (m *backupsModel) backupListContent() string {
	if len(m.backups) == 0 {
		return m.styles.StatusMuted.Render("No backups")
	}

	cursorStyle := lipgloss.NewStyle().Foreground(m.styles.FocusedBorderColor)

	var b strings.Builder
	for i, bk := range m.backups {
		if i > 0 {
			b.WriteByte('\n')
		}
		line := m.renderBackupLine(&bk)
		if i == m.backupCursor {
			if m.focus == panelLeft {
				line = cursorStyle.Render("▶ ") + m.styles.Bold.Render(line)
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

// restoreListContent builds the restore list content string.
func (m *backupsModel) restoreListContent() string {
	if len(m.restores) == 0 {
		return m.styles.StatusMuted.Render("No restores")
	}

	cursorStyle := lipgloss.NewStyle().Foreground(m.styles.FocusedBorderColor)

	var b strings.Builder
	for i, rs := range m.restores {
		if i > 0 {
			b.WriteByte('\n')
		}
		line := m.renderRestoreLine(&rs)
		if i == m.restoreCursor {
			if m.focus == panelLeft {
				line = cursorStyle.Render("▶ ") + m.styles.Bold.Render(line)
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

// renderBackupLine renders a single backup line for the list.
// Shows the restore-to time (LastWriteTS), type, status, and size.
func (m *backupsModel) renderBackupLine(bk *sdk.Backup) string {
	ind := statusIndicator(bk.Status, m.styles)

	ts := bk.LastWriteTS.Time().Format(backupTimeFormat)
	if bk.LastWriteTS.IsZero() {
		ts = bk.StartTS.Format(backupTimeFormat)
	}

	size := ""
	if bk.Size > 0 {
		size = "  " + humanBytes(bk.Size)
	}
	return fmt.Sprintf("%s %s  %s%s", ind, ts, bk.Type, size)
}

// renderRestoreLine renders a single restore line for the list.
// Shows the start time, source backup type, and status.
func (m *backupsModel) renderRestoreLine(rs *sdk.Restore) string {
	ind := statusIndicator(rs.Status, m.styles)
	ts := rs.StartTS.Format(backupTimeFormat)
	return fmt.Sprintf("%s %s  %s", ind, ts, rs.Type)
}

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

	sel := m.selectedBackup()
	if sel == nil {
		return m.styles.StatusMuted.Render("No selection")
	}
	var b strings.Builder
	renderBackupDetail(&b, sel, m.styles)
	return b.String()
}

// borderColor returns the border color for the given panel, highlighting
// the focused panel.
func (m *backupsModel) borderColor(p panel) lipgloss.TerminalColor {
	if m.focus == p {
		return m.styles.FocusedBorderColor
	}
	return m.styles.UnfocusedBorderColor
}

// segmentedTitle renders the toggle title: [Backups] Restores or Backups [Restores].
// The first letter of each label is rendered in the hint-key color to indicate
// the keyboard accelerator; the active label is bold, the inactive one muted.
func (m *backupsModel) segmentedTitle(borderColor lipgloss.TerminalColor) string {
	keyStyle := m.styles.HintKey
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(borderColor)
	inactiveStyle := m.styles.StatusMuted
	bracketStyle := lipgloss.NewStyle().Bold(true).Foreground(borderColor)

	renderLabel := func(label string, active bool) string {
		first := keyStyle.Render(string(label[0]))
		rest := label[1:]
		if active {
			return bracketStyle.Render("[") + first + activeStyle.Render(rest) + bracketStyle.Render("]")
		}
		return first + inactiveStyle.Render(rest)
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
