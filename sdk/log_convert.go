package sdk

import (
	"log/slog"

	"github.com/percona/percona-backup-mongodb/pbm/log"
)

// convertLogEntry converts a PBM log.Entry to an SDK LogEntry.
// The structured LogKeys fields are flattened into the Attrs map.
func convertLogEntry(e *log.Entry) LogEntry {
	return LogEntry{
		Timestamp: convertUnixToTime(e.TS),
		Severity:  convertLogSeverity(e.Severity),
		Message:   e.Msg,
		Attrs:     convertLogAttrs(&e.LogKeys),
	}
}

// convertLogSeverity converts a PBM log.Severity (int) to an SDK LogSeverity.
func convertLogSeverity(s log.Severity) LogSeverity {
	parsed, err := ParseLogSeverity(s.String())
	if err != nil {
		slog.Warn("unknown PBM log severity", "value", s.String())
	}
	return parsed
}

// convertLogSeverityToInternal converts an SDK LogSeverity to PBM's
// log.Severity. Zero value (unset) defaults to Info.
func convertLogSeverityToInternal(s LogSeverity) log.Severity {
	if s.IsZero() {
		return log.Info
	}

	switch {
	case s.Equal(LogSeverityFatal):
		return log.Fatal
	case s.Equal(LogSeverityError):
		return log.Error
	case s.Equal(LogSeverityWarning):
		return log.Warning
	case s.Equal(LogSeverityInfo):
		return log.Info
	case s.Equal(LogSeverityDebug):
		return log.Debug
	default:
		return log.Info
	}
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
