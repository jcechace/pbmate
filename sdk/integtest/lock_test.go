//go:build integration

package integtest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/percona-backup-mongodb/pbm/ctrl"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// --- Lock checking: ConcurrentOperationError ---

func TestStartBlockedByConcurrentOp(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed a fresh (non-stale) lock.
	h.seedLock(t, newLockData(ctrl.CmdBackup, "rs0", "rs0:27017"))

	_, err := h.client.Backups.Start(ctx, sdk.StartLogicalBackup{})
	require.Error(t, err)

	var concErr *sdk.ConcurrentOperationError
	assert.True(t, errors.As(err, &concErr))
	assert.NotEmpty(t, concErr.OPID)
}

func TestDeleteBlockedByConcurrentOp(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedLock(t, newLockData(ctrl.CmdRestore, "rs0", "rs0:27017"))

	_, err := h.client.Backups.Delete(ctx, sdk.DeleteBackupByName{Name: "test"})
	require.Error(t, err)

	var concErr *sdk.ConcurrentOperationError
	assert.True(t, errors.As(err, &concErr))
}

func TestResyncBlockedByConcurrentOp(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedLock(t, newLockData(ctrl.CmdBackup, "rs0", "rs0:27017"))

	_, err := h.client.Config.Resync(ctx, sdk.ResyncMain{})
	require.Error(t, err)

	var concErr *sdk.ConcurrentOperationError
	assert.True(t, errors.As(err, &concErr))
}

func TestPITRDeleteBlockedByConcurrentOp(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedLock(t, newLockData(ctrl.CmdBackup, "rs0", "rs0:27017"))

	cutoff := time.Now().Add(-time.Hour)
	_, err := h.client.PITR.Delete(ctx, sdk.DeletePITRBefore{OlderThan: cutoff})
	require.Error(t, err)

	var concErr *sdk.ConcurrentOperationError
	assert.True(t, errors.As(err, &concErr))
}

// --- Stale locks are ignored ---

func TestStartSucceedsWithStaleLock(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Get cluster time, then seed a lock with heartbeat in the stale
	// window (older than 30s threshold but within 35s GC). PBM GCs at
	// 35s; SDK marks stale at 30s. So heartbeat at clusterTime-32 is
	// stale but not GC'd.
	clusterTime, err := h.client.Cluster.ClusterTime(ctx)
	require.NoError(t, err)

	h.seedLock(t, newLockData(ctrl.CmdBackup, "rs0", "rs0:27017",
		withLockHeartbeat(clusterTime.T-32),
	))

	// Should succeed — stale lock is ignored.
	result, err := h.client.Backups.Start(ctx, sdk.StartLogicalBackup{})
	require.NoError(t, err)
	assert.NotEmpty(t, result.OPID)
}

// --- Cancel bypasses lock check ---

func TestCancelIgnoresLock(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed a fresh lock — Cancel should still succeed.
	h.seedLock(t, newLockData(ctrl.CmdBackup, "rs0", "rs0:27017"))

	result, err := h.client.Backups.Cancel(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, result.OPID)
}

// --- Validation errors: no command dispatched ---

func TestStartValidationError(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// UsersAndRoles requires whole-database namespace patterns.
	_, err := h.client.Backups.Start(ctx, sdk.StartLogicalBackup{
		Namespaces:    []string{"db.specific_collection"},
		UsersAndRoles: true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "users-and-roles")

	// No command should have been dispatched.
	assert.Equal(t, int64(0), h.commandCount(t))
}

func TestRestoreValidationErrorMissingBackup(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	_, err := h.client.Restores.Start(ctx, sdk.StartSnapshotRestore{
		BackupName: "", // required
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backup name")
	assert.Equal(t, int64(0), h.commandCount(t))
}

func TestRestoreValidationErrorMissingTarget(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	_, err := h.client.Restores.Start(ctx, sdk.StartPITRRestore{
		BackupName: "2024-01-01T00:00:00Z",
		// Target is zero — should fail.
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target")
	assert.Equal(t, int64(0), h.commandCount(t))
}

func TestDeleteValidationErrorEmptyName(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	_, err := h.client.Backups.Delete(ctx, sdk.DeleteBackupByName{Name: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
	assert.Equal(t, int64(0), h.commandCount(t))
}

func TestResyncProfileValidationErrorEmptyName(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	_, err := h.client.Config.Resync(ctx, sdk.ResyncProfile{Name: ""})
	require.Error(t, err)
	assert.Equal(t, int64(0), h.commandCount(t))
}

func TestRemoveProfileValidationErrorEmptyName(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	_, err := h.client.Config.RemoveProfile(ctx, "")
	require.Error(t, err)
	assert.Equal(t, int64(0), h.commandCount(t))
}

func TestPITRDeleteValidationErrorFuture(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	future := time.Now().Add(24 * time.Hour)
	_, err := h.client.PITR.Delete(ctx, sdk.DeletePITRBefore{OlderThan: future})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "future")
	assert.Equal(t, int64(0), h.commandCount(t))
}

func TestDeleteBackupsValidationErrorFuture(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	future := time.Now().Add(24 * time.Hour)
	_, err := h.client.Backups.Delete(ctx, sdk.DeleteBackupsBefore{OlderThan: future})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "future")
	assert.Equal(t, int64(0), h.commandCount(t))
}

func TestDeleteBackupsOlderThanNegative(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	_, err := h.client.Backups.Delete(ctx, sdk.DeleteBackupsOlderThan{
		OlderThan: -time.Hour,
	})
	require.Error(t, err)
	// Negative duration is converted to a future cutoff time, which
	// DeleteBackupsBefore.Validate() rejects as "in the future".
	assert.Contains(t, err.Error(), "future")
	assert.Equal(t, int64(0), h.commandCount(t))
}
