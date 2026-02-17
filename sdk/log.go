package sdk

import (
	"context"
	"time"
)

// Well-known log attribute keys.
const (
	LogKeyReplicaSet = "rs"
	LogKeyNode       = "node"
	LogKeyEvent      = "event"
	LogKeyObjName    = "objName"
	LogKeyOPID       = "opid"
	LogKeyEpoch      = "epoch"
)

// LogService provides access to PBM's stored logs.
type LogService interface {
	// Get returns log entries, most recent first.
	// Limit controls how many entries to return. Zero means no limit.
	Get(ctx context.Context, limit int64) ([]LogEntry, error)

	// Follow streams log entries as they arrive.
	// The returned channels are closed when the context is cancelled.
	Follow(ctx context.Context) (<-chan LogEntry, <-chan error)
}

// LogEntry represents a single PBM log entry.
type LogEntry struct {
	Timestamp time.Time
	Severity  LogSeverity
	Message   string
	Attrs     map[string]any
}
