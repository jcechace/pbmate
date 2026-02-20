package sdk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// defaultPollInterval is used when the caller does not specify a poll interval.
const defaultPollInterval = time.Second

// waitParams holds the function-based callbacks for waitForTerminal.
// Using functions instead of an interface avoids adding methods to domain types
// solely to satisfy an internal abstraction.
type waitParams[T any] struct {
	get        func(ctx context.Context, name string) (T, error)
	status     func(T) Status
	errMsg     func(T) string
	onProgress func(T)
	log        *slog.Logger
	entity     string // "backup" or "restore", used in log messages and error wrapping
}

// waitForTerminal polls until the named entity reaches a terminal status or
// the context is cancelled. Returns the final entity and nil on success, or
// the entity and an *OperationError on failure (StatusError/StatusPartlyDone).
// On context cancellation, returns the last observed entity (may be zero) and
// ctx.Err().
func waitForTerminal[T any](ctx context.Context, name string, interval time.Duration, p waitParams[T]) (T, error) {
	if interval == 0 {
		interval = defaultPollInterval
	}

	var last T
	timer := time.NewTimer(0) // fires immediately for first check
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return last, ctx.Err()
		case <-timer.C:
		}

		val, err := p.get(ctx, name)
		if err != nil {
			if ctx.Err() != nil {
				return last, ctx.Err()
			}
			if !errors.Is(err, ErrNotFound) {
				return last, fmt.Errorf("wait for %s %q: %w", p.entity, name, err)
			}
			p.log.DebugContext(ctx, p.entity+" not found yet, retrying", "name", name)
		} else {
			last = val
			st := p.status(val)
			p.log.DebugContext(ctx, "polling "+p.entity+" status", "name", name, "status", st)

			if p.onProgress != nil {
				p.onProgress(val)
			}

			if st.IsTerminal() {
				p.log.InfoContext(ctx, p.entity+" reached terminal status", "name", name, "status", st)
				if st.Equal(StatusError) || st.Equal(StatusPartlyDone) {
					return val, &OperationError{Name: name, Message: p.errMsg(val)}
				}
				return val, nil
			}
		}

		timer.Reset(interval)
	}
}
