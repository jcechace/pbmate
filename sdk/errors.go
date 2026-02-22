package sdk

import (
	"errors"
	"fmt"
)

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("not found")

// ErrBackupInProgress is returned by [BackupService.CanDelete] when the
// target backup has not reached a terminal status and cannot be deleted yet.
var ErrBackupInProgress = errors.New("backup is currently running")

// ErrDeleteProtectedByPITR is returned by [BackupService.CanDelete] when
// the backup is the last PITR base snapshot. Deleting it would break
// point-in-time recovery continuity while PITR is enabled.
var ErrDeleteProtectedByPITR = errors.New("backup is the last PITR base snapshot and cannot be deleted while PITR is enabled")

// ConcurrentOperationError is returned when a command cannot be dispatched
// because another PBM operation is already running.
type ConcurrentOperationError struct {
	Type CommandType
	OPID string
}

func (e *ConcurrentOperationError) Error() string {
	return fmt.Sprintf("another operation is running: %s (opid: %s)", e.Type, e.OPID)
}

// OperationError is returned by Wait when the operation reaches a failed
// terminal status (StatusError or StatusPartlyDone). The corresponding
// Backup or Restore struct is still returned alongside the error so callers
// can inspect the full metadata.
type OperationError struct {
	Name    string // backup or restore name
	Message string // error message from the operation
}

func (e *OperationError) Error() string {
	return fmt.Sprintf("operation %q failed: %s", e.Name, e.Message)
}
