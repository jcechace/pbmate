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
	// Example:
	//
	//	result, err := client.Backups.Start(ctx, sdk.StartBackupOptions{
	//	    Type: sdk.BackupTypeLogical,
	//	})
	//	if err != nil {
	//	    var concurrent *sdk.ConcurrentOperationError
	//	    if errors.As(err, &concurrent) {
	//	        fmt.Printf("busy: %s is running\n", concurrent.Type)
	//	    }
	//	    return err
	//	}
	//	fmt.Printf("started backup %s (opid: %s)\n", result.Name, result.OPID)
	Start(ctx context.Context, opts StartBackupOptions) (BackupResult, error)

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
	//	result, _ := client.Backups.Start(ctx, sdk.StartBackupOptions{})
	//	bk, err := client.Backups.Wait(ctx, result.Name, sdk.BackupWaitOptions{
	//	    PollInterval: 2 * time.Second,
	//	    OnProgress: func(b *sdk.Backup) {
	//	        fmt.Printf("status: %s\n", b.Status)
	//	    },
	//	})
	Wait(ctx context.Context, name string, opts BackupWaitOptions) (*Backup, error)

	// Delete requests deletion of a backup by name. The deletion is processed
	// asynchronously by PBM agents — the command returns immediately.
	// Returns a [*ConcurrentOperationError] if another operation is running.
	//
	// Example:
	//
	//	_, err := client.Backups.Delete(ctx, "2026-02-19T20:28:16Z")
	Delete(ctx context.Context, name string) (CommandResult, error)

	// Cancel requests cancellation of the currently running backup. If no
	// backup is running, the command is accepted but has no effect.
	//
	// Example:
	//
	//	_, err := client.Backups.Cancel(ctx)
	Cancel(ctx context.Context) (CommandResult, error)
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
	ConfigName       ConfigName      // storage profile; MainConfig for the default storage
	StartTS          time.Time       // when the backup process started
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

// BackupReplset holds per-replica-set metadata for a backup.
type BackupReplset struct {
	Name             string    // replica set name
	Status           Status    // per-RS status (may differ from the backup's overall status)
	Node             string    // the node that performed the backup for this RS
	LastWriteTS      Timestamp // per-RS restore-to point
	LastTransitionTS time.Time // when this RS's status last changed
	IsConfigSvr      bool      // true for the config server replica set
	Error            string    // per-RS error message, if any
}
