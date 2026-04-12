package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

const (
	// idleInterval is the polling interval when no operations are running.
	idleInterval = 10 * time.Second

	// activeInterval is the polling interval when an operation is in progress.
	activeInterval = 2 * time.Second

	// connectTimeout is the maximum time to wait for a single connection
	// attempt to MongoDB before giving up and scheduling a retry.
	connectTimeout = 10 * time.Second

	// connectRetryMin is the initial delay before retrying a failed connection.
	connectRetryMin = 2 * time.Second

	// connectRetryMax is the maximum delay between connection retry attempts.
	connectRetryMax = 30 * time.Second

	// quitTimeout is how long the "press q again" hint stays active before
	// reverting to normal state.
	quitTimeout = 2 * time.Second

	// actionFlashTimeout is how long action error messages persist in the
	// status bar before auto-clearing. Must be long enough to read.
	actionFlashTimeout = 5 * time.Second
)

// connectBackoff returns the retry delay for the given attempt number (1-based).
// Uses exponential backoff: 2s, 4s, 8s, 16s, 30s, 30s, ...
func connectBackoff(attempt int) time.Duration {
	d := connectRetryMin
	for i := 1; i < attempt; i++ {
		d *= 2
		if d >= connectRetryMax {
			return connectRetryMax
		}
	}
	return d
}

// quitTimeoutMsg signals that the quit-pending window has expired without a
// second q press. The Model clears quitPending and resumes normal operation.
type quitTimeoutMsg struct{}

// quitTimeoutCmd returns a tea.Cmd that sends a quitTimeoutMsg after the
// quit confirmation window elapses.
func quitTimeoutCmd() tea.Cmd {
	return tea.Tick(quitTimeout, func(time.Time) tea.Msg { return quitTimeoutMsg{} })
}

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
