package sdk

import (
	"context"
	"time"
)

// BackupWaitOptions controls the polling behavior of BackupService.Wait.
type BackupWaitOptions struct {
	// PollInterval is the duration between status checks. Defaults to 1s.
	PollInterval time.Duration

	// OnProgress is called after each successful poll with the current state.
	// It is not called when the poll returns an error. Optional.
	OnProgress func(*Backup)
}

// BackupService provides access to PBM backup operations and metadata.
//
// Example — list recent backups and inspect the latest:
//
//	backups, err := client.Backups.List(ctx, sdk.ListBackupsOptions{Limit: 5})
//	if err != nil {
//	    return err
//	}
//	for _, bk := range backups {
//	    fmt.Printf("%s  %s  %s\n", bk.Name, bk.Type, bk.Status)
//	}
type BackupService interface {
	// List returns backups matching the given options, ordered by start time
	// (most recent first). Returns an empty slice when no backups match.
	//
	// Example:
	//
	//	// All backups, no limit.
	//	backups, err := client.Backups.List(ctx, sdk.ListBackupsOptions{})
	//
	//	// Last 10 logical backups from a named profile.
	//	name, _ := sdk.NewConfigName("archive")
	//	backups, err := client.Backups.List(ctx, sdk.ListBackupsOptions{
	//	    Limit:      10,
	//	    Type:       sdk.BackupTypeLogical,
	//	    ConfigName: name,
	//	})
	List(ctx context.Context, opts ListBackupsOptions) ([]Backup, error)

	// Get returns a single backup by name. Returns [ErrNotFound] if the
	// backup does not exist.
	//
	// Example:
	//
	//	bk, err := client.Backups.Get(ctx, "2026-02-19T20:28:16Z")
	//	if errors.Is(err, sdk.ErrNotFound) {
	//	    fmt.Println("backup not found")
	//	}
	Get(ctx context.Context, name string) (*Backup, error)

	// GetByOpID returns a single backup by operation ID. Returns [ErrNotFound]
	// if no backup matches.
	GetByOpID(ctx context.Context, opid string) (*Backup, error)

	// Start initiates a new backup and returns the result. The backup name
	// is auto-generated from the current timestamp. Returns a
	// [*ConcurrentOperationError] if another PBM operation is already running.
	//
	// The cmd parameter is a sealed [StartBackupCommand] with variants:
	//   - [StartLogicalBackup] for logical (mongodump) backups.
	//   - [StartPhysicalBackup] for physical (WiredTiger file-level) backups.
	//   - [StartIncrementalBackup] for incremental backups.
	//
	// Example — logical backup:
	//
	//	result, err := client.Backups.Start(ctx, sdk.StartLogicalBackup{})
	//
	// Example — selective logical backup to a named profile:
	//
	//	name, _ := sdk.NewConfigName("archive")
	//	result, err := client.Backups.Start(ctx, sdk.StartLogicalBackup{
	//	    ConfigName: name,
	//	    Namespaces: []string{"mydb.mycol"},
	//	})
	//
	// Example — physical backup:
	//
	//	result, err := client.Backups.Start(ctx, sdk.StartPhysicalBackup{})
	//
	// Example — incremental backup base:
	//
	//	result, err := client.Backups.Start(ctx, sdk.StartIncrementalBackup{
	//	    Base: true,
	//	})
	Start(ctx context.Context, cmd StartBackupCommand) (BackupResult, error)

	// Wait polls until the named backup reaches a terminal status or the
	// context is cancelled. Context cancellation stops waiting but does NOT
	// cancel the running backup — use [BackupService.Cancel] for that.
	//
	// Returns the final Backup and nil on success ([StatusDone], [StatusCancelled]).
	// Returns the Backup and an [*OperationError] on failure ([StatusError],
	// [StatusPartlyDone]). On context cancellation, returns the last observed
	// Backup (may be nil) and ctx.Err().
	//
	// Example:
	//
	//	result, _ := client.Backups.Start(ctx, sdk.StartLogicalBackup{})
	//	bk, err := client.Backups.Wait(ctx, result.Name, sdk.BackupWaitOptions{
	//	    PollInterval: 2 * time.Second,
	//	    OnProgress: func(b *sdk.Backup) {
	//	        fmt.Printf("status: %s\n", b.Status)
	//	    },
	//	})
	Wait(ctx context.Context, name string, opts BackupWaitOptions) (*Backup, error)

	// Delete requests deletion of one or more backups. The deletion is
	// processed asynchronously by PBM agents — the command returns
	// immediately. Returns a [*ConcurrentOperationError] if another
	// operation is running.
	//
	// The cmd parameter is a sealed [DeleteBackupCommand] with two variants:
	//   - [DeleteBackupByName] deletes a single backup.
	//   - [DeleteBackupsBefore] bulk-deletes backups older than a cutoff.
	//
	// Example — delete by name:
	//
	//	_, err := client.Backups.Delete(ctx, sdk.DeleteBackupByName{
	//	    Name: "2026-02-19T20:28:16Z",
	//	})
	//
	// Example — bulk delete older than a cutoff:
	//
	//	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	//	_, err := client.Backups.Delete(ctx, sdk.DeleteBackupsBefore{
	//	    OlderThan: cutoff,
	//	    Type:      sdk.BackupTypeLogical,
	//	})
	Delete(ctx context.Context, cmd DeleteBackupCommand) (CommandResult, error)

	// Cancel requests cancellation of the currently running backup. If no
	// backup is running, the command is accepted but has no effect.
	//
	// Example:
	//
	//	_, err := client.Backups.Cancel(ctx)
	Cancel(ctx context.Context) (CommandResult, error)

	// CanDelete checks whether the named backup can be safely deleted.
	// Returns nil if deletion is safe, or a descriptive error explaining
	// why the backup is protected.
	//
	// For incremental backups, the name must be the chain base. Non-base
	// increments are rejected with [ErrNotChainBase] — callers should
	// resolve to the base name (via [BackupChain.Base] or [FindChainBase])
	// before calling CanDelete.
	//
	// Possible errors:
	//   - [ErrNotFound]: no backup with this name exists.
	//   - [ErrBackupInProgress]: the backup has not reached a terminal status.
	//   - [ErrDeleteProtectedByPITR]: the backup is the last PITR base snapshot.
	//   - [ErrNotChainBase]: the backup is a non-base incremental backup.
	//
	// Example:
	//
	//	if err := client.Backups.CanDelete(ctx, bk.Name); err != nil {
	//	    fmt.Printf("cannot delete: %v\n", err)
	//	}
	CanDelete(ctx context.Context, name string) error
}

// ListBackupsOptions controls filtering and pagination for backup listing.
type ListBackupsOptions struct {
	// Limit is the maximum number of backups to return. Zero means no limit.
	Limit int

	// ConfigName filters by storage configuration name. Zero value means all.
	ConfigName ConfigName

	// Type filters by backup type. Empty means all types.
	Type BackupType
}

// Backup represents a PBM backup snapshot.
type Backup struct {
	Name             string          // unique backup name (typically a UTC timestamp like "2026-02-19T20:28:16Z")
	OPID             string          // operation ID assigned by PBM
	Type             BackupType      // logical, physical, incremental, or external
	Status           Status          // current lifecycle status (use Status.IsTerminal to check completion)
	Compression      CompressionType // compression algorithm used; zero if server default
	ConfigName       ConfigName      // storage profile; always normalized — MainConfig for the default storage, never zero
	StartTS          time.Time       // when the backup process started
	FirstWriteTS     Timestamp       // first oplog write position; use with LastWriteTS for oplog range
	LastWriteTS      Timestamp       // restore-to point (oplog position); use LastWriteTS.Time() for display
	LastTransitionTS time.Time       // when the status last changed
	Size             int64           // compressed size in bytes; zero while in progress
	SizeUncompressed int64           // original data size in bytes
	Namespaces       []string        // nil means full backup; non-nil lists specific db.collection patterns
	SrcBackup        string          // for incremental: parent backup name
	MongoVersion     string          // MongoDB server version at backup time
	FCV              string          // feature compatibility version
	PBMVersion       string          // PBM agent version that created the backup
	Error            string          // non-empty on StatusError or StatusPartlyDone
	Replsets         []BackupReplset // per-replica-set breakdown
}

// IsLogical reports whether this is a logical (mongodump-based) backup.
func (b Backup) IsLogical() bool {
	return b.Type.Equal(BackupTypeLogical)
}

// IsPhysical reports whether this is a physical (WiredTiger file-level) backup.
func (b Backup) IsPhysical() bool {
	return b.Type.Equal(BackupTypePhysical)
}

// IsIncremental reports whether this is an incremental backup.
func (b Backup) IsIncremental() bool {
	return b.Type.Equal(BackupTypeIncremental)
}

// IsIncrementalBase reports whether this backup is the base of an incremental
// chain. An incremental base has no parent (SrcBackup is empty).
func (b Backup) IsIncrementalBase() bool {
	return b.IsIncremental() && b.SrcBackup == ""
}

// IsSelective reports whether this backup targets specific namespaces
// rather than the full cluster.
func (b Backup) IsSelective() bool {
	return len(b.Namespaces) > 0
}

// InProgress reports whether the backup is still running (not in a terminal
// state).
func (b Backup) InProgress() bool {
	return !b.Status.IsTerminal()
}

// Duration returns the wall-clock time from start to completion.
// Returns zero if the backup hasn't started or hasn't reached a terminal
// status yet. For in-progress duration, use [Backup.Elapsed].
func (b Backup) Duration() time.Duration {
	if b.StartTS.IsZero() || b.LastTransitionTS.IsZero() || !b.Status.IsTerminal() {
		return 0
	}
	return b.LastTransitionTS.Sub(b.StartTS)
}

// Elapsed returns the time spent so far. For completed backups this is the
// final duration (identical to [Backup.Duration]). For in-progress backups
// it returns the live elapsed time since start. Returns zero if the backup
// hasn't started.
func (b Backup) Elapsed() time.Duration {
	if b.StartTS.IsZero() {
		return 0
	}
	if b.Status.IsTerminal() && !b.LastTransitionTS.IsZero() {
		return b.LastTransitionTS.Sub(b.StartTS)
	}
	return time.Since(b.StartTS)
}

// BackupReplset holds per-replica-set metadata for a backup.
type BackupReplset struct {
	Name             string    // replica set name
	Status           Status    // per-RS status (may differ from the backup's overall status)
	Node             string    // the node that performed the backup for this RS
	LastWriteTS      Timestamp // per-RS restore-to point
	LastTransitionTS time.Time // when this RS's status last changed
	Size             int64     // compressed size in bytes for this replica set
	SizeUncompressed int64     // original data size in bytes for this replica set
	IsConfigSvr      bool      // true for the config server replica set
	Error            string    // per-RS error message, if any
}
