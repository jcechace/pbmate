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

// --- Backup.Start ---

func TestBackupStartLogical(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	profileName, err := sdk.NewConfigName("my-s3")
	require.NoError(t, err)
	compLevel := 5
	parallel := 4

	result, err := h.client.Backups.Start(ctx, sdk.StartLogicalBackup{
		ConfigName:       profileName,
		Compression:      sdk.CompressionTypeZSTD,
		CompressionLevel: &compLevel,
		Namespaces:       []string{"db1.*", "db2.*"},
		UsersAndRoles:    true,
		NumParallelColls: &parallel,
	})
	require.NoError(t, err)

	// Result fields.
	assert.NotEmpty(t, result.OPID)
	assert.NotEmpty(t, result.Name)

	// Name is RFC 3339 formatted.
	_, parseErr := time.Parse(time.RFC3339, result.Name)
	assert.NoError(t, parseErr, "backup name should be RFC 3339")

	// Verify the dispatched command.
	cmd := h.lastCommand(t)
	assert.Equal(t, ctrl.CmdBackup, cmd.Cmd)
	require.NotNil(t, cmd.Backup)
	assert.Equal(t, defs.LogicalBackup, cmd.Backup.Type)
	assert.Equal(t, result.Name, cmd.Backup.Name)
	assert.Equal(t, []string{"db1.*", "db2.*"}, cmd.Backup.Namespaces)
	assert.True(t, cmd.Backup.UsersAndRoles)
	assert.Equal(t, "zstd", string(cmd.Backup.Compression))
	require.NotNil(t, cmd.Backup.CompressionLevel)
	assert.Equal(t, 5, *cmd.Backup.CompressionLevel)
	require.NotNil(t, cmd.Backup.NumParallelColls)
	assert.Equal(t, int32(4), *cmd.Backup.NumParallelColls)
	assert.Equal(t, "my-s3", cmd.Backup.Profile)
}

func TestBackupStartPhysical(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	result, err := h.client.Backups.Start(ctx, sdk.StartPhysicalBackup{
		Compression: sdk.CompressionTypeGZIP,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.OPID)
	assert.NotEmpty(t, result.Name)

	cmd := h.lastCommand(t)
	assert.Equal(t, ctrl.CmdBackup, cmd.Cmd)
	require.NotNil(t, cmd.Backup)
	assert.Equal(t, defs.PhysicalBackup, cmd.Backup.Type)
	assert.Equal(t, result.Name, cmd.Backup.Name)
	assert.Equal(t, "gzip", string(cmd.Backup.Compression))
	// Physical backups have no namespaces or parallel colls.
	assert.Empty(t, cmd.Backup.Namespaces)
	assert.Nil(t, cmd.Backup.NumParallelColls)
}

func TestBackupStartIncremental(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	result, err := h.client.Backups.Start(ctx, sdk.StartIncrementalBackup{
		Base: true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.OPID)

	cmd := h.lastCommand(t)
	assert.Equal(t, ctrl.CmdBackup, cmd.Cmd)
	require.NotNil(t, cmd.Backup)
	assert.Equal(t, defs.IncrementalBackup, cmd.Backup.Type)
	assert.True(t, cmd.Backup.IncrBase)
	assert.Equal(t, result.Name, cmd.Backup.Name)
}

func TestBackupStartIncrementalNonBase(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	result, err := h.client.Backups.Start(ctx, sdk.StartIncrementalBackup{
		Base: false,
	})
	require.NoError(t, err)

	cmd := h.lastCommand(t)
	require.NotNil(t, cmd.Backup)
	assert.False(t, cmd.Backup.IncrBase)
	assert.Equal(t, result.Name, cmd.Backup.Name)
}

func TestBackupStartDefaultCompression(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	_, err := h.client.Backups.Start(ctx, sdk.StartLogicalBackup{})
	require.NoError(t, err)

	cmd := h.lastCommand(t)
	require.NotNil(t, cmd.Backup)
	// Zero-value CompressionType maps to "" (server default).
	assert.Empty(t, string(cmd.Backup.Compression))
	assert.Nil(t, cmd.Backup.CompressionLevel)
	assert.Nil(t, cmd.Backup.NumParallelColls)
	assert.Empty(t, cmd.Backup.Profile) // main config
}

// --- Backup.Cancel ---

func TestBackupCancel(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	result, err := h.client.Backups.Cancel(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, result.OPID)

	cmd := h.lastCommand(t)
	assert.Equal(t, ctrl.CmdCancelBackup, cmd.Cmd)
	// Cancel has no sub-documents.
	assert.Nil(t, cmd.Backup)
	assert.Nil(t, cmd.Delete)
	assert.Nil(t, cmd.Restore)
}

// --- Backup.Delete ---

func TestBackupDeleteByName(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	result, err := h.client.Backups.Delete(ctx, sdk.DeleteBackupByName{
		Name: "2024-01-01T00:00:00Z",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.OPID)

	cmd := h.lastCommand(t)
	assert.Equal(t, ctrl.CmdDeleteBackup, cmd.Cmd)
	require.NotNil(t, cmd.Delete)
	assert.Equal(t, "2024-01-01T00:00:00Z", cmd.Delete.Backup)
}

func TestBackupDeleteBefore(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	cutoff := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	profileName, err := sdk.NewConfigName("my-s3")
	require.NoError(t, err)

	result, err := h.client.Backups.Delete(ctx, sdk.DeleteBackupsBefore{
		OlderThan:  cutoff,
		Type:       sdk.BackupTypeLogical,
		ConfigName: profileName,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.OPID)

	cmd := h.lastCommand(t)
	assert.Equal(t, ctrl.CmdDeleteBackup, cmd.Cmd)
	require.NotNil(t, cmd.Delete)
	assert.Equal(t, cutoff.Unix(), cmd.Delete.OlderThan)
	assert.Equal(t, defs.LogicalBackup, cmd.Delete.Type)
	assert.Equal(t, "my-s3", cmd.Delete.Profile)
}

func TestBackupDeleteOlderThan(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	before := time.Now().UTC()
	result, err := h.client.Backups.Delete(ctx, sdk.DeleteBackupsOlderThan{
		OlderThan: 7 * 24 * time.Hour,
	})
	after := time.Now().UTC()
	require.NoError(t, err)
	assert.NotEmpty(t, result.OPID)

	cmd := h.lastCommand(t)
	assert.Equal(t, ctrl.CmdDeleteBackup, cmd.Cmd)
	require.NotNil(t, cmd.Delete)

	// The cutoff should be approximately now - 7 days.
	expectedCutoff := before.Add(-7 * 24 * time.Hour).Unix()
	lateCutoff := after.Add(-7 * 24 * time.Hour).Unix()
	assert.GreaterOrEqual(t, cmd.Delete.OlderThan, expectedCutoff)
	assert.LessOrEqual(t, cmd.Delete.OlderThan, lateCutoff)
}

func TestBackupDeleteOlderThanZero(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Zero duration = delete all (cutoff = now).
	before := time.Now().UTC()
	result, err := h.client.Backups.Delete(ctx, sdk.DeleteBackupsOlderThan{
		OlderThan: 0,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.OPID)

	cmd := h.lastCommand(t)
	require.NotNil(t, cmd.Delete)
	// Cutoff should be approximately now (within a second).
	assert.InDelta(t, before.Unix(), cmd.Delete.OlderThan, 2)
}

// --- Backup.CanDelete ---

func TestBackupCanDeleteSuccess(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// CanDelete checks PITR requirements, which reads the config.
	h.seedConfig(t, newMainConfig())

	// Seed a completed logical backup.
	h.seedBackup(t, newBackupMeta("2024-01-01T00:00:00Z",
		withBackupStatus(defs.StatusDone),
		withBackupStartTS(100),
	))

	err := h.client.Backups.CanDelete(ctx, "2024-01-01T00:00:00Z")
	assert.NoError(t, err)
}

func TestBackupCanDeleteNotFound(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	err := h.client.Backups.CanDelete(ctx, "nonexistent")
	assert.ErrorIs(t, err, sdk.ErrNotFound)
}

func TestBackupCanDeleteInProgress(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed a running backup.
	h.seedBackup(t, newBackupMeta("2024-01-01T00:00:00Z",
		withBackupStatus(defs.StatusRunning),
		withBackupStartTS(100),
	))

	err := h.client.Backups.CanDelete(ctx, "2024-01-01T00:00:00Z")
	assert.ErrorIs(t, err, sdk.ErrBackupInProgress)
}

func TestBackupCanDeleteIncrementalNonBase(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// CanDelete checks PITR requirements, which reads the config.
	h.seedConfig(t, newMainConfig())

	// Seed a base and a child increment.
	h.seedBackup(t, newBackupMeta("2024-01-01T00:00:00Z",
		withBackupType(defs.IncrementalBackup),
		withBackupStatus(defs.StatusDone),
		withBackupStartTS(200),
	))
	h.seedBackup(t, newBackupMeta("2024-01-02T00:00:00Z",
		withBackupType(defs.IncrementalBackup),
		withBackupSrcBackup("2024-01-01T00:00:00Z"),
		withBackupStatus(defs.StatusDone),
		withBackupStartTS(100),
	))

	// Can delete the base.
	err := h.client.Backups.CanDelete(ctx, "2024-01-01T00:00:00Z")
	assert.NoError(t, err)

	// Cannot delete the child (not the chain base).
	err = h.client.Backups.CanDelete(ctx, "2024-01-02T00:00:00Z")
	assert.ErrorIs(t, err, sdk.ErrNotChainBase)
}
