package sdk

import (
	"context"
	"time"
)

// LogService provides access to PBM's stored logs.
type LogService interface {
	// Query returns log entries matching the given options.
	Query(ctx context.Context, opts LogQuery) ([]LogEntry, error)

	// Follow streams log entries matching the given options.
	// The returned channels are closed when the context is cancelled.
	Follow(ctx context.Context, opts LogQuery) (<-chan LogEntry, <-chan error)
}

// LogQuery controls filtering for log queries.
type LogQuery struct {
	// Severity is the minimum severity level. Empty means all levels.
	Severity LogSeverity

	// Event filters by event type (e.g. "backup", "restore", "pitr").
	Event string

	// OPID filters by operation ID.
	OPID string

	// Limit is the maximum number of entries to return. Zero means no limit.
	Limit int64
}

// LogEntry represents a single PBM log entry.
type LogEntry struct {
	Timestamp  time.Time
	Severity   LogSeverity
	ReplicaSet string
	Node       string
	Event      string
	OPID       string
	Message    string
}
