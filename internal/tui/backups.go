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

const maxBackupNameList = 22 // max backup name length in the backup list

// backupsModel is the sub-model for the Backups tab.
type backupsModel struct {
	client  *sdk.Client
	backups []sdk.Backup
	cursor  int
	focus   panel
	styles  *Styles

	// Panel viewports — each produces exactly its allocated height.
	listVP   viewport.Model
	detailVP viewport.Model
}

// newBackupsModel creates a new backups sub-model.
func newBackupsModel(client *sdk.Client, styles *Styles) backupsModel {
	return backupsModel{
		client:   client,
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
	case key.Matches(msg, keys.Down):
		if m.cursor < len(m.backups)-1 {
			m.cursor++
		}
		m.rebuildListContent()
		m.rebuildDetailContent()
	case key.Matches(msg, keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
		m.rebuildListContent()
		m.rebuildDetailContent()
	case key.Matches(msg, keys.Left):
		m.focus = panelLeft
		m.rebuildListContent()
	case key.Matches(msg, keys.Right):
		m.focus = panelRight
		m.rebuildListContent()
	case key.Matches(msg, backupKeys.Start):
		return startBackupCmd(m.client)
	case key.Matches(msg, backupKeys.Cancel):
		return cancelBackupCmd(m.client)
	case key.Matches(msg, backupKeys.Delete):
		if sel := m.selectedBackup(); sel != nil {
			return deleteBackupCmd(m.client, sel.Name)
		}
	}
	return nil
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

// renderBackupLine renders a single backup line for the list.
func (m *backupsModel) renderBackupLine(bk *sdk.Backup) string {
	ind := statusIndicator(bk.Status, m.styles)
	name := bk.Name
	if len(name) > maxBackupNameList {
		name = name[:maxBackupNameList]
	}
	size := ""
	if bk.Size > 0 {
		size = humanBytes(bk.Size)
	}
	return fmt.Sprintf("%s %s  %s  %s", ind, name, bk.Type, size)
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

// --- Viewport content rebuilders ---

func (m *backupsModel) rebuildListContent() {
	m.listVP.SetContent(m.listContent())
}

func (m *backupsModel) rebuildDetailContent() {
	m.detailVP.SetContent(m.detailContent())
}

// --- Viewport size setters ---

func (m *backupsModel) setListSize(width, height int) {
	m.listVP.Width = width
	m.listVP.Height = height
}

func (m *backupsModel) setDetailSize(width, height int) {
	m.detailVP.Width = width
	m.detailVP.Height = height
}

// --- Viewport view methods ---

func (m *backupsModel) listView() string   { return m.listVP.View() }
func (m *backupsModel) detailView() string { return m.detailVP.View() }
