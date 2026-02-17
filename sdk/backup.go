package sdk

import (
	"context"
	"time"
)

// BackupService provides read access to PBM backup metadata.
type BackupService interface {
	// List returns backups matching the given options.
	List(ctx context.Context, opts ListBackupsOptions) ([]Backup, error)

	// Get returns a single backup by name.
	Get(ctx context.Context, name string) (*Backup, error)

	// GetByOpID returns a single backup by operation ID.
	GetByOpID(ctx context.Context, opid string) (*Backup, error)
}

// ListBackupsOptions controls filtering and pagination for backup listing.
type ListBackupsOptions struct {
	// Limit is the maximum number of backups to return. Zero means no limit.
	Limit int

	// Profile filters by storage profile name. Empty means the default profile.
	Profile string

	// Type filters by backup type. Empty means all types.
	Type BackupType
}

// Backup represents a PBM backup snapshot.
type Backup struct {
	Name             string
	OPID             string
	Type             BackupType
	Status           Status
	Compression      CompressionType
	Profile          string
	StartTS          time.Time
	LastWriteTS      Timestamp
	LastTransitionTS time.Time
	Size             int64
	SizeUncompressed int64
	Namespaces       []string // nil means full backup
	SrcBackup        string   // for incremental: parent backup name
	MongoVersion     string
	FCV              string
	PBMVersion       string
	Error            string
	Replsets         []BackupReplset
}

// BackupReplset holds per-replica-set metadata for a backup.
type BackupReplset struct {
	Name             string
	Status           Status
	Node             string
	LastWriteTS      Timestamp
	LastTransitionTS time.Time
	IsConfigSvr      bool
	Error            string
}
