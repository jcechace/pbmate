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

// backupsModel is the sub-model for the Backups tab.
type backupsModel struct {
	backups []sdk.Backup
	cursor  int
	focus   panel
	styles  *Styles

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

// setData updates the backup list from a fresh poll.
func (m *backupsModel) setData(d backupsData) {
	m.backups = d.backups
	if m.cursor >= len(m.backups) {
		m.cursor = max(0, len(m.backups)-1)
	}
	m.rebuildListContent()
	m.rebuildDetailContent()
}

// update handles key messages for the Backups tab.
// Returns a tea.Cmd if an action was triggered, nil otherwise.
func (m *backupsModel) update(msg tea.KeyMsg, keys globalKeyMap) tea.Cmd {
	switch {
	case key.Matches(msg, keys.NextPanel):
		m.cyclePanel(1)
	case key.Matches(msg, keys.PrevPanel):
		m.cyclePanel(-1)
	case key.Matches(msg, keys.Down):
		m.handleVertical(1)
	case key.Matches(msg, keys.Up):
		m.handleVertical(-1)
	case key.Matches(msg, keys.Delete):
		if sel := m.selectedBackup(); sel != nil {
			return requestConfirmDelete(sel.Name)
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
		if delta > 0 && m.cursor < len(m.backups)-1 {
			m.cursor++
		} else if delta < 0 && m.cursor > 0 {
			m.cursor--
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
	if m.cursor >= 0 && m.cursor < len(m.backups) {
		return &m.backups[m.cursor]
	}
	return nil
}

// listContent builds the backup list content string.
func (m *backupsModel) listContent() string {
	if len(m.backups) == 0 {
		return m.styles.StatusMuted.Render("No backups")
	}

	cursor := lipgloss.NewStyle().Foreground(m.styles.FocusedBorderColor)

	var b strings.Builder
	for i, bk := range m.backups {
		if i > 0 {
			b.WriteByte('\n')
		}
		line := m.renderBackupLine(&bk)
		if i == m.cursor {
			if m.focus == panelLeft {
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

// detailContent builds the detail content string for the selected backup.
func (m *backupsModel) detailContent() string {
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
	left := renderTitledPanel("Backups", m.listVP.View(),
		m.styles.LeftPanel, panelLeftW, innerH, border, m.borderColor(panelLeft))
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
