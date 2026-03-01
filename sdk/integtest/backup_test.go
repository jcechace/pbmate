//go:build integration

package integtest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/percona/percona-backup-mongodb/pbm/compress"
	"github.com/percona/percona-backup-mongodb/pbm/defs"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

func TestBackupList(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed 3 backups with different timestamps (newest first in start_ts).
	h.seedBackup(t, newBackupMeta("2024-01-03T00:00:00Z",
		withBackupStartTS(time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC).Unix()),
	))
	h.seedBackup(t, newBackupMeta("2024-01-02T00:00:00Z",
		withBackupStartTS(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC).Unix()),
	))
	h.seedBackup(t, newBackupMeta("2024-01-01T00:00:00Z",
		withBackupStartTS(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix()),
	))

	backups, err := h.client.Backups.List(ctx, sdk.ListBackupsOptions{})
	require.NoError(t, err)
	require.Len(t, backups, 3)

	// PBM sorts by start_ts descending — newest first.
	assert.Equal(t, "2024-01-03T00:00:00Z", backups[0].Name)
	assert.Equal(t, "2024-01-02T00:00:00Z", backups[1].Name)
	assert.Equal(t, "2024-01-01T00:00:00Z", backups[2].Name)
}

func TestBackupListEmpty(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	backups, err := h.client.Backups.List(ctx, sdk.ListBackupsOptions{})
	require.NoError(t, err)
	assert.Empty(t, backups)
}

func TestBackupListWithLimit(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		ts := time.Date(2024, 1, i, 0, 0, 0, 0, time.UTC)
		h.seedBackup(t, newBackupMeta(ts.Format(time.RFC3339),
			withBackupStartTS(ts.Unix()),
		))
	}

	backups, err := h.client.Backups.List(ctx, sdk.ListBackupsOptions{Limit: 2})
	require.NoError(t, err)
	require.Len(t, backups, 2)

	// Should return the 2 newest (sorted by start_ts desc).
	assert.Equal(t, "2024-01-05T00:00:00Z", backups[0].Name)
	assert.Equal(t, "2024-01-04T00:00:00Z", backups[1].Name)
}

func TestBackupListFilterByType(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedBackup(t, newBackupMeta("2024-01-01T00:00:00Z",
		withBackupType(defs.LogicalBackup),
		withBackupStartTS(300),
	))
	h.seedBackup(t, newBackupMeta("2024-01-02T00:00:00Z",
		withBackupType(defs.PhysicalBackup),
		withBackupStartTS(200),
	))
	h.seedBackup(t, newBackupMeta("2024-01-03T00:00:00Z",
		withBackupType(defs.IncrementalBackup),
		withBackupStartTS(100),
	))

	backups, err := h.client.Backups.List(ctx, sdk.ListBackupsOptions{
		Type: sdk.BackupTypePhysical,
	})
	require.NoError(t, err)
	require.Len(t, backups, 1)
	assert.Equal(t, "2024-01-02T00:00:00Z", backups[0].Name)
	assert.True(t, backups[0].Type.Equal(sdk.BackupTypePhysical))
}

func TestBackupListFilterByConfigName(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedBackup(t, newBackupMeta("2024-01-01T00:00:00Z",
		withBackupStartTS(200),
	)) // main config (Store.Name="")
	h.seedBackup(t, newBackupMeta("2024-01-02T00:00:00Z",
		withBackupProfile("my-s3"),
		withBackupStartTS(100),
	))

	// Filter for main config only.
	backups, err := h.client.Backups.List(ctx, sdk.ListBackupsOptions{
		ConfigName: sdk.MainConfig,
	})
	require.NoError(t, err)
	require.Len(t, backups, 1)
	assert.Equal(t, "2024-01-01T00:00:00Z", backups[0].Name)

	// Filter for profile.
	profileName, err := sdk.NewConfigName("my-s3")
	require.NoError(t, err)
	backups, err = h.client.Backups.List(ctx, sdk.ListBackupsOptions{
		ConfigName: profileName,
	})
	require.NoError(t, err)
	require.Len(t, backups, 1)
	assert.Equal(t, "2024-01-02T00:00:00Z", backups[0].Name)
}

func TestBackupGet(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedBackup(t, newBackupMeta("2024-06-15T10:30:00Z",
		withBackupType(defs.PhysicalBackup),
		withBackupStatus(defs.StatusDone),
		withBackupCompression(compress.CompressionTypeS2),
		withBackupStartTS(time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC).Unix()),
		withBackupLastWriteTS(1718444000, 5),
		withBackupSize(1024*1024, 2048*1024),
		withBackupNamespaces("db1.coll1", "db2.coll2"),
	))

	bk, err := h.client.Backups.Get(ctx, "2024-06-15T10:30:00Z")
	require.NoError(t, err)

	// Core fields.
	assert.Equal(t, "2024-06-15T10:30:00Z", bk.Name)
	assert.True(t, bk.Type.Equal(sdk.BackupTypePhysical))
	assert.True(t, bk.Status.Equal(sdk.StatusDone))
	assert.True(t, bk.Compression.Equal(sdk.CompressionTypeS2))

	// Timestamps.
	assert.Equal(t, time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC), bk.StartTS)
	assert.Equal(t, uint32(1718444000), bk.LastWriteTS.T)
	assert.Equal(t, uint32(5), bk.LastWriteTS.I)

	// Sizes.
	assert.Equal(t, int64(1024*1024), bk.Size)
	assert.Equal(t, int64(2048*1024), bk.SizeUncompressed)

	// Namespaces.
	assert.Equal(t, []string{"db1.coll1", "db2.coll2"}, bk.Namespaces)

	// Version fields.
	assert.Equal(t, "8.0.0", bk.MongoVersion)
	assert.Equal(t, "2.13.0", bk.PBMVersion)

	// Domain methods.
	assert.True(t, bk.IsPhysical())
	assert.False(t, bk.IsLogical())
	assert.False(t, bk.IsIncremental())
	assert.True(t, bk.IsSelective())
	assert.False(t, bk.InProgress())
}

func TestBackupGetSelectiveBackup(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedBackup(t, newBackupMeta("2024-01-01T00:00:00Z",
		withBackupNamespaces("mydb.mycoll"),
		withBackupStartTS(100),
	))

	bk, err := h.client.Backups.Get(ctx, "2024-01-01T00:00:00Z")
	require.NoError(t, err)
	assert.True(t, bk.IsSelective())
	assert.Equal(t, []string{"mydb.mycoll"}, bk.Namespaces)
}

func TestBackupGetNotFound(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	_, err := h.client.Backups.Get(ctx, "nonexistent")
	require.ErrorIs(t, err, sdk.ErrNotFound)
}

func TestBackupGetByOpID(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	opid := primitive.NewObjectID().Hex()
	meta := newBackupMeta("2024-03-01T00:00:00Z", withBackupStartTS(100))
	meta.OPID = opid
	h.seedBackup(t, meta)

	bk, err := h.client.Backups.GetByOpID(ctx, opid)
	require.NoError(t, err)
	assert.Equal(t, "2024-03-01T00:00:00Z", bk.Name)
	assert.Equal(t, opid, bk.OPID)
}

func TestBackupGetByOpIDNotFound(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	_, err := h.client.Backups.GetByOpID(ctx, "deadbeef")
	require.ErrorIs(t, err, sdk.ErrNotFound)
}

func TestBackupGetIncrementalChain(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed a base and a child increment.
	h.seedBackup(t, newBackupMeta("2024-01-01T00:00:00Z",
		withBackupType(defs.IncrementalBackup),
		withBackupStartTS(200),
	))
	h.seedBackup(t, newBackupMeta("2024-01-02T00:00:00Z",
		withBackupType(defs.IncrementalBackup),
		withBackupSrcBackup("2024-01-01T00:00:00Z"),
		withBackupStartTS(100),
	))

	// Get the base.
	base, err := h.client.Backups.Get(ctx, "2024-01-01T00:00:00Z")
	require.NoError(t, err)
	assert.True(t, base.IsIncremental())
	assert.True(t, base.IsIncrementalBase())
	assert.Empty(t, base.SrcBackup)

	// Get the child.
	child, err := h.client.Backups.Get(ctx, "2024-01-02T00:00:00Z")
	require.NoError(t, err)
	assert.True(t, child.IsIncremental())
	assert.False(t, child.IsIncrementalBase())
	assert.Equal(t, "2024-01-01T00:00:00Z", child.SrcBackup)
}

func TestBackupGetConfigName(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Main config backup (Store.Name="" in PBM).
	h.seedBackup(t, newBackupMeta("2024-01-01T00:00:00Z",
		withBackupStartTS(200),
	))
	// Profile backup.
	h.seedBackup(t, newBackupMeta("2024-01-02T00:00:00Z",
		withBackupProfile("aws-prod"),
		withBackupStartTS(100),
	))

	main, err := h.client.Backups.Get(ctx, "2024-01-01T00:00:00Z")
	require.NoError(t, err)
	assert.True(t, main.ConfigName.Equal(sdk.MainConfig))

	profile, err := h.client.Backups.Get(ctx, "2024-01-02T00:00:00Z")
	require.NoError(t, err)
	expected, _ := sdk.NewConfigName("aws-prod")
	assert.True(t, profile.ConfigName.Equal(expected))
}

func TestBackupGetErrorStatus(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	meta := newBackupMeta("2024-01-01T00:00:00Z",
		withBackupStatus(defs.StatusError),
		withBackupStartTS(100),
	)
	meta.Err = "storage unreachable"
	h.seedBackup(t, meta)

	bk, err := h.client.Backups.Get(ctx, "2024-01-01T00:00:00Z")
	require.NoError(t, err)
	assert.True(t, bk.Status.Equal(sdk.StatusError))
	assert.True(t, bk.Status.IsTerminal())
	assert.Equal(t, "storage unreachable", bk.Error)
}

func TestBackupListAllStatuses(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	tests := []struct {
		name      string
		pbmStatus defs.Status
		sdkStatus sdk.Status
	}{
		{"done", defs.StatusDone, sdk.StatusDone},
		{"error", defs.StatusError, sdk.StatusError},
		{"running", defs.StatusRunning, sdk.StatusRunning},
		{"starting", defs.StatusStarting, sdk.StatusStarting},
		{"cancelled", defs.StatusCancelled, sdk.StatusCancelled},
	}

	for i, tt := range tests {
		ts := int64(1000 - i) // descending start_ts
		h.seedBackup(t, newBackupMeta(tt.name,
			withBackupStatus(tt.pbmStatus),
			withBackupStartTS(ts),
		))
	}

	backups, err := h.client.Backups.List(ctx, sdk.ListBackupsOptions{})
	require.NoError(t, err)
	require.Len(t, backups, len(tests))

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.name, backups[i].Name)
			assert.True(t, backups[i].Status.Equal(tt.sdkStatus),
				"expected %s, got %s", tt.sdkStatus, backups[i].Status)
		})
	}
}
