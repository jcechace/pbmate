package tui

import (
	"context"
	"fmt"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// firstErrCollector records the first non-nil error from concurrent
// goroutines. Safe for use from errgroup workers.
type firstErrCollector struct {
	mu  sync.Mutex
	err error
}

// set records err if it is non-nil and no prior error has been recorded.
func (c *firstErrCollector) set(err error) {
	if err != nil {
		c.mu.Lock()
		if c.err == nil {
			c.err = err
		}
		c.mu.Unlock()
	}
}

// connectMsg carries the result of the background SDK connection attempt.
type connectMsg struct {
	client *sdk.Client
	err    error
}

// connectCmd returns a tea.Cmd that connects to PBM in the background.
// Each attempt is bounded by connectTimeout (defined in poll.go).
func connectCmd(uri string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		client, err := sdk.NewClient(ctx,
			sdk.WithMongoURI(uri),
			sdk.WithConnectTimeout(connectTimeout),
		)
		return connectMsg{client: client, err: err}
	}
}

// reconnectMsg signals that the retry delay has elapsed and a new connection
// attempt should be made.
type reconnectMsg struct{}

// reconnectCmd returns a tea.Cmd that waits for the given delay, then sends
// a reconnectMsg to trigger another connection attempt.
func reconnectCmd(delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(time.Time) tea.Msg { return reconnectMsg{} })
}

// actionResultMsg carries the result of any user-initiated action (backup,
// restore, resync, config). The action string identifies the operation for
// flash messages; err is nil on success.
type actionResultMsg struct {
	action string // "start", "cancel", "delete", "restore", "resync", "apply config", etc.
	err    error
}

// drainErr reads one error from the channel without blocking.
// Returns nil if the channel is empty or closed.
func drainErr(errs <-chan error) error {
	if errs == nil {
		return nil
	}
	select {
	case err := <-errs:
		return err
	default:
		return nil
	}
}

// formatStorageSummary returns a compact string describing the storage config.
func formatStorageSummary(s sdk.StorageConfig) string {
	if s.Type.IsZero() {
		return ""
	}
	if s.Path != "" {
		return fmt.Sprintf("%s %s", s.Type, s.Path)
	}
	return s.Type.String()
}
