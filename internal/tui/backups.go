package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// backupsModel is the sub-model for the Backups tab.
type backupsModel struct {
	client  *sdk.Client
	backups []sdk.Backup
	cursor  int
	focus   panel
	styles  *Styles
}

// newBackupsModel creates a new backups sub-model.
func newBackupsModel(client *sdk.Client, styles *Styles) backupsModel {
	return backupsModel{
		client: client,
		styles: styles,
		focus:  panelLeft,
	}
}

// setData updates the backup list from a fresh poll.
func (m *backupsModel) setData(d backupsData) {
	m.backups = d.backups
	if m.cursor >= len(m.backups) {
		m.cursor = max(0, len(m.backups)-1)
	}
}

// update handles key messages for the Backups tab.
// Returns a tea.Cmd if an action was triggered, nil otherwise.
func (m *backupsModel) update(msg tea.KeyMsg, keys globalKeyMap) tea.Cmd {
	switch {
	case key.Matches(msg, keys.Down):
		if m.cursor < len(m.backups)-1 {
			m.cursor++
		}
	case key.Matches(msg, keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, keys.Left):
		m.focus = panelLeft
	case key.Matches(msg, keys.Right):
		m.focus = panelRight
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

// leftView renders the backup list for the left panel.
func (m *backupsModel) leftView(width, height int) string {
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
	if len(name) > 22 {
		name = name[:22]
	}
	size := ""
	if bk.Size > 0 {
		size = humanBytes(bk.Size)
	}
	return fmt.Sprintf("%s %s  %s  %s", ind, name, bk.Type, size)
}

// detailView renders the full detail for the selected backup.
func (m *backupsModel) detailView() string {
	sel := m.selectedBackup()
	if sel == nil {
		return m.styles.StatusMuted.Render("No selection")
	}
	var b strings.Builder
	renderBackupDetail(&b, sel, m.styles)
	return b.String()
}
