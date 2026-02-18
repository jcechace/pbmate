package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	// idleInterval is the polling interval when no operations are running.
	idleInterval = 10 * time.Second

	// activeInterval is the polling interval when an operation is in progress.
	activeInterval = 2 * time.Second
)

// tickMsg signals that it is time to fetch fresh data.
type tickMsg time.Time

// tickCmd returns a tea.Cmd that sends a tickMsg after the given interval.
// An interval of zero fires immediately.
func tickCmd(d time.Duration) tea.Cmd {
	if d == 0 {
		return func() tea.Msg { return tickMsg(time.Now()) }
	}
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}
