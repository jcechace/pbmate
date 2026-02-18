package sdk

import (
	"context"
	"fmt"
)

// CommandService handles dispatching commands to PBM agents.
// It performs pre-flight checks (lock validation) before sending.
type CommandService interface {
	// Send dispatches a command to PBM agents and returns the result.
	Send(ctx context.Context, cmd Command) (CommandResult, error)
}

// Command is a sealed interface representing a PBM command.
// Only types defined in this package can implement it.
type Command interface {
	kind() string // unexported method prevents external implementations
}

// CommandResult is returned when a command is dispatched to PBM agents.
type CommandResult struct {
	OPID string
}

// BackupCommand describes a backup to initiate.
type BackupCommand struct {
	Name        string          // backup name (typically timestamp-based)
	Type        BackupType      // logical, physical, incremental, external
	ConfigName  ConfigName      // storage target; zero value = main config
	Compression CompressionType // zero value = use server default
	Namespaces  []string        // nil = full backup
	IncrBase    bool            // for incremental: start a new base
}

func (c BackupCommand) kind() string { return fmt.Sprintf("%T", c) }

// RestoreCommand describes a restore to initiate.
type RestoreCommand struct {
	Name       string    // restore name (typically timestamp-based)
	BackupName string    // required: which backup to restore from
	PITRTarget Timestamp // zero = snapshot restore, non-zero = PITR target
	Namespaces []string  // nil = full restore
}

func (c RestoreCommand) kind() string { return fmt.Sprintf("%T", c) }

// DeleteBackupCommand requests deletion of a specific backup by name.
type DeleteBackupCommand struct {
	Name string // backup name to delete
}

func (c DeleteBackupCommand) kind() string { return fmt.Sprintf("%T", c) }

// CancelBackupCommand requests cancellation of the currently running backup.
type CancelBackupCommand struct{}

func (c CancelBackupCommand) kind() string { return fmt.Sprintf("%T", c) }

// BackupResult is returned by BackupService.Start.
type BackupResult struct {
	CommandResult
	Name string // generated backup name
}

// RestoreResult is returned by RestoreService.Start.
type RestoreResult struct {
	CommandResult
	Name string // generated restore name
}

// StartBackupOptions configures a backup initiated via BackupService.Start.
type StartBackupOptions struct {
	Type        BackupType
	ConfigName  ConfigName
	Compression CompressionType
	Namespaces  []string
	IncrBase    bool
}

// StartRestoreOptions configures a restore initiated via RestoreService.Start.
type StartRestoreOptions struct {
	BackupName string    // required: which backup to restore from
	PITRTarget Timestamp // zero = snapshot restore, non-zero = PITR target
	Namespaces []string  // nil = full restore
}
