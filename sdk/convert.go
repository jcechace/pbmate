package sdk

import (
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
// Returns the zero value for unrecognized statuses.
func convertStatus(s defs.Status) Status {
	parsed, _ := ParseStatus(string(s))
	return parsed
}

// convertBackupType converts a PBM backup type string to an SDK BackupType.
// Returns the zero value for unrecognized types.
func convertBackupType(bt defs.BackupType) BackupType {
	parsed, _ := ParseBackupType(string(bt))
	return parsed
}

// convertCompressionType converts a PBM compression type to an SDK CompressionType.
// Returns the zero value for unrecognized types.
func convertCompressionType(ct compress.CompressionType) CompressionType {
	parsed, _ := ParseCompressionType(string(ct))
	return parsed
}

// convertStorageType converts a PBM storage type to an SDK StorageType.
// Returns the zero value for unrecognized types.
func convertStorageType(st storage.Type) StorageType {
	parsed, _ := ParseStorageType(string(st))
	return parsed
}

// convertConfigName converts a PBM profile/config name to an SDK ConfigName.
// Empty string (PBM's representation of the main config) maps to MainConfig.
func convertConfigName(name string) ConfigName {
	if name == "" {
		return MainConfig
	}
	cn, _ := NewConfigName(name)
	return cn
}
