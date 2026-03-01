//go:build integration

package integtest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/percona-backup-mongodb/pbm/defs"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

func TestPITRTimelines(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed contiguous chunks: [1000, 1500] and [1500, 2000].
	// PBM merges these into a single timeline [1000, 2000].
	h.seedPITRChunk(t, newPITRChunk("rs", 1000, 1500))
	h.seedPITRChunk(t, newPITRChunk("rs", 1500, 2000))

	timelines, err := h.client.PITR.Timelines(ctx)
	require.NoError(t, err)
	require.Len(t, timelines, 1)

	assert.Equal(t, uint32(1000), timelines[0].Start.T)
	assert.Equal(t, uint32(2000), timelines[0].End.T)
}

func TestPITRTimelinesEmpty(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	timelines, err := h.client.PITR.Timelines(ctx)
	require.NoError(t, err)
	assert.Empty(t, timelines)
}

func TestPITRTimelinesGap(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Two non-contiguous chunks create separate timelines.
	h.seedPITRChunk(t, newPITRChunk("rs", 1000, 1500))
	h.seedPITRChunk(t, newPITRChunk("rs", 2000, 2500))

	timelines, err := h.client.PITR.Timelines(ctx)
	require.NoError(t, err)
	require.Len(t, timelines, 2)

	// PBM returns timelines in chronological order.
	assert.Equal(t, uint32(1000), timelines[0].Start.T)
	assert.Equal(t, uint32(1500), timelines[0].End.T)
	assert.Equal(t, uint32(2000), timelines[1].Start.T)
	assert.Equal(t, uint32(2500), timelines[1].End.T)
}

func TestPITRBases(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	target := sdk.Timestamp{T: 1500}

	// Seed a timeline covering [1000, 2000].
	h.seedPITRChunk(t, newPITRChunk("rs", 1000, 1500))
	h.seedPITRChunk(t, newPITRChunk("rs", 1500, 2000))

	// Valid base: done, main config, LastWriteTS(1200) before target(1500), within timeline.
	h.seedBackup(t, newBackupMeta("2024-01-01T00:00:00Z",
		withBackupStatus(defs.StatusDone),
		withBackupLastWriteTS(1200, 0),
		withBackupStartTS(1100),
	))

	// Invalid: LastWriteTS(1600) after target(1500).
	h.seedBackup(t, newBackupMeta("2024-01-02T00:00:00Z",
		withBackupStatus(defs.StatusDone),
		withBackupLastWriteTS(1600, 0),
		withBackupStartTS(1500),
	))

	bases, err := h.client.PITR.Bases(ctx, target)
	require.NoError(t, err)
	require.Len(t, bases, 1)
	assert.Equal(t, "2024-01-01T00:00:00Z", bases[0].Name)
}

func TestPITRBasesEmpty(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	target := sdk.Timestamp{T: 1500}

	// Seed timeline but no backups at all.
	h.seedPITRChunk(t, newPITRChunk("rs", 1000, 2000))

	bases, err := h.client.PITR.Bases(ctx, target)
	require.NoError(t, err)
	assert.Empty(t, bases)
}

func TestPITRBasesFiltersSelective(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	target := sdk.Timestamp{T: 1500}

	h.seedPITRChunk(t, newPITRChunk("rs", 1000, 2000))

	// Selective backup (has namespaces) should be excluded from PITR bases.
	h.seedBackup(t, newBackupMeta("2024-01-01T00:00:00Z",
		withBackupStatus(defs.StatusDone),
		withBackupLastWriteTS(1200, 0),
		withBackupStartTS(1100),
		withBackupNamespaces("db1.coll1"),
	))

	bases, err := h.client.PITR.Bases(ctx, target)
	require.NoError(t, err)
	assert.Empty(t, bases)
}

func TestPITRBasesFiltersProfile(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	target := sdk.Timestamp{T: 1500}

	h.seedPITRChunk(t, newPITRChunk("rs", 1000, 2000))

	// Profile backup should be excluded (PITR chunks are stored on main config only).
	h.seedBackup(t, newBackupMeta("2024-01-01T00:00:00Z",
		withBackupStatus(defs.StatusDone),
		withBackupLastWriteTS(1200, 0),
		withBackupStartTS(1100),
		withBackupProfile("my-s3"),
	))

	bases, err := h.client.PITR.Bases(ctx, target)
	require.NoError(t, err)
	assert.Empty(t, bases)
}

func TestPITRStatusDisabled(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed config without PITR section — defaults to disabled.
	h.seedConfig(t, newMainConfig())

	status, err := h.client.PITR.Status(ctx)
	require.NoError(t, err)
	assert.False(t, status.Enabled)
	assert.False(t, status.Running)
	assert.Empty(t, status.Nodes)
}

func TestPITRStatusEnabledNotRunning(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed config with PITR enabled but no active PITR locks.
	h.seedConfig(t, newMainConfig(withConfigPITR(true)))

	status, err := h.client.PITR.Status(ctx)
	require.NoError(t, err)
	assert.True(t, status.Enabled)
	assert.False(t, status.Running)
	assert.Empty(t, status.Nodes)
}

func TestPITREnableDisable(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed a config (PITR disabled by default).
	h.seedConfig(t, newMainConfig())

	// Verify initially disabled.
	status, err := h.client.PITR.Status(ctx)
	require.NoError(t, err)
	require.False(t, status.Enabled)

	// Enable PITR.
	err = h.client.PITR.Enable(ctx)
	require.NoError(t, err)

	status, err = h.client.PITR.Status(ctx)
	require.NoError(t, err)
	assert.True(t, status.Enabled)

	// Disable PITR.
	err = h.client.PITR.Disable(ctx)
	require.NoError(t, err)

	status, err = h.client.PITR.Status(ctx)
	require.NoError(t, err)
	assert.False(t, status.Enabled)
}
