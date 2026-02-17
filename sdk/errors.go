package sdk

import (
	"errors"
	"fmt"
)

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("not found")

// ConcurrentOperationError is returned when a command cannot be dispatched
// because another PBM operation is already running.
type ConcurrentOperationError struct {
	Type CommandType
	OPID string
}

func (e *ConcurrentOperationError) Error() string {
	return fmt.Sprintf("another operation is running: %s (opid: %s)", e.Type, e.OPID)
}
