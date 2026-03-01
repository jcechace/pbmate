//go:build integration

package integtest

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/percona-backup-mongodb/pbm/ctrl"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// --- Config.SetProfile ---

func TestConfigSetProfile(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// SetProfile dispatches a command — seed a main config first
	// so that PBM has something to anchor against.
	h.seedConfig(t, newMainConfig())

	yamlStr := `
storage:
  type: filesystem
  filesystem:
    path: /tmp/profile-backups
`
	result, err := h.client.Config.SetProfile(ctx, "new-profile", strings.NewReader(yamlStr))
	require.NoError(t, err)
	assert.NotEmpty(t, result.OPID)

	cmd := h.lastCommand(t)
	assert.Equal(t, ctrl.CmdAddConfigProfile, cmd.Cmd)
	require.NotNil(t, cmd.Profile)
	assert.Equal(t, "new-profile", cmd.Profile.Name)
	assert.True(t, cmd.Profile.IsProfile)
}

// --- Config.RemoveProfile ---

func TestConfigRemoveProfile(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	result, err := h.client.Config.RemoveProfile(ctx, "old-profile")
	require.NoError(t, err)
	assert.NotEmpty(t, result.OPID)

	cmd := h.lastCommand(t)
	assert.Equal(t, ctrl.CmdRemoveConfigProfile, cmd.Cmd)
	require.NotNil(t, cmd.Profile)
	assert.Equal(t, "old-profile", cmd.Profile.Name)
}

// --- Config.Resync ---

func TestConfigResyncMain(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	result, err := h.client.Config.Resync(ctx, sdk.ResyncMain{})
	require.NoError(t, err)
	assert.NotEmpty(t, result.OPID)

	cmd := h.lastCommand(t)
	assert.Equal(t, ctrl.CmdResync, cmd.Cmd)
	// ResyncMain without IncludeRestores -> nil Resync sub-document.
	assert.Nil(t, cmd.Resync)
}

func TestConfigResyncMainWithRestores(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	result, err := h.client.Config.Resync(ctx, sdk.ResyncMain{IncludeRestores: true})
	require.NoError(t, err)
	assert.NotEmpty(t, result.OPID)

	cmd := h.lastCommand(t)
	assert.Equal(t, ctrl.CmdResync, cmd.Cmd)
	require.NotNil(t, cmd.Resync)
	assert.True(t, cmd.Resync.IncludeRestores)
}

func TestConfigResyncProfile(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	result, err := h.client.Config.Resync(ctx, sdk.ResyncProfile{
		Name:  "my-s3",
		Clear: true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.OPID)

	cmd := h.lastCommand(t)
	assert.Equal(t, ctrl.CmdResync, cmd.Cmd)
	require.NotNil(t, cmd.Resync)
	assert.Equal(t, "my-s3", cmd.Resync.Name)
	assert.True(t, cmd.Resync.Clear)
}

func TestConfigResyncAllProfiles(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	result, err := h.client.Config.Resync(ctx, sdk.ResyncAllProfiles{Clear: false})
	require.NoError(t, err)
	assert.NotEmpty(t, result.OPID)

	cmd := h.lastCommand(t)
	assert.Equal(t, ctrl.CmdResync, cmd.Cmd)
	require.NotNil(t, cmd.Resync)
	assert.True(t, cmd.Resync.All)
	assert.False(t, cmd.Resync.Clear)
}
