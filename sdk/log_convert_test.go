package sdk

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/percona/percona-backup-mongodb/pbm/log"
)

func TestConvertLogEntry(t *testing.T) {
	t.Run("full entry", func(t *testing.T) {
		e := &log.Entry{
			TS:  1700000000,
			Msg: "backup started",
			LogKeys: log.LogKeys{
				Severity: log.Info,
				RS:       "rs0",
				Node:     "mongo1:27018",
				Event:    "backup",
				ObjName:  "2024-01-15T10:00:00Z",
				OPID:     "abc123",
				Epoch:    primitive.Timestamp{T: 1699999000, I: 1},
			},
		}

		result := convertLogEntry(e)

		assert.Equal(t, time.Unix(1700000000, 0).UTC(), result.Timestamp)
		assert.Equal(t, LogSeverityInfo, result.Severity)
		assert.Equal(t, "backup started", result.Message)

		assert.Equal(t, "rs0", result.Attrs[LogKeyReplicaSet])
		assert.Equal(t, "mongo1:27018", result.Attrs[LogKeyNode])
		assert.Equal(t, "backup", result.Attrs[LogKeyEvent])
		assert.Equal(t, "2024-01-15T10:00:00Z", result.Attrs[LogKeyObjName])
		assert.Equal(t, "abc123", result.Attrs[LogKeyOPID])
		assert.Equal(t, Timestamp{T: 1699999000, I: 1}, result.Attrs[LogKeyEpoch])
	})

	t.Run("minimal entry", func(t *testing.T) {
		e := &log.Entry{
			TS:  1700000000,
			Msg: "simple message",
			LogKeys: log.LogKeys{
				Severity: log.Error,
			},
		}

		result := convertLogEntry(e)

		assert.Equal(t, LogSeverityError, result.Severity)
		assert.Equal(t, "simple message", result.Message)
		assert.Nil(t, result.Attrs)
	})
}

func TestConvertLogSeverity(t *testing.T) {
	tests := []struct {
		name     string
		input    log.Severity
		expected LogSeverity
	}{
		{"fatal", log.Fatal, LogSeverityFatal},
		{"error", log.Error, LogSeverityError},
		{"warning", log.Warning, LogSeverityWarning},
		{"info", log.Info, LogSeverityInfo},
		{"debug", log.Debug, LogSeverityDebug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertLogSeverity(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertLogSeverityToInternal(t *testing.T) {
	tests := []struct {
		name     string
		input    LogSeverity
		expected log.Severity
	}{
		{"fatal", LogSeverityFatal, log.Fatal},
		{"error", LogSeverityError, log.Error},
		{"warning", LogSeverityWarning, log.Warning},
		{"info", LogSeverityInfo, log.Info},
		{"debug", LogSeverityDebug, log.Debug},
		{"zero defaults to info", LogSeverity{}, log.Info},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertLogSeverityToInternal(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertLogFilter(t *testing.T) {
	t.Run("all fields set", func(t *testing.T) {
		f := &LogFilter{
			Severity:   LogSeverityDebug,
			ReplicaSet: "rs0",
			Node:       "mongo1:27018",
			Event:      "backup",
			ObjectName: "2024-01-15T10:00:00Z",
			OPID:       "abc123",
			Epoch:      Timestamp{T: 1700000000, I: 1},
		}

		keys := convertLogFilter(f)

		assert.Equal(t, log.Debug, keys.Severity)
		assert.Equal(t, "rs0", keys.RS)
		assert.Equal(t, "mongo1:27018", keys.Node)
		assert.Equal(t, "backup", keys.Event)
		assert.Equal(t, "2024-01-15T10:00:00Z", keys.ObjName)
		assert.Equal(t, "abc123", keys.OPID)
		assert.Equal(t, primitive.Timestamp{T: 1700000000, I: 1}, keys.Epoch)
	})

	t.Run("zero filter defaults severity to info", func(t *testing.T) {
		f := &LogFilter{}

		keys := convertLogFilter(f)

		assert.Equal(t, log.Info, keys.Severity)
		assert.Empty(t, keys.RS)
		assert.Empty(t, keys.Node)
		assert.Empty(t, keys.Event)
		assert.Empty(t, keys.ObjName)
		assert.Empty(t, keys.OPID)
		assert.Equal(t, primitive.Timestamp{}, keys.Epoch)
	})

	t.Run("partial fields", func(t *testing.T) {
		f := &LogFilter{
			Severity:   LogSeverityWarning,
			ReplicaSet: "rs1",
			OPID:       "op42",
		}

		keys := convertLogFilter(f)

		assert.Equal(t, log.Warning, keys.Severity)
		assert.Equal(t, "rs1", keys.RS)
		assert.Empty(t, keys.Node)
		assert.Empty(t, keys.Event)
		assert.Empty(t, keys.ObjName)
		assert.Equal(t, "op42", keys.OPID)
	})
}

func TestConvertLogRequest(t *testing.T) {
	t.Run("with time range", func(t *testing.T) {
		f := &LogFilter{
			Severity:   LogSeverityError,
			ReplicaSet: "rs0",
		}
		tMin := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		tMax := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

		req := convertLogRequest(f, tMin, tMax)

		assert.Equal(t, log.Error, req.Severity)
		assert.Equal(t, "rs0", req.RS)
		assert.Equal(t, tMin, req.TimeMin)
		assert.Equal(t, tMax, req.TimeMax)
	})

	t.Run("without time range", func(t *testing.T) {
		f := &LogFilter{OPID: "abc"}

		req := convertLogRequest(f, time.Time{}, time.Time{})

		assert.Equal(t, "abc", req.OPID)
		assert.True(t, req.TimeMin.IsZero())
		assert.True(t, req.TimeMax.IsZero())
	})
}

func TestConvertLogAttrs(t *testing.T) {
	t.Run("all fields set", func(t *testing.T) {
		keys := &log.LogKeys{
			Severity: log.Info,
			RS:       "rs0",
			Node:     "mongo1:27018",
			Event:    "backup",
			ObjName:  "backup-name",
			OPID:     "abc123",
			Epoch:    primitive.Timestamp{T: 1700000000, I: 1},
		}

		attrs := convertLogAttrs(keys)

		assert.Len(t, attrs, 6)
		assert.Equal(t, "rs0", attrs[LogKeyReplicaSet])
		assert.Equal(t, "mongo1:27018", attrs[LogKeyNode])
		assert.Equal(t, "backup", attrs[LogKeyEvent])
		assert.Equal(t, "backup-name", attrs[LogKeyObjName])
		assert.Equal(t, "abc123", attrs[LogKeyOPID])
		assert.Equal(t, Timestamp{T: 1700000000, I: 1}, attrs[LogKeyEpoch])
	})

	t.Run("no fields set", func(t *testing.T) {
		keys := &log.LogKeys{
			Severity: log.Info,
		}

		attrs := convertLogAttrs(keys)

		assert.Nil(t, attrs)
	})

	t.Run("partial fields", func(t *testing.T) {
		keys := &log.LogKeys{
			Severity: log.Info,
			RS:       "rs0",
			OPID:     "abc123",
		}

		attrs := convertLogAttrs(keys)

		assert.Len(t, attrs, 2)
		assert.Equal(t, "rs0", attrs[LogKeyReplicaSet])
		assert.Equal(t, "abc123", attrs[LogKeyOPID])
	})

	t.Run("epoch with zero T", func(t *testing.T) {
		keys := &log.LogKeys{
			Severity: log.Info,
			RS:       "rs0",
			Epoch:    primitive.Timestamp{T: 0, I: 0},
		}

		attrs := convertLogAttrs(keys)

		// Epoch with T=0 should not be included.
		assert.Len(t, attrs, 1)
		assert.NotContains(t, attrs, LogKeyEpoch)
	})
}
