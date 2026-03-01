//go:build integration

package integtest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/percona-backup-mongodb/pbm/ctrl"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// --- PITR.Delete ---

func TestPITRDeleteBefore(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	cutoff := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	result, err := h.client.PITR.Delete(ctx, sdk.DeletePITRBefore{
		OlderThan: cutoff,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.OPID)

	cmd := h.lastCommand(t)
	assert.Equal(t, ctrl.CmdDeletePITR, cmd.Cmd)
	require.NotNil(t, cmd.DeletePITR)
	assert.Equal(t, cutoff.Unix(), cmd.DeletePITR.OlderThan)
}

func TestPITRDeleteOlderThan(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	before := time.Now().UTC()
	result, err := h.client.PITR.Delete(ctx, sdk.DeletePITROlderThan{
		OlderThan: 3 * 24 * time.Hour,
	})
	after := time.Now().UTC()
	require.NoError(t, err)
	assert.NotEmpty(t, result.OPID)

	cmd := h.lastCommand(t)
	assert.Equal(t, ctrl.CmdDeletePITR, cmd.Cmd)
	require.NotNil(t, cmd.DeletePITR)

	// The cutoff should be approximately now - 3 days.
	expectedCutoff := before.Add(-3 * 24 * time.Hour).Unix()
	lateCutoff := after.Add(-3 * 24 * time.Hour).Unix()
	assert.GreaterOrEqual(t, cmd.DeletePITR.OlderThan, expectedCutoff)
	assert.LessOrEqual(t, cmd.DeletePITR.OlderThan, lateCutoff)
}

func TestPITRDeleteOlderThanZero(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Zero duration = delete all (cutoff = now).
	before := time.Now().UTC()
	result, err := h.client.PITR.Delete(ctx, sdk.DeletePITROlderThan{
		OlderThan: 0,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.OPID)

	cmd := h.lastCommand(t)
	require.NotNil(t, cmd.DeletePITR)
	assert.InDelta(t, before.Unix(), cmd.DeletePITR.OlderThan, 2)
}
