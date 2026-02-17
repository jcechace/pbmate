package sdk

import (
	"time"

	"github.com/percona/percona-backup-mongodb/pbm/log"
)

// convertLogEntry converts a PBM log.Entry to an SDK LogEntry.
// The structured LogKeys fields are flattened into the Attrs map.
func convertLogEntry(e *log.Entry) LogEntry {
	return LogEntry{
		Timestamp: convertLogTimestamp(e.TS),
		Severity:  convertLogSeverity(e.Severity),
		Message:   e.Msg,
		Attrs:     convertLogAttrs(&e.LogKeys),
	}
}

// convertLogTimestamp converts a Unix timestamp (seconds) to time.Time.
func convertLogTimestamp(ts int64) time.Time {
	if ts == 0 {
		return time.Time{}
	}
	return time.Unix(ts, 0).UTC()
}

// convertLogSeverity converts a PBM log.Severity (int) to an SDK LogSeverity.
func convertLogSeverity(s log.Severity) LogSeverity {
	parsed, _ := ParseLogSeverity(s.String())
	return parsed
}

// convertLogAttrs extracts the structured LogKeys fields into a map.
// Only non-empty values are included.
func convertLogAttrs(keys *log.LogKeys) map[string]any {
	attrs := make(map[string]any)

	if keys.RS != "" {
		attrs[LogKeyReplicaSet] = keys.RS
	}
	if keys.Node != "" {
		attrs[LogKeyNode] = keys.Node
	}
	if keys.Event != "" {
		attrs[LogKeyEvent] = keys.Event
	}
	if keys.ObjName != "" {
		attrs[LogKeyObjName] = keys.ObjName
	}
	if keys.OPID != "" {
		attrs[LogKeyOPID] = keys.OPID
	}
	if keys.Epoch.T != 0 {
		attrs[LogKeyEpoch] = convertTimestamp(keys.Epoch)
	}

	if len(attrs) == 0 {
		return nil
	}
	return attrs
}
