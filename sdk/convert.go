package sdk

import (
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/percona/percona-backup-mongodb/pbm/compress"
	"github.com/percona/percona-backup-mongodb/pbm/defs"
	"github.com/percona/percona-backup-mongodb/pbm/storage"
)

// convertTimestamp converts a BSON primitive.Timestamp to an SDK Timestamp.
func convertTimestamp(ts primitive.Timestamp) Timestamp {
	return Timestamp{T: ts.T, I: ts.I}
}

// convertTimestampToPBM converts an SDK Timestamp to a BSON primitive.Timestamp.
func convertTimestampToPBM(ts Timestamp) primitive.Timestamp {
	return primitive.Timestamp{T: ts.T, I: ts.I}
}

// convertUnixToTime converts a Unix timestamp in seconds to time.Time.
// Returns the zero time for zero input.
func convertUnixToTime(unix int64) time.Time {
	if unix == 0 {
		return time.Time{}
	}
	return time.Unix(unix, 0).UTC()
}

// convertStatus converts a PBM status string to an SDK Status.
// Returns the zero value for empty input. Logs a warning for unrecognized statuses.
func convertStatus(s defs.Status) Status {
	if s == "" {
		return Status{}
	}
	parsed, err := ParseStatus(string(s))
	if err != nil {
		slog.Warn("unknown PBM status", "value", string(s))
	}
	return parsed
}

// convertBackupType converts a PBM backup type string to an SDK BackupType.
// Returns the zero value for empty input. Logs a warning for unrecognized types.
func convertBackupType(bt defs.BackupType) BackupType {
	if bt == "" {
		return BackupType{}
	}
	parsed, err := ParseBackupType(string(bt))
	if err != nil {
		slog.Warn("unknown PBM backup type", "value", string(bt))
	}
	return parsed
}

// convertCompressionType converts a PBM compression type to an SDK CompressionType.
// Returns the zero value for empty input. Logs a warning for unrecognized types.
func convertCompressionType(ct compress.CompressionType) CompressionType {
	if ct == "" {
		return CompressionType{}
	}
	parsed, err := ParseCompressionType(string(ct))
	if err != nil {
		slog.Warn("unknown PBM compression type", "value", string(ct))
	}
	return parsed
}

// convertStorageType converts a PBM storage type to an SDK StorageType.
// Returns the zero value for empty input. Logs a warning for unrecognized types.
func convertStorageType(st storage.Type) StorageType {
	if st == "" {
		return StorageType{}
	}
	parsed, err := ParseStorageType(string(st))
	if err != nil {
		slog.Warn("unknown PBM storage type", "value", string(st))
	}
	return parsed
}

// convertSlice converts a slice of input values to a slice of output values
// using the provided conversion function. Returns nil for empty input.
func convertSlice[In, Out any](items []In, fn func(In) Out) []Out {
	if len(items) == 0 {
		return nil
	}
	result := make([]Out, len(items))
	for i := range items {
		result[i] = fn(items[i])
	}
	return result
}

// isLockStale reports whether a lock heartbeat is stale relative to clusterTime.
// A lock is considered stale when its heartbeat is older than PBM's stale frame
// threshold. Both callers — CheckLock and RunningOperations — must agree on
// this definition to avoid inconsistencies.
func isLockStale(heartbeatT, clusterTimeT uint32) bool {
	return heartbeatT+defs.StaleFrameSec < clusterTimeT
}

// convertConfigName converts a PBM profile/config name to an SDK ConfigName.
// Empty string (PBM's representation of the main config) maps to MainConfig.
func convertConfigName(name string) ConfigName {
	if name == "" {
		return MainConfig
	}
	cn, err := NewConfigName(name)
	if err != nil {
		slog.Warn("invalid PBM config name", "value", name)
	}
	return cn
}
