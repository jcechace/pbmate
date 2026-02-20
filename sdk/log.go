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

// LogService provides access to PBM's stored logs. Logs are stored in
// a MongoDB collection and include entries from all PBM agents.
//
// Example — fetch recent logs and stream new entries:
//
//	entries, err := client.Logs.Get(ctx, sdk.GetLogsOptions{Limit: 50})
//	for _, e := range entries {
//	    fmt.Printf("[%s] %s: %s\n", e.Severity, e.Timestamp.UTC(), e.Message)
//	}
//
//	// Stream new entries until context is cancelled.
//	ch, errs := client.Logs.Follow(ctx, sdk.FollowOptions{})
//	for entry := range ch {
//	    fmt.Println(entry.Message)
//	}
//	if err := <-errs; err != nil {
//	    log.Printf("follow ended: %v", err)
//	}
type LogService interface {
	// Get returns log entries, most recent first.
	//
	// Example:
	//
	//	entries, err := client.Logs.Get(ctx, sdk.GetLogsOptions{Limit: 100})
	//
	//	// Include debug-level entries.
	//	entries, err := client.Logs.Get(ctx, sdk.GetLogsOptions{
	//	    Limit:    100,
	//	    Severity: sdk.LogSeverityDebug,
	//	})
	Get(ctx context.Context, opts GetLogsOptions) ([]LogEntry, error)

	// Follow streams log entries as they arrive via a MongoDB change stream.
	// The entries channel receives new log entries in real time. Both channels
	// are closed when the context is cancelled. The error channel receives at
	// most one error if the stream fails.
	//
	// Example:
	//
	//	ctx, cancel := context.WithCancel(ctx)
	//	defer cancel()
	//	entries, errs := client.Logs.Follow(ctx, sdk.FollowOptions{})
	//	for entry := range entries {
	//	    fmt.Printf("[%s] %s\n", entry.Severity, entry.Message)
	//	}
	Follow(ctx context.Context, opts FollowOptions) (<-chan LogEntry, <-chan error)
}

// GetLogsOptions controls filtering and pagination for log retrieval.
type GetLogsOptions struct {
	// Limit is the maximum number of log entries to return. Zero means no limit.
	Limit int64

	// Severity is the minimum severity level to include. PBM's severity filter
	// includes all entries at the given level and above (e.g. Info includes
	// Fatal, Error, Warning, and Info). Zero value defaults to Info.
	Severity LogSeverity
}

// FollowOptions controls filtering for log streaming.
type FollowOptions struct {
	// Severity is the minimum severity level to include. PBM's severity filter
	// includes all entries at the given level and above (e.g. Info includes
	// Fatal, Error, Warning, and Info). Zero value defaults to Info.
	Severity LogSeverity
}

// LogEntry represents a single PBM log entry from the centralized log
// collection. Entries are written by PBM agents on all nodes.
type LogEntry struct {
	Timestamp time.Time      // when the log entry was created (UTC)
	Severity  LogSeverity    // log level: D(ebug), I(nfo), W(arning), E(rror), F(atal)
	Message   string         // human-readable log message
	Attrs     map[string]any // structured attributes (rs, node, event, opid, etc.)
}
