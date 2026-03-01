//go:build integration

package integtest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/percona-backup-mongodb/pbm/ctrl"
	"github.com/percona/percona-backup-mongodb/pbm/defs"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// --- Restore.Start ---

func TestRestoreStartSnapshot(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed the backup that the restore references (needed for result type detection).
	h.seedBackup(t, newBackupMeta("2024-06-15T10:30:00Z",
		withBackupStartTS(100),
	))

	parallel := 4
	workers := 2
	allowPartly := true
	fallback := true

	result, err := h.client.Restores.Start(ctx, sdk.StartSnapshotRestore{
		BackupName:          "2024-06-15T10:30:00Z",
		Namespaces:          []string{"db1.coll1"},
		NamespaceFrom:       "db1",
		NamespaceTo:         "db2",
		NumParallelColls:    &parallel,
		NumInsertionWorkers: &workers,
		AllowPartlyDone:     &allowPartly,
		Fallback:            &fallback,
	})
	require.NoError(t, err)

	// Result fields.
	assert.NotEmpty(t, result.OPID())
	assert.NotEmpty(t, result.Name())

	// Restore names use RFC 3339 Nano.
	_, parseErr := time.Parse(time.RFC3339Nano, result.Name())
	assert.NoError(t, parseErr, "restore name should be RFC 3339 Nano")

	// Logical backup -> waitable result.
	assert.True(t, result.Waitable())

	// Verify the dispatched command.
	cmd := h.lastCommand(t)
	assert.Equal(t, ctrl.CmdRestore, cmd.Cmd)
	require.NotNil(t, cmd.Restore)
	assert.Equal(t, result.Name(), cmd.Restore.Name)
	assert.Equal(t, "2024-06-15T10:30:00Z", cmd.Restore.BackupName)
	assert.Equal(t, []string{"db1.coll1"}, cmd.Restore.Namespaces)
	assert.Equal(t, "db1", cmd.Restore.NamespaceFrom)
	assert.Equal(t, "db2", cmd.Restore.NamespaceTo)
	require.NotNil(t, cmd.Restore.NumParallelColls)
	assert.Equal(t, int32(4), *cmd.Restore.NumParallelColls)
	require.NotNil(t, cmd.Restore.NumInsertionWorkers)
	assert.Equal(t, int32(2), *cmd.Restore.NumInsertionWorkers)
	require.NotNil(t, cmd.Restore.AllowPartlyDone)
	assert.True(t, *cmd.Restore.AllowPartlyDone)
	require.NotNil(t, cmd.Restore.Fallback)
	assert.True(t, *cmd.Restore.Fallback)
}

func TestRestoreStartPITR(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed a logical backup as the base.
	h.seedBackup(t, newBackupMeta("2024-06-15T10:30:00Z",
		withBackupStartTS(100),
	))

	target := sdk.Timestamp{T: 1718444000, I: 5}

	result, err := h.client.Restores.Start(ctx, sdk.StartPITRRestore{
		BackupName: "2024-06-15T10:30:00Z",
		Target:     target,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.OPID())
	assert.True(t, result.Waitable()) // logical base -> waitable

	cmd := h.lastCommand(t)
	assert.Equal(t, ctrl.CmdRestore, cmd.Cmd)
	require.NotNil(t, cmd.Restore)
	assert.Equal(t, "2024-06-15T10:30:00Z", cmd.Restore.BackupName)
	assert.Equal(t, uint32(1718444000), cmd.Restore.OplogTS.T)
	assert.Equal(t, uint32(5), cmd.Restore.OplogTS.I)
}

func TestRestoreStartPhysicalBackupUnwaitable(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed a physical backup.
	h.seedBackup(t, newBackupMeta("2024-06-15T10:30:00Z",
		withBackupType(defs.PhysicalBackup),
		withBackupStartTS(100),
	))

	result, err := h.client.Restores.Start(ctx, sdk.StartSnapshotRestore{
		BackupName: "2024-06-15T10:30:00Z",
	})
	require.NoError(t, err)

	// Physical backup -> unwaitable result.
	assert.False(t, result.Waitable())

	// Wait should return ErrRestoreUnwaitable.
	_, waitErr := result.Wait(ctx, sdk.RestoreWaitOptions{})
	assert.ErrorIs(t, waitErr, sdk.ErrRestoreUnwaitable)
}

func TestRestoreStartIncrementalBackupUnwaitable(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed an incremental backup.
	h.seedBackup(t, newBackupMeta("2024-06-15T10:30:00Z",
		withBackupType(defs.IncrementalBackup),
		withBackupStartTS(100),
	))

	result, err := h.client.Restores.Start(ctx, sdk.StartSnapshotRestore{
		BackupName: "2024-06-15T10:30:00Z",
	})
	require.NoError(t, err)

	// Incremental backup -> unwaitable result.
	assert.False(t, result.Waitable())
}

// --- RestoreResult.Wait ---

func TestRestoreStartAndWait(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed a logical backup (produces a waitable result).
	h.seedBackup(t, newBackupMeta("2024-06-15T10:30:00Z",
		withBackupStartTS(100),
	))

	result, err := h.client.Restores.Start(ctx, sdk.StartSnapshotRestore{
		BackupName: "2024-06-15T10:30:00Z",
	})
	require.NoError(t, err)
	require.True(t, result.Waitable())

	// Seed restore metadata with terminal status BEFORE calling Wait.
	h.seedRestore(t, newRestoreMeta(result.Name(),
		withRestoreBackup("2024-06-15T10:30:00Z"),
		withRestoreStatus(defs.StatusDone),
		withRestoreStartTS(200),
		withRestoreLastTransitionTS(300),
	))

	rs, err := result.Wait(ctx, sdk.RestoreWaitOptions{
		PollInterval: 100 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NotNil(t, rs)
	assert.Equal(t, result.Name(), rs.Name)
	assert.True(t, rs.Status.Equal(sdk.StatusDone))
}

func TestRestoreStartUnwaitableWait(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed a physical backup (produces an unwaitable result).
	h.seedBackup(t, newBackupMeta("2024-06-15T10:30:00Z",
		withBackupType(defs.PhysicalBackup),
		withBackupStartTS(100),
	))

	result, err := h.client.Restores.Start(ctx, sdk.StartSnapshotRestore{
		BackupName: "2024-06-15T10:30:00Z",
	})
	require.NoError(t, err)
	require.False(t, result.Waitable())

	// Wait should return ErrRestoreUnwaitable immediately (no seed needed).
	rs, err := result.Wait(ctx, sdk.RestoreWaitOptions{})
	assert.ErrorIs(t, err, sdk.ErrRestoreUnwaitable)
	assert.Nil(t, rs)
}

func TestRestoreStartMinimal(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedBackup(t, newBackupMeta("2024-01-01T00:00:00Z",
		withBackupStartTS(100),
	))

	result, err := h.client.Restores.Start(ctx, sdk.StartSnapshotRestore{
		BackupName: "2024-01-01T00:00:00Z",
	})
	require.NoError(t, err)

	cmd := h.lastCommand(t)
	require.NotNil(t, cmd.Restore)
	// Optional fields should be nil/empty.
	assert.Empty(t, cmd.Restore.Namespaces)
	assert.Empty(t, cmd.Restore.NamespaceFrom)
	assert.Empty(t, cmd.Restore.NamespaceTo)
	assert.Nil(t, cmd.Restore.NumParallelColls)
	assert.Nil(t, cmd.Restore.NumInsertionWorkers)
	assert.Nil(t, cmd.Restore.AllowPartlyDone)
	assert.Nil(t, cmd.Restore.Fallback)
	assert.Equal(t, result.Name(), cmd.Restore.Name)
}
