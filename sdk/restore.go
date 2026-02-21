package sdk

import (
	"context"
	"time"
)

// RestoreWaitOptions controls the polling behavior of RestoreService.Wait.
type RestoreWaitOptions struct {
	// PollInterval is the duration between status checks. Defaults to 1s.
	PollInterval time.Duration

	// OnProgress is called after each successful poll with the current state.
	// It is not called when the poll returns an error. Optional.
	OnProgress func(*Restore)
}

// RestoreService provides access to PBM restore operations and metadata.
//
// Example — restore from the latest backup:
//
//	backups, _ := client.Backups.List(ctx, sdk.ListBackupsOptions{Limit: 1})
//	result, err := client.Restores.Start(ctx, sdk.StartRestoreOptions{
//	    BackupName: backups[0].Name,
//	})
//	if err != nil {
//	    return err
//	}
//	restore, err := client.Restores.Wait(ctx, result.Name, sdk.RestoreWaitOptions{})
type RestoreService interface {
	// List returns restores matching the given options, ordered by start time
	// (most recent first). Returns an empty slice when no restores match.
	//
	// Example:
	//
	//	restores, err := client.Restores.List(ctx, sdk.ListRestoresOptions{Limit: 20})
	List(ctx context.Context, opts ListRestoresOptions) ([]Restore, error)

	// Get returns a single restore by name. Returns [ErrNotFound] if the
	// restore does not exist.
	Get(ctx context.Context, name string) (*Restore, error)

	// GetByOpID returns a single restore by operation ID. Returns [ErrNotFound]
	// if no restore matches.
	GetByOpID(ctx context.Context, opid string) (*Restore, error)

	// Start initiates a new restore and returns the result. The restore name
	// is auto-generated from the current timestamp. Returns a
	// [*ConcurrentOperationError] if another PBM operation is already running.
	//
	// Example — snapshot restore:
	//
	//	result, err := client.Restores.Start(ctx, sdk.StartRestoreOptions{
	//	    BackupName: "2026-02-19T20:28:16Z",
	//	})
	//
	// Example — PITR restore to a specific point in time:
	//
	//	result, err := client.Restores.Start(ctx, sdk.StartRestoreOptions{
	//	    BackupName: "2026-02-19T20:28:16Z",
	//	    PITRTarget: sdk.Timestamp{T: 1740000000},
	//	})
	Start(ctx context.Context, opts StartRestoreOptions) (RestoreResult, error)

	// Wait polls until the named restore reaches a terminal status or the
	// context is cancelled. Context cancellation stops waiting but does NOT
	// cancel the running restore.
	// TODO(pbm-fix): PBM does not support restore cancellation.
	//
	// Returns the final Restore and nil on success ([StatusDone], [StatusCancelled]).
	// Returns the Restore and an [*OperationError] on failure ([StatusError],
	// [StatusPartlyDone]). On context cancellation, returns the last observed
	// Restore (may be nil) and ctx.Err().
	//
	// Example:
	//
	//	restore, err := client.Restores.Wait(ctx, result.Name, sdk.RestoreWaitOptions{
	//	    PollInterval: 5 * time.Second,
	//	    OnProgress: func(r *sdk.Restore) {
	//	        fmt.Printf("restore %s: %s\n", r.Name, r.Status)
	//	    },
	//	})
	Wait(ctx context.Context, name string, opts RestoreWaitOptions) (*Restore, error)
}

// ListRestoresOptions controls filtering and pagination for restore listing.
type ListRestoresOptions struct {
	// Limit is the maximum number of restores to return. Zero means no limit.
	Limit int
}

// Restore represents a PBM restore operation.
type Restore struct {
	Name             string           // unique restore name (auto-generated UTC timestamp)
	OPID             string           // operation ID assigned by PBM
	Backup           string           // source backup name
	BcpChain         []string         // for incremental restores: the ordered backup chain
	Type             BackupType       // type of the source backup (logical, physical, etc.)
	Status           Status           // current lifecycle status
	StartTS          time.Time        // when the restore process started
	FinishTS         time.Time        // zero if not finished; derived from LastTransitionTS on terminal status
	PITRTarget       Timestamp        // zero if not a PITR restore; the target oplog position
	Namespaces       []string         // nil means full restore; non-nil lists specific db.collection patterns
	LastTransitionTS time.Time        // when the status last changed
	Error            string           // non-empty on StatusError or StatusPartlyDone
	Replsets         []RestoreReplset // per-replica-set breakdown
}

// InProgress reports whether the restore is still running (not in a terminal
// state).
func (r Restore) InProgress() bool {
	return !r.Status.IsTerminal()
}

// Duration returns the elapsed time from start to the last status transition.
// Returns zero if the restore hasn't started or hasn't reached a terminal
// status yet.
func (r Restore) Duration() time.Duration {
	if r.StartTS.IsZero() || r.LastTransitionTS.IsZero() || !r.Status.IsTerminal() {
		return 0
	}
	return r.LastTransitionTS.Sub(r.StartTS)
}

// RestoreReplset holds per-replica-set metadata for a restore.
type RestoreReplset struct {
	Name             string        // replica set name
	Status           Status        // per-RS status
	LastTransitionTS time.Time     // when this RS's status last changed
	Error            string        // per-RS error message, if any
	Nodes            []RestoreNode // per-node status; populated for physical restores only
}

// RestoreNode holds per-node metadata for a physical restore.
type RestoreNode struct {
	Name             string    // MongoDB node hostname
	Status           Status    // per-node status
	LastTransitionTS time.Time // when this node's status last changed
	Error            string    // per-node error message, if any
}
