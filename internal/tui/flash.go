package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// clearActionFlashMsg signals that the action flash timeout has expired.
type clearActionFlashMsg struct{}

// setFlash sets or clears the transient error message in the status bar.
// On success (err == nil) the message is cleared — unless a sticky action
// flash is active, in which case it is preserved until its timeout expires.
// On failure the prefix and error are combined into a flash message.
func (m *Model) setFlash(prefix string, err error) {
	if err != nil {
		m.flashErr = fmt.Sprintf("%s: %v", prefix, err)
		m.flashFromAction = false
	} else if !m.flashFromAction {
		m.flashErr = ""
	}
}

// setActionFlash sets or clears a sticky action error in the status bar.
// Action errors persist across poll cycles and auto-clear after a timeout.
// Returns a tea.Cmd that schedules the auto-clear (nil on success/clear).
func (m *Model) setActionFlash(err error) tea.Cmd {
	if err != nil {
		m.flashErr = err.Error()
		m.flashFromAction = true
		return tea.Tick(actionFlashTimeout, func(time.Time) tea.Msg {
			return clearActionFlashMsg{}
		})
	}
	m.flashErr = ""
	m.flashFromAction = false
	return nil
}
