//go:build integration

package integtest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/percona/percona-backup-mongodb/pbm/defs"
	"github.com/percona/percona-backup-mongodb/pbm/restore"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

func TestRestoreList(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed 3 restores with different timestamps (newest first in start_ts).
	h.seedRestore(t, newRestoreMeta("2024-01-03T00:00:00Z",
		withRestoreStartTS(time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC).Unix()),
	))
	h.seedRestore(t, newRestoreMeta("2024-01-02T00:00:00Z",
		withRestoreStartTS(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC).Unix()),
	))
	h.seedRestore(t, newRestoreMeta("2024-01-01T00:00:00Z",
		withRestoreStartTS(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix()),
	))

	restores, err := h.client.Restores.List(ctx, sdk.ListRestoresOptions{})
	require.NoError(t, err)
	require.Len(t, restores, 3)

	// PBM sorts by start_ts descending — newest first.
	assert.Equal(t, "2024-01-03T00:00:00Z", restores[0].Name)
	assert.Equal(t, "2024-01-02T00:00:00Z", restores[1].Name)
	assert.Equal(t, "2024-01-01T00:00:00Z", restores[2].Name)
}

func TestRestoreListEmpty(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	restores, err := h.client.Restores.List(ctx, sdk.ListRestoresOptions{})
	require.NoError(t, err)
	assert.Empty(t, restores)
}

func TestRestoreListWithLimit(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		ts := time.Date(2024, 1, i, 0, 0, 0, 0, time.UTC)
		h.seedRestore(t, newRestoreMeta(ts.Format(time.RFC3339),
			withRestoreStartTS(ts.Unix()),
		))
	}

	restores, err := h.client.Restores.List(ctx, sdk.ListRestoresOptions{Limit: 2})
	require.NoError(t, err)
	require.Len(t, restores, 2)

	// Should return the 2 newest (sorted by start_ts desc).
	assert.Equal(t, "2024-01-05T00:00:00Z", restores[0].Name)
	assert.Equal(t, "2024-01-04T00:00:00Z", restores[1].Name)
}

func TestRestoreGet(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	startTS := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	lastTransTS := time.Date(2024, 6, 15, 10, 45, 0, 0, time.UTC)

	h.seedRestore(t, newRestoreMeta("2024-06-15T10:30:00Z",
		withRestoreBackup("2024-06-14T00:00:00Z"),
		withRestoreType(defs.PhysicalBackup),
		withRestoreStatus(defs.StatusDone),
		withRestoreOPID("abc123"),
		withRestoreStartTS(startTS.Unix()),
		withRestoreLastTransitionTS(lastTransTS.Unix()),
		withRestorePITR(1718444000),
		withRestoreNamespaces("db1.coll1", "db2.coll2"),
		withRestoreReplsets(restore.RestoreReplset{
			Name:             "rs",
			Status:           defs.StatusDone,
			LastTransitionTS: lastTransTS.Unix(),
			Nodes: []restore.RestoreNode{
				{
					Name:             "rs/localhost:27017",
					Status:           defs.StatusDone,
					LastTransitionTS: lastTransTS.Unix(),
				},
			},
		}),
	))

	r, err := h.client.Restores.Get(ctx, "2024-06-15T10:30:00Z")
	require.NoError(t, err)

	// Core fields.
	assert.Equal(t, "2024-06-15T10:30:00Z", r.Name)
	assert.Equal(t, "abc123", r.OPID)
	assert.Equal(t, "2024-06-14T00:00:00Z", r.Backup)
	assert.True(t, r.Type.Equal(sdk.BackupTypePhysical))
	assert.True(t, r.Status.Equal(sdk.StatusDone))

	// Timestamps.
	assert.Equal(t, startTS, r.StartTS)
	assert.Equal(t, lastTransTS, r.LastTransitionTS)
	// FinishTS derived from LastTransitionTS on terminal status.
	assert.Equal(t, lastTransTS, r.FinishTS)

	// PITR target.
	assert.Equal(t, uint32(1718444000), r.PITRTarget.T)

	// Namespaces.
	assert.Equal(t, []string{"db1.coll1", "db2.coll2"}, r.Namespaces)

	// Replsets.
	require.Len(t, r.Replsets, 1)
	rs := r.Replsets[0]
	assert.Equal(t, "rs", rs.Name)
	assert.True(t, rs.Status.Equal(sdk.StatusDone))
	assert.Equal(t, lastTransTS, rs.LastTransitionTS)
	require.Len(t, rs.Nodes, 1)
	assert.Equal(t, "rs/localhost:27017", rs.Nodes[0].Name)
	assert.True(t, rs.Nodes[0].Status.Equal(sdk.StatusDone))

	// Domain methods.
	assert.False(t, r.InProgress())
	assert.Equal(t, lastTransTS.Sub(startTS), r.Duration())
}

func TestRestoreGetNotFound(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	_, err := h.client.Restores.Get(ctx, "nonexistent")
	require.ErrorIs(t, err, sdk.ErrNotFound)
}

func TestRestoreGetByOpID(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	opid := primitive.NewObjectID().Hex()
	h.seedRestore(t, newRestoreMeta("2024-03-01T00:00:00Z",
		withRestoreOPID(opid),
		withRestoreStartTS(100),
	))

	r, err := h.client.Restores.GetByOpID(ctx, opid)
	require.NoError(t, err)
	assert.Equal(t, "2024-03-01T00:00:00Z", r.Name)
	assert.Equal(t, opid, r.OPID)
}

func TestRestoreGetByOpIDNotFound(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	_, err := h.client.Restores.GetByOpID(ctx, "deadbeef")
	require.ErrorIs(t, err, sdk.ErrNotFound)
}

func TestRestoreGetPITRTarget(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedRestore(t, newRestoreMeta("2024-01-01T00:00:00Z",
		withRestorePITR(1700000000),
		withRestoreStartTS(100),
	))

	r, err := h.client.Restores.Get(ctx, "2024-01-01T00:00:00Z")
	require.NoError(t, err)
	assert.Equal(t, uint32(1700000000), r.PITRTarget.T)
	assert.Equal(t, uint32(0), r.PITRTarget.I)
}

func TestRestoreGetNoPITRTarget(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedRestore(t, newRestoreMeta("2024-01-01T00:00:00Z",
		withRestoreStartTS(100),
	))

	r, err := h.client.Restores.Get(ctx, "2024-01-01T00:00:00Z")
	require.NoError(t, err)
	assert.True(t, r.PITRTarget.IsZero())
}

func TestRestoreGetBcpChain(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedRestore(t, newRestoreMeta("2024-01-01T00:00:00Z",
		withRestoreType(defs.IncrementalBackup),
		withRestoreBcpChain("base-backup", "incr-1", "incr-2"),
		withRestoreStartTS(100),
	))

	r, err := h.client.Restores.Get(ctx, "2024-01-01T00:00:00Z")
	require.NoError(t, err)
	assert.Equal(t, []string{"base-backup", "incr-1", "incr-2"}, r.BcpChain)
	assert.True(t, r.Type.Equal(sdk.BackupTypeIncremental))
}

func TestRestoreGetDomainMethods(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Running restore (non-terminal).
	h.seedRestore(t, newRestoreMeta("running-restore",
		withRestoreStatus(defs.StatusRunning),
		withRestoreStartTS(time.Now().Add(-5*time.Minute).Unix()),
	))

	r, err := h.client.Restores.Get(ctx, "running-restore")
	require.NoError(t, err)
	assert.True(t, r.InProgress())
	assert.Zero(t, r.Duration(), "Duration should be zero for in-progress restore")
	assert.NotZero(t, r.Elapsed(), "Elapsed should be non-zero for in-progress restore")
}

func TestRestoreGetErrorStatus(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedRestore(t, newRestoreMeta("2024-01-01T00:00:00Z",
		withRestoreStatus(defs.StatusError),
		withRestoreError("connection lost"),
		withRestoreStartTS(100),
	))

	r, err := h.client.Restores.Get(ctx, "2024-01-01T00:00:00Z")
	require.NoError(t, err)
	assert.True(t, r.Status.Equal(sdk.StatusError))
	assert.True(t, r.Status.IsTerminal())
	assert.Equal(t, "connection lost", r.Error)
}

func TestRestoreListAllStatuses(t *testing.T) {
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
		h.seedRestore(t, newRestoreMeta(tt.name,
			withRestoreStatus(tt.pbmStatus),
			withRestoreStartTS(ts),
		))
	}

	restores, err := h.client.Restores.List(ctx, sdk.ListRestoresOptions{})
	require.NoError(t, err)
	require.Len(t, restores, len(tests))

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.name, restores[i].Name)
			assert.True(t, restores[i].Status.Equal(tt.sdkStatus),
				"expected %s, got %s", tt.sdkStatus, restores[i].Status)
		})
	}
}
