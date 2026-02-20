package sdk

import (
	"context"
	"fmt"
)

// CommandService handles dispatching commands to PBM agents.
// It performs pre-flight checks (lock validation) before sending.
//
// Most callers should prefer the higher-level methods on [BackupService] and
// [RestoreService] (Start, Delete, Cancel) which build the correct command
// internally. Use CommandService directly only when you need low-level control
// over command construction.
//
// Example — dispatch a backup command directly:
//
//	cmd := sdk.BackupCommand{
//	    Name: "2026-02-20T10:00:00Z",
//	    Type: sdk.BackupTypeLogical,
//	}
//	result, err := client.Commands.Send(ctx, cmd)
//	if err != nil {
//	    var concurrent *sdk.ConcurrentOperationError
//	    if errors.As(err, &concurrent) {
//	        fmt.Printf("blocked by %s (opid: %s)\n", concurrent.Type, concurrent.OPID)
//	    }
//	    return err
//	}
//	fmt.Printf("dispatched, opid: %s\n", result.OPID)
type CommandService interface {
	// Send dispatches a command to PBM agents and returns the result.
	// It calls [CommandService.CheckLock] internally before dispatching;
	// callers do not need to check for concurrent operations separately.
	//
	// Returns a [*ConcurrentOperationError] if a non-stale PBM operation is
	// already running.
	Send(ctx context.Context, cmd Command) (CommandResult, error)

	// CheckLock verifies no non-stale PBM operation is currently running.
	// Returns a [*ConcurrentOperationError] if one is, nil otherwise.
	//
	// A lock is considered stale when its heartbeat is older than PBM's
	// stale frame threshold relative to the cluster time.
	CheckLock(ctx context.Context) error
}

// Command is a sealed interface representing a PBM command.
// Only types defined in this package can implement it.
type Command interface {
	kind() string // unexported method prevents external implementations
}

// CommandResult is returned when a command is dispatched to PBM agents.
type CommandResult struct {
	// OPID is the operation ID (MongoDB ObjectID hex string) assigned to the
	// dispatched command. Use it with [BackupService.GetByOpID] to look up
	// the resulting backup metadata.
	OPID string
}

// BackupCommand describes a backup to initiate.
// Most callers should use [BackupService.Start] instead, which auto-generates
// the name and wraps this command type internally.
type BackupCommand struct {
	// Name is the backup identifier, typically an RFC 3339 timestamp
	// (e.g. "2026-02-20T10:00:00Z"). Must be unique across all backups.
	Name string

	// Type selects the backup strategy. Zero value is passed through to PBM
	// which defaults to logical.
	Type BackupType

	// ConfigName selects the storage profile target. Zero value and
	// [MainConfig] both route to the main PBM storage configuration.
	ConfigName ConfigName

	// Compression overrides the server-configured compression algorithm.
	// Zero value uses the server default.
	Compression CompressionType

	// Namespaces limits the backup to specific databases or collections
	// (e.g. ["mydb.mycol"]). Nil means a full backup of all namespaces.
	Namespaces []string

	// IncrBase starts a new incremental backup base when true. Only
	// meaningful when Type is [BackupTypeIncremental].
	IncrBase bool
}

func (c BackupCommand) kind() string { return fmt.Sprintf("%T", c) }

// RestoreCommand describes a restore to initiate.
// Most callers should use [RestoreService.Start] instead, which auto-generates
// the name and wraps this command type internally.
type RestoreCommand struct {
	// Name is the restore identifier, typically an RFC 3339 timestamp.
	Name string

	// BackupName is the source backup to restore from. Required.
	BackupName string

	// PITRTarget selects the point-in-time to restore to. A zero value
	// requests a snapshot restore (restore the backup as-is). A non-zero
	// value requests PITR replay up to that timestamp.
	PITRTarget Timestamp

	// Namespaces limits the restore to specific databases or collections.
	// Nil means a full restore of all namespaces in the backup.
	Namespaces []string
}

func (c RestoreCommand) kind() string { return fmt.Sprintf("%T", c) }

// DeleteBackupCommand requests deletion of a specific backup and its data
// from storage. Most callers should use [BackupService.Delete] instead.
type DeleteBackupCommand struct {
	// Name is the backup to delete (e.g. "2026-02-20T10:00:00Z").
	Name string
}

func (c DeleteBackupCommand) kind() string { return fmt.Sprintf("%T", c) }

// AddProfileCommand requests creation or replacement of a named storage profile.
// The storage field is unexported and populated by [ConfigService] from parsed YAML;
// callers cannot construct this command directly — use ConfigService instead.
type AddProfileCommand struct {
	// Name is the profile name to create or replace.
	Name string

	storage any // holds parsed PBM config.StorageConf; set by ConfigService
}

func (c AddProfileCommand) kind() string { return fmt.Sprintf("%T", c) }

// RemoveProfileCommand requests deletion of a named storage profile.
type RemoveProfileCommand struct {
	// Name is the profile to remove.
	Name string
}

func (c RemoveProfileCommand) kind() string { return fmt.Sprintf("%T", c) }

// CancelBackupCommand requests cancellation of the currently running backup.
type CancelBackupCommand struct{}

func (c CancelBackupCommand) kind() string { return fmt.Sprintf("%T", c) }

// BackupResult is returned by [BackupService.Start].
type BackupResult struct {
	CommandResult

	// Name is the auto-generated backup name (RFC 3339 UTC timestamp).
	// Use it with [BackupService.Get] or [BackupService.Wait] to track progress.
	Name string
}

// RestoreResult is returned by [RestoreService.Start].
type RestoreResult struct {
	CommandResult

	// Name is the auto-generated restore name (RFC 3339 UTC timestamp).
	// Use it with [RestoreService.Get] or [RestoreService.Wait] to track progress.
	Name string
}

// StartBackupOptions configures a backup initiated via [BackupService.Start].
// All fields have sensible zero values: a zero-value StartBackupOptions
// requests a full logical backup to the main storage with server-default
// compression.
type StartBackupOptions struct {
	// Type selects the backup strategy. Zero value is passed through to PBM
	// which defaults to logical.
	Type BackupType

	// ConfigName selects the storage profile. Zero value targets the main
	// PBM storage configuration.
	ConfigName ConfigName

	// Compression overrides the server-configured compression algorithm.
	// Zero value uses the server default.
	Compression CompressionType

	// Namespaces limits the backup to specific databases or collections.
	// Nil means a full backup of all namespaces.
	Namespaces []string

	// IncrBase starts a new incremental backup base when true. Only
	// meaningful when Type is [BackupTypeIncremental].
	IncrBase bool
}

// StartRestoreOptions configures a restore initiated via [RestoreService.Start].
type StartRestoreOptions struct {
	// BackupName is the source backup to restore from. Required.
	BackupName string

	// PITRTarget selects the point-in-time to restore to. A zero value
	// requests a snapshot restore. A non-zero value requests PITR replay
	// up to that timestamp; the target must fall within a continuous oplog
	// timeline that starts from the named backup.
	PITRTarget Timestamp

	// Namespaces limits the restore to specific databases or collections.
	// Nil means a full restore of all namespaces in the backup.
	Namespaces []string
}
