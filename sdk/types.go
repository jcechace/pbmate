package sdk

import "time"

// Timestamp represents a MongoDB cluster timestamp.
// It combines a Unix seconds value with an ordinal increment,
// which is essential for PITR where exact oplog positions matter.
type Timestamp struct {
	T uint32 // seconds since Unix epoch
	I uint32 // ordinal increment within the second
}

// Time converts the Timestamp to a standard time.Time.
func (ts Timestamp) Time() time.Time {
	return time.Unix(int64(ts.T), 0)
}

// IsZero reports whether the timestamp is the zero value.
func (ts Timestamp) IsZero() bool {
	return ts.T == 0 && ts.I == 0
}

// Status represents the lifecycle state of a PBM operation.
type Status string

const (
	StatusInit       Status = "init"
	StatusReady      Status = "ready"
	StatusStarting   Status = "starting"
	StatusRunning    Status = "running"
	StatusDumpDone   Status = "dumpDone"
	StatusCopyReady  Status = "copyReady"
	StatusCopyDone   Status = "copyDone"
	StatusPartlyDone Status = "partlyDone"
	StatusDone       Status = "done"
	StatusCancelled  Status = "cancelled"
	StatusError      Status = "error"
)

// BackupType represents the type of backup.
type BackupType string

const (
	BackupLogical     BackupType = "logical"
	BackupPhysical    BackupType = "physical"
	BackupIncremental BackupType = "incremental"
	BackupExternal    BackupType = "external"
)

// CompressionType represents the compression algorithm used.
type CompressionType string

const (
	CompressionNone   CompressionType = "none"
	CompressionGZIP   CompressionType = "gzip"
	CompressionPGZIP  CompressionType = "pgzip"
	CompressionSNAPPY CompressionType = "snappy"
	CompressionLZ4    CompressionType = "lz4"
	CompressionS2     CompressionType = "s2"
	CompressionZSTD   CompressionType = "zstandard"
)

// StorageType represents the type of backup storage backend.
type StorageType string

const (
	StorageS3         StorageType = "s3"
	StorageMinio      StorageType = "minio"
	StorageGCS        StorageType = "gcs"
	StorageAzure      StorageType = "azure"
	StorageFilesystem StorageType = "filesystem"
)

// NodeRole represents the role of a MongoDB node in a replica set.
type NodeRole string

const (
	RolePrimary   NodeRole = "P"
	RoleSecondary NodeRole = "S"
	RoleArbiter   NodeRole = "A"
	RoleHidden    NodeRole = "H"
	RoleDelayed   NodeRole = "D"
)

// LogSeverity represents the severity level of a log entry.
type LogSeverity string

const (
	LogDebug   LogSeverity = "D"
	LogInfo    LogSeverity = "I"
	LogWarning LogSeverity = "W"
	LogError   LogSeverity = "E"
	LogFatal   LogSeverity = "F"
)
