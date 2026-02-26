package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jcechace/pbmate/internal/config"
)

// newTestConfig creates a config with a temp file path for testing.
func newTestConfig(t *testing.T) (*config.AppConfig, configFilePath) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.AppConfig{
		Theme:          "default",
		CurrentContext: "prod",
		Contexts: map[string]config.Context{
			"prod":    {URI: "mongodb://prod:27017"},
			"staging": {URI: "mongodb://staging:27017", Theme: "latte"},
		},
	}
	require.NoError(t, cfg.Save(path))
	return cfg, configFilePath(path)
}

func TestCfgSetRun(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		context string
		check   func(t *testing.T, cfg *config.AppConfig)
		wantErr string
	}{
		{
			name:  "set global theme",
			key:   "theme",
			value: "mocha",
			check: func(t *testing.T, cfg *config.AppConfig) {
				assert.Equal(t, "mocha", cfg.Theme)
			},
		},
		{
			name:  "set global readonly",
			key:   "readonly",
			value: "true",
			check: func(t *testing.T, cfg *config.AppConfig) {
				assert.True(t, cfg.Readonly)
			},
		},
		{
			name:    "set context theme",
			key:     "theme",
			value:   "frappe",
			context: "staging",
			check: func(t *testing.T, cfg *config.AppConfig) {
				assert.Equal(t, "frappe", cfg.Contexts["staging"].Theme)
			},
		},
		{
			name:    "set uri with validation",
			key:     "uri",
			value:   "mongodb://newhost:27017",
			context: "prod",
			check: func(t *testing.T, cfg *config.AppConfig) {
				assert.Equal(t, "mongodb://newhost:27017", cfg.Contexts["prod"].URI)
			},
		},
		{
			name:    "set uri rejects invalid",
			key:     "uri",
			value:   "not-a-uri",
			context: "prod",
			wantErr: "invalid URI scheme",
		},
		{
			name:    "unknown context",
			key:     "theme",
			value:   "mocha",
			context: "nonexistent",
			wantErr: "not found",
		},
		{
			name:    "unknown key",
			key:     "nonexistent",
			value:   "x",
			wantErr: "unknown key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, path := newTestConfig(t)
			cmd := &cfgSetCmd{Key: tt.key, Value: tt.value, Context: tt.context}
			err := cmd.Run(cfg, path)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			tt.check(t, cfg)

			// Verify persistence: reload and check.
			loaded, err := config.Load(string(path))
			require.NoError(t, err)
			tt.check(t, loaded)
		})
	}
}

func TestCfgUnsetRun(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		context string
		check   func(t *testing.T, cfg *config.AppConfig)
		wantErr string
	}{
		{
			name: "unset global theme",
			key:  "theme",
			check: func(t *testing.T, cfg *config.AppConfig) {
				assert.Equal(t, "", cfg.Theme)
			},
		},
		{
			name:    "unset context theme",
			key:     "theme",
			context: "staging",
			check: func(t *testing.T, cfg *config.AppConfig) {
				assert.Equal(t, "", cfg.Contexts["staging"].Theme)
			},
		},
		{
			name:    "unknown context",
			key:     "theme",
			context: "nonexistent",
			wantErr: "not found",
		},
		{
			name:    "unknown key",
			key:     "nonexistent",
			wantErr: "unknown key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, path := newTestConfig(t)
			cmd := &cfgUnsetCmd{Key: tt.key, Context: tt.context}
			err := cmd.Run(cfg, path)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			tt.check(t, cfg)

			// Verify persistence.
			loaded, err := config.Load(string(path))
			require.NoError(t, err)
			tt.check(t, loaded)
		})
	}
}

func TestContextAddRunValidatesURI(t *testing.T) {
	cfg, path := newTestConfig(t)
	cmd := &contextAddCmd{Name: "bad", URI: "not-a-uri"}
	err := cmd.Run(cfg, path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid URI scheme")

	// Context should not have been added.
	_, exists := cfg.Contexts["bad"]
	assert.False(t, exists)
}

func TestContextAddRunValid(t *testing.T) {
	cfg, path := newTestConfig(t)
	cmd := &contextAddCmd{Name: "local", URI: "mongodb://localhost:27017"}
	err := cmd.Run(cfg, path)
	require.NoError(t, err)

	ctx, exists := cfg.Contexts["local"]
	require.True(t, exists)
	assert.Equal(t, "mongodb://localhost:27017", ctx.URI)

	// Verify persistence.
	loaded, err := config.Load(string(path))
	require.NoError(t, err)
	_, exists = loaded.Contexts["local"]
	assert.True(t, exists)
}

func TestCfgSetUnsetRoundTrip(t *testing.T) {
	cfg, path := newTestConfig(t)

	// Set a per-context readonly override.
	setCmd := &cfgSetCmd{Key: "readonly", Value: "true", Context: "prod"}
	require.NoError(t, setCmd.Run(cfg, path))
	require.NotNil(t, cfg.Contexts["prod"].Readonly)
	assert.True(t, *cfg.Contexts["prod"].Readonly)

	// Unset it — should go back to nil (inherit).
	unsetCmd := &cfgUnsetCmd{Key: "readonly", Context: "prod"}
	require.NoError(t, unsetCmd.Run(cfg, path))
	assert.Nil(t, cfg.Contexts["prod"].Readonly)

	// Verify persistence.
	loaded, err := config.Load(string(path))
	require.NoError(t, err)
	assert.Nil(t, loaded.Contexts["prod"].Readonly)
}
