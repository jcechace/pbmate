package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

func TestResyncFormResult_ToCommand(t *testing.T) {
	t.Run("main scope", func(t *testing.T) {
		r := &resyncFormResult{
			scope:           resyncScopeMain,
			includeRestores: true,
		}
		cmd := r.toCommand()
		main, ok := cmd.(sdk.ResyncMain)
		require.True(t, ok, "expected ResyncMain")
		assert.True(t, main.IncludeRestores)
	})

	t.Run("main scope without restores", func(t *testing.T) {
		r := &resyncFormResult{
			scope:           resyncScopeMain,
			includeRestores: false,
		}
		cmd := r.toCommand()
		main, ok := cmd.(sdk.ResyncMain)
		require.True(t, ok)
		assert.False(t, main.IncludeRestores)
	})

	t.Run("all profiles", func(t *testing.T) {
		r := &resyncFormResult{
			scope:       resyncScopeProfile,
			profileName: resyncProfileAll,
			clear:       true,
		}
		cmd := r.toCommand()
		all, ok := cmd.(sdk.ResyncAllProfiles)
		require.True(t, ok, "expected ResyncAllProfiles")
		assert.True(t, all.Clear)
	})

	t.Run("specific profile", func(t *testing.T) {
		r := &resyncFormResult{
			scope:       resyncScopeProfile,
			profileName: "my-s3",
			clear:       false,
		}
		cmd := r.toCommand()
		profile, ok := cmd.(sdk.ResyncProfile)
		require.True(t, ok, "expected ResyncProfile")
		assert.Equal(t, "my-s3", profile.Name)
		assert.False(t, profile.Clear)
	})
}
