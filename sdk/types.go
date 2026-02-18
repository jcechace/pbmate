package sdk

import (
	"fmt"
	"time"
)

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

// =============================================================================
// Status
// =============================================================================

// Status represents the lifecycle state of a PBM operation.
type Status struct {
	value string
}

var (
	StatusInit           = newStatus("init")
	StatusReady          = newStatus("ready")
	StatusStarting       = newStatus("starting")
	StatusRunning        = newStatus("running")
	StatusDumpDone       = newStatus("dumpDone")
	StatusCopyReady      = newStatus("copyReady")
	StatusCopyDone       = newStatus("copyDone")
	StatusPartlyDone     = newStatus("partlyDone")
	StatusDone           = newStatus("done")
	StatusCancelled      = newStatus("canceled")
	StatusError          = newStatus("error")
	StatusDown           = newStatus("down")           // physical restore: mongod processes stopped
	StatusCleanupCluster = newStatus("cleanupCluster") // restore: clearing existing data
)

var statuses = make(map[string]Status)

func newStatus(s string) Status {
	st := Status{s}
	statuses[s] = st
	return st
}

// String returns the string representation of the status.
func (s Status) String() string { return s.value }

// IsZero reports whether the status is the zero value (unset).
func (s Status) IsZero() bool { return s.value == "" }

// Equal reports whether two status values are identical.
func (s Status) Equal(other Status) bool { return s.value == other.value }

// MarshalText implements encoding.TextMarshaler.
func (s Status) MarshalText() ([]byte, error) { return []byte(s.value), nil }

// UnmarshalText implements encoding.TextUnmarshaler.
func (s *Status) UnmarshalText(b []byte) error {
	parsed, err := ParseStatus(string(b))
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

// ParseStatus parses a string into a Status.
// Returns an error if the value is not a known status.
func ParseStatus(value string) (Status, error) {
	st, ok := statuses[value]
	if !ok {
		return Status{}, fmt.Errorf("invalid status %q", value)
	}
	return st, nil
}

// =============================================================================
// BackupType
// =============================================================================

// BackupType represents the type of backup.
type BackupType struct {
	value string
}

var (
	BackupTypeLogical     = newBackupType("logical")
	BackupTypePhysical    = newBackupType("physical")
	BackupTypeIncremental = newBackupType("incremental")
	BackupTypeExternal    = newBackupType("external")
)

var backupTypes = make(map[string]BackupType)

func newBackupType(s string) BackupType {
	bt := BackupType{s}
	backupTypes[s] = bt
	return bt
}

// String returns the string representation of the backup type.
func (b BackupType) String() string { return b.value }

// IsZero reports whether the backup type is the zero value (unset).
func (b BackupType) IsZero() bool { return b.value == "" }

// Equal reports whether two backup type values are identical.
func (b BackupType) Equal(other BackupType) bool { return b.value == other.value }

// MarshalText implements encoding.TextMarshaler.
func (b BackupType) MarshalText() ([]byte, error) { return []byte(b.value), nil }

// UnmarshalText implements encoding.TextUnmarshaler.
func (b *BackupType) UnmarshalText(data []byte) error {
	parsed, err := ParseBackupType(string(data))
	if err != nil {
		return err
	}
	*b = parsed
	return nil
}

// ParseBackupType parses a string into a BackupType.
// Returns an error if the value is not a known backup type.
func ParseBackupType(value string) (BackupType, error) {
	bt, ok := backupTypes[value]
	if !ok {
		return BackupType{}, fmt.Errorf("invalid backup type %q", value)
	}
	return bt, nil
}

// =============================================================================
// CompressionType
// =============================================================================

// CompressionType represents the compression algorithm used.
type CompressionType struct {
	value string
}

var (
	CompressionTypeNone   = newCompressionType("none")
	CompressionTypeGZIP   = newCompressionType("gzip")
	CompressionTypePGZIP  = newCompressionType("pgzip")
	CompressionTypeSNAPPY = newCompressionType("snappy")
	CompressionTypeLZ4    = newCompressionType("lz4")
	CompressionTypeS2     = newCompressionType("s2")
	CompressionTypeZSTD   = newCompressionType("zstd")
)

var compressionTypes = make(map[string]CompressionType)

func newCompressionType(s string) CompressionType {
	ct := CompressionType{s}
	compressionTypes[s] = ct
	return ct
}

// String returns the string representation of the compression type.
func (c CompressionType) String() string { return c.value }

// IsZero reports whether the compression type is the zero value (unset).
func (c CompressionType) IsZero() bool { return c.value == "" }

// Equal reports whether two compression type values are identical.
func (c CompressionType) Equal(other CompressionType) bool { return c.value == other.value }

// MarshalText implements encoding.TextMarshaler.
func (c CompressionType) MarshalText() ([]byte, error) { return []byte(c.value), nil }

// UnmarshalText implements encoding.TextUnmarshaler.
func (c *CompressionType) UnmarshalText(data []byte) error {
	parsed, err := ParseCompressionType(string(data))
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}

// ParseCompressionType parses a string into a CompressionType.
// Returns an error if the value is not a known compression type.
func ParseCompressionType(value string) (CompressionType, error) {
	ct, ok := compressionTypes[value]
	if !ok {
		return CompressionType{}, fmt.Errorf("invalid compression type %q", value)
	}
	return ct, nil
}

// =============================================================================
// StorageType
// =============================================================================

// StorageType represents the type of backup storage backend.
type StorageType struct {
	value string
}

var (
	StorageTypeS3         = newStorageType("s3")
	StorageTypeMinio      = newStorageType("minio")
	StorageTypeGCS        = newStorageType("gcs")
	StorageTypeAzure      = newStorageType("azure")
	StorageTypeFilesystem = newStorageType("filesystem")
	StorageTypeBlackhole  = newStorageType("blackhole")
	StorageTypeOSS        = newStorageType("oss")
)

var storageTypes = make(map[string]StorageType)

func newStorageType(s string) StorageType {
	st := StorageType{s}
	storageTypes[s] = st
	return st
}

// String returns the string representation of the storage type.
func (s StorageType) String() string { return s.value }

// IsZero reports whether the storage type is the zero value (unset).
func (s StorageType) IsZero() bool { return s.value == "" }

// Equal reports whether two storage type values are identical.
func (s StorageType) Equal(other StorageType) bool { return s.value == other.value }

// MarshalText implements encoding.TextMarshaler.
func (s StorageType) MarshalText() ([]byte, error) { return []byte(s.value), nil }

// UnmarshalText implements encoding.TextUnmarshaler.
func (s *StorageType) UnmarshalText(data []byte) error {
	parsed, err := ParseStorageType(string(data))
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

// ParseStorageType parses a string into a StorageType.
// Returns an error if the value is not a known storage type.
func ParseStorageType(value string) (StorageType, error) {
	st, ok := storageTypes[value]
	if !ok {
		return StorageType{}, fmt.Errorf("invalid storage type %q", value)
	}
	return st, nil
}

// =============================================================================
// NodeRole
// =============================================================================

// NodeRole represents the role of a MongoDB node in a replica set.
type NodeRole struct {
	value string
}

var (
	NodeRolePrimary   = newNodeRole("P")
	NodeRoleSecondary = newNodeRole("S")
	NodeRoleArbiter   = newNodeRole("A")
	NodeRoleHidden    = newNodeRole("H")
	NodeRoleDelayed   = newNodeRole("D")
)

var nodeRoles = make(map[string]NodeRole)

func newNodeRole(s string) NodeRole {
	nr := NodeRole{s}
	nodeRoles[s] = nr
	return nr
}

// String returns the string representation of the node role.
func (n NodeRole) String() string { return n.value }

// IsZero reports whether the node role is the zero value (unset).
func (n NodeRole) IsZero() bool { return n.value == "" }

// Equal reports whether two node role values are identical.
func (n NodeRole) Equal(other NodeRole) bool { return n.value == other.value }

// MarshalText implements encoding.TextMarshaler.
func (n NodeRole) MarshalText() ([]byte, error) { return []byte(n.value), nil }

// UnmarshalText implements encoding.TextUnmarshaler.
func (n *NodeRole) UnmarshalText(data []byte) error {
	parsed, err := ParseNodeRole(string(data))
	if err != nil {
		return err
	}
	*n = parsed
	return nil
}

// ParseNodeRole parses a string into a NodeRole.
// Returns an error if the value is not a known node role.
func ParseNodeRole(value string) (NodeRole, error) {
	nr, ok := nodeRoles[value]
	if !ok {
		return NodeRole{}, fmt.Errorf("invalid node role %q", value)
	}
	return nr, nil
}

// =============================================================================
// LogSeverity
// =============================================================================

// LogSeverity represents the severity level of a log entry.
type LogSeverity struct {
	value string
}

var (
	LogSeverityDebug   = newLogSeverity("D")
	LogSeverityInfo    = newLogSeverity("I")
	LogSeverityWarning = newLogSeverity("W")
	LogSeverityError   = newLogSeverity("E")
	LogSeverityFatal   = newLogSeverity("F")
)

var logSeverities = make(map[string]LogSeverity)

func newLogSeverity(s string) LogSeverity {
	ls := LogSeverity{s}
	logSeverities[s] = ls
	return ls
}

// String returns the string representation of the log severity.
func (l LogSeverity) String() string { return l.value }

// IsZero reports whether the log severity is the zero value (unset).
func (l LogSeverity) IsZero() bool { return l.value == "" }

// Equal reports whether two log severity values are identical.
func (l LogSeverity) Equal(other LogSeverity) bool { return l.value == other.value }

// MarshalText implements encoding.TextMarshaler.
func (l LogSeverity) MarshalText() ([]byte, error) { return []byte(l.value), nil }

// UnmarshalText implements encoding.TextUnmarshaler.
func (l *LogSeverity) UnmarshalText(data []byte) error {
	parsed, err := ParseLogSeverity(string(data))
	if err != nil {
		return err
	}
	*l = parsed
	return nil
}

// ParseLogSeverity parses a string into a LogSeverity.
// Returns an error if the value is not a known log severity.
func ParseLogSeverity(value string) (LogSeverity, error) {
	ls, ok := logSeverities[value]
	if !ok {
		return LogSeverity{}, fmt.Errorf("invalid log severity %q", value)
	}
	return ls, nil
}

// =============================================================================
// ConfigName
// =============================================================================

// ConfigName identifies a PBM configuration by name. The main configuration
// is identified by MainConfig ("main"). Named storage profiles use their
// profile name as the ConfigName.
//
// The zero value is invalid and indicates "not set".
//
// PBM internally uses "" for the main configuration, but this SDK normalizes
// that to "main" so the empty string never leaks to consumers.
type ConfigName struct {
	value string
}

// MainConfig identifies the main (default) PBM configuration.
var MainConfig = ConfigName{"main"}

// NewConfigName creates a ConfigName from the given name.
// Returns an error if name is empty — use MainConfig instead.
func NewConfigName(name string) (ConfigName, error) {
	if name == "" {
		return ConfigName{}, fmt.Errorf("config name cannot be empty; use MainConfig for the main configuration")
	}
	return ConfigName{name}, nil
}

// String returns the string representation of the config name.
func (c ConfigName) String() string { return c.value }

// IsZero reports whether the config name is the zero value (unset).
func (c ConfigName) IsZero() bool { return c.value == "" }

// Equal reports whether two config name values are identical.
func (c ConfigName) Equal(other ConfigName) bool { return c.value == other.value }

// MarshalText implements encoding.TextMarshaler.
func (c ConfigName) MarshalText() ([]byte, error) { return []byte(c.value), nil }

// UnmarshalText implements encoding.TextUnmarshaler.
func (c *ConfigName) UnmarshalText(data []byte) error {
	name := string(data)
	if name == "" {
		*c = MainConfig
		return nil
	}
	cn, err := NewConfigName(name)
	if err != nil {
		return err
	}
	*c = cn
	return nil
}

// =============================================================================
// CommandType
// =============================================================================

// CommandType represents the type of a PBM command or operation.
type CommandType struct {
	value string
}

var (
	CmdTypeBackup        = newCommandType("backup")
	CmdTypeRestore       = newCommandType("restore")
	CmdTypeReplay        = newCommandType("replay")
	CmdTypeCancelBackup  = newCommandType("cancelBackup")
	CmdTypeResync        = newCommandType("resync")
	CmdTypePITR          = newCommandType("pitr")
	CmdTypeDelete        = newCommandType("delete")
	CmdTypeDeletePITR    = newCommandType("deletePitr")
	CmdTypeCleanup       = newCommandType("cleanup")
	CmdTypeAddProfile    = newCommandType("addConfigProfile")
	CmdTypeRemoveProfile = newCommandType("removeConfigProfile")
)

var commandTypes = make(map[string]CommandType)

func newCommandType(s string) CommandType {
	ct := CommandType{s}
	commandTypes[s] = ct
	return ct
}

// String returns the string representation of the command type.
func (c CommandType) String() string { return c.value }

// IsZero reports whether the command type is the zero value (unset).
func (c CommandType) IsZero() bool { return c.value == "" }

// Equal reports whether two command type values are identical.
func (c CommandType) Equal(other CommandType) bool { return c.value == other.value }

// MarshalText implements encoding.TextMarshaler.
func (c CommandType) MarshalText() ([]byte, error) { return []byte(c.value), nil }

// UnmarshalText implements encoding.TextUnmarshaler.
func (c *CommandType) UnmarshalText(data []byte) error {
	parsed, err := ParseCommandType(string(data))
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}

// ParseCommandType parses a string into a CommandType.
// Returns an error if the value is not a known command type.
func ParseCommandType(value string) (CommandType, error) {
	ct, ok := commandTypes[value]
	if !ok {
		return CommandType{}, fmt.Errorf("invalid command type %q", value)
	}
	return ct, nil
}
