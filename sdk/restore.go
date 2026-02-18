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

// RestoreService provides read access to PBM restore metadata.
type RestoreService interface {
	// List returns restores matching the given options.
	List(ctx context.Context, opts ListRestoresOptions) ([]Restore, error)

	// Get returns a single restore by name.
	Get(ctx context.Context, name string) (*Restore, error)

	// GetByOpID returns a single restore by operation ID.
	GetByOpID(ctx context.Context, opid string) (*Restore, error)

	// Start initiates a new restore and returns the result.
	Start(ctx context.Context, opts StartRestoreOptions) (RestoreResult, error)

	// Wait polls until the named restore reaches a terminal status or the
	// context is cancelled. Context cancellation stops waiting but does NOT
	// cancel the running restore.
	//
	// Returns the final Restore and nil on success (StatusDone, StatusCancelled).
	// Returns the Restore and an *OperationError on failure (StatusError,
	// StatusPartlyDone). On context cancellation, returns the last observed
	// Restore (may be nil) and ctx.Err().
	Wait(ctx context.Context, name string, opts RestoreWaitOptions) (*Restore, error)
}

// ListRestoresOptions controls filtering and pagination for restore listing.
type ListRestoresOptions struct {
	// Limit is the maximum number of restores to return. Zero means no limit.
	Limit int
}

// Restore represents a PBM restore operation.
type Restore struct {
	Name             string
	OPID             string
	Backup           string   // source backup name
	BcpChain         []string // for incremental restores: the ordered backup chain
	Type             BackupType
	Status           Status
	StartTS          time.Time
	FinishTS         time.Time // zero if not finished; derived from LastTransitionTS on terminal status
	PITRTarget       Timestamp // zero if not a PITR restore
	Namespaces       []string
	LastTransitionTS time.Time
	Error            string
	Replsets         []RestoreReplset
}

// RestoreReplset holds per-replica-set metadata for a restore.
type RestoreReplset struct {
	Name             string
	Status           Status
	LastTransitionTS time.Time
	Error            string
	Nodes            []RestoreNode // per-node status; populated for physical restores
}

// RestoreNode holds per-node metadata for a physical restore.
type RestoreNode struct {
	Name             string
	Status           Status
	LastTransitionTS time.Time
	Error            string
}
