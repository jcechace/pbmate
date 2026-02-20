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
//	entries, err := client.Logs.Get(ctx, 50)
//	for _, e := range entries {
//	    fmt.Printf("[%s] %s: %s\n", e.Severity, e.Timestamp.UTC(), e.Message)
//	}
//
//	// Stream new entries until context is cancelled.
//	ch, errs := client.Logs.Follow(ctx)
//	for entry := range ch {
//	    fmt.Println(entry.Message)
//	}
//	if err := <-errs; err != nil {
//	    log.Printf("follow ended: %v", err)
//	}
type LogService interface {
	// Get returns log entries, most recent first.
	// Limit controls how many entries to return. Zero means no limit.
	//
	// Example:
	//
	//	entries, err := client.Logs.Get(ctx, 100)
	Get(ctx context.Context, limit int64) ([]LogEntry, error)

	// Follow streams log entries as they arrive via a MongoDB change stream.
	// The entries channel receives new log entries in real time. Both channels
	// are closed when the context is cancelled. The error channel receives at
	// most one error if the stream fails.
	//
	// Example:
	//
	//	ctx, cancel := context.WithCancel(ctx)
	//	defer cancel()
	//	entries, errs := client.Logs.Follow(ctx)
	//	for entry := range entries {
	//	    fmt.Printf("[%s] %s\n", entry.Severity, entry.Message)
	//	}
	Follow(ctx context.Context) (<-chan LogEntry, <-chan error)
}

// LogEntry represents a single PBM log entry from the centralized log
// collection. Entries are written by PBM agents on all nodes.
type LogEntry struct {
	Timestamp time.Time      // when the log entry was created (UTC)
	Severity  LogSeverity    // log level: D(ebug), I(nfo), W(arning), E(rror), F(atal)
	Message   string         // human-readable log message
	Attrs     map[string]any // structured attributes (rs, node, event, opid, etc.)
}
