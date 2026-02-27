package sdk

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConcurrentOperationErrorMessage(t *testing.T) {
	err := &ConcurrentOperationError{
		Type: CmdTypeBackup,
		OPID: "abc123",
	}

	assert.Equal(t, "another operation is running: backup (opid: abc123)", err.Error())
}

func TestConcurrentOperationErrorAs(t *testing.T) {
	original := &ConcurrentOperationError{
		Type: CmdTypeRestore,
		OPID: "xyz789",
	}
	wrapped := fmt.Errorf("start restore: %w", original)

	var target *ConcurrentOperationError
	assert.True(t, errors.As(wrapped, &target))
	assert.Equal(t, CmdTypeRestore, target.Type)
	assert.Equal(t, "xyz789", target.OPID)
}

func TestOperationErrorMessage(t *testing.T) {
	err := &OperationError{
		Name:    "2026-02-19T20:28:16Z",
		Message: "mongodump failed: connection refused",
	}

	assert.Equal(t, `operation "2026-02-19T20:28:16Z" failed: mongodump failed: connection refused`, err.Error())
}

func TestOperationErrorAs(t *testing.T) {
	original := &OperationError{
		Name:    "2026-02-19T20:28:16Z",
		Message: "timeout",
	}
	wrapped := fmt.Errorf("wait: %w", original)

	var target *OperationError
	assert.True(t, errors.As(wrapped, &target))
	assert.Equal(t, "2026-02-19T20:28:16Z", target.Name)
	assert.Equal(t, "timeout", target.Message)
}
