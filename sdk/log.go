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
	//	// Include debug-level entries for a specific replica set.
	//	entries, err := client.Logs.Get(ctx, sdk.GetLogsOptions{
	//	    LogFilter: sdk.LogFilter{
	//	        Severity:   sdk.LogSeverityDebug,
	//	        ReplicaSet: "rs0",
	//	    },
	//	    Limit: 100,
	//	})
	Get(ctx context.Context, opts GetLogsOptions) ([]LogEntry, error)

	// Follow streams log entries as they arrive via a MongoDB tailable cursor.
	// The entries channel receives new log entries in real time. Both channels
	// are closed when the context is cancelled. The error channel receives at
	// most one error if the stream fails.
	//
	// Example:
	//
	//	ctx, cancel := context.WithCancel(ctx)
	//	defer cancel()
	//	entries, errs := client.Logs.Follow(ctx, sdk.FollowOptions{
	//	    LogFilter: sdk.LogFilter{ReplicaSet: "rs0"},
	//	})
	//	for entry := range entries {
	//	    fmt.Printf("[%s] %s\n", entry.Severity, entry.Message)
	//	}
	Follow(ctx context.Context, opts FollowOptions) (<-chan LogEntry, <-chan error)
}

// LogFilter controls which log entries to include. All fields are optional;
// zero values are ignored. When multiple fields are set, they are combined
// with AND semantics (all conditions must match).
type LogFilter struct {
	// Severity is the minimum severity level to include. PBM's severity filter
	// includes all entries at the given level and above (e.g. Info includes
	// Fatal, Error, Warning, and Info). Zero value defaults to Info.
	Severity LogSeverity

	// ReplicaSet filters entries by replica set name.
	ReplicaSet string

	// Node filters entries by node identifier (hostname:port).
	Node string

	// Event filters entries by event type (e.g. "backup", "restore").
	Event string

	// ObjectName filters entries by object name (e.g. backup name).
	ObjectName string

	// OPID filters entries by operation ID.
	OPID string

	// Epoch filters entries by the PBM epoch timestamp.
	Epoch Timestamp
}

// GetLogsOptions controls filtering and pagination for log retrieval.
type GetLogsOptions struct {
	LogFilter

	// Limit is the maximum number of log entries to return. Zero means no limit.
	Limit int

	// TimeMin limits results to entries at or after this time. Zero value means
	// no lower bound.
	TimeMin time.Time

	// TimeMax limits results to entries at or before this time. Zero value means
	// no upper bound.
	TimeMax time.Time
}

// FollowOptions controls filtering for log streaming.
type FollowOptions struct {
	LogFilter

	// TimeMin limits the tailable cursor to entries at or after this time.
	// When set, historical entries older than TimeMin are skipped — only
	// entries created at or after this timestamp are delivered. This is
	// useful to avoid replaying entries the caller already has.
	// Zero value means no lower bound (all matching entries are delivered).
	TimeMin time.Time
}

// LogEntry represents a single PBM log entry from the centralized log
// collection. Entries are written by PBM agents on all nodes.
type LogEntry struct {
	Timestamp time.Time      // when the log entry was created (UTC)
	Severity  LogSeverity    // log level: D(ebug), I(nfo), W(arning), E(rror), F(atal)
	Message   string         // human-readable log message
	Attrs     map[string]any // structured attributes (rs, node, event, opid, etc.)
}
