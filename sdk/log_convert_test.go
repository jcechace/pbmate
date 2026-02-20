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

func TestConvertLogTimestamp(t *testing.T) {
	t.Run("valid timestamp", func(t *testing.T) {
		result := convertLogTimestamp(1700000000)
		assert.Equal(t, time.Unix(1700000000, 0).UTC(), result)
	})

	t.Run("zero timestamp", func(t *testing.T) {
		result := convertLogTimestamp(0)
		assert.True(t, result.IsZero())
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
