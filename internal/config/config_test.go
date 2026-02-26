package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPath(t *testing.T) {
	t.Run("XDG set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/custom/config")
		path, err := DefaultPath()
		require.NoError(t, err)
		assert.Equal(t, "/custom/config/pbmate/config.yaml", path)
	})

	t.Run("XDG unset", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		path, err := DefaultPath()
		require.NoError(t, err)

		home, err := os.UserHomeDir()
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(home, ".config", "pbmate", "config.yaml"), path)
	})
}

func TestLoadMissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Empty(t, cfg.Contexts)
	assert.Empty(t, cfg.CurrentContext)
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config.yaml")

	readonly := true
	original := &AppConfig{
		Theme:          "mocha",
		Readonly:       false,
		CurrentContext: "production",
		Contexts: map[string]Context{
			"production": {URI: "mongodb://prod:27017"},
			"staging": {
				URI:      "mongodb://staging:27017",
				Theme:    "latte",
				Readonly: &readonly,
			},
		},
	}

	err := original.Save(path)
	require.NoError(t, err)

	loaded, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, original.Theme, loaded.Theme)
	assert.Equal(t, original.Readonly, loaded.Readonly)
	assert.Equal(t, original.CurrentContext, loaded.CurrentContext)
	assert.Equal(t, len(original.Contexts), len(loaded.Contexts))
	assert.Equal(t, "mongodb://prod:27017", loaded.Contexts["production"].URI)
	assert.Equal(t, "latte", loaded.Contexts["staging"].Theme)
	require.NotNil(t, loaded.Contexts["staging"].Readonly)
	assert.True(t, *loaded.Contexts["staging"].Readonly)
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte("{{invalid yaml"), 0o644)
	require.NoError(t, err)

	_, err = Load(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse config")
}

func TestCurrentCtx(t *testing.T) {
	cfg := &AppConfig{
		CurrentContext: "prod",
		Contexts: map[string]Context{
			"prod": {URI: "mongodb://prod:27017"},
		},
	}

	t.Run("found", func(t *testing.T) {
		ctx := cfg.CurrentCtx()
		require.NotNil(t, ctx)
		assert.Equal(t, "mongodb://prod:27017", ctx.URI)
	})

	t.Run("not found", func(t *testing.T) {
		cfg.CurrentContext = "missing"
		assert.Nil(t, cfg.CurrentCtx())
	})

	t.Run("empty", func(t *testing.T) {
		cfg.CurrentContext = ""
		assert.Nil(t, cfg.CurrentCtx())
	})
}

func TestResolveURI(t *testing.T) {
	cfg := &AppConfig{
		CurrentContext: "prod",
		Contexts: map[string]Context{
			"prod":    {URI: "mongodb://prod:27017"},
			"staging": {URI: "mongodb://staging:27017"},
		},
	}

	tests := []struct {
		name        string
		flagURI     string
		flagContext string
		want        string
		wantErr     bool
	}{
		{
			name:    "explicit URI wins",
			flagURI: "mongodb://explicit:27017",
			want:    "mongodb://explicit:27017",
		},
		{
			name:        "explicit URI wins over context flag",
			flagURI:     "mongodb://explicit:27017",
			flagContext: "staging",
			want:        "mongodb://explicit:27017",
		},
		{
			name:        "context flag overrides current",
			flagContext: "staging",
			want:        "mongodb://staging:27017",
		},
		{
			name: "current context used as fallback",
			want: "mongodb://prod:27017",
		},
		{
			name:        "unknown context flag",
			flagContext: "missing",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cfg.ResolveURI(tt.flagURI, tt.flagContext)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("no URI no context", func(t *testing.T) {
		empty := &AppConfig{}
		_, err := empty.ResolveURI("", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no URI provided")
	})
}

func TestResolveTheme(t *testing.T) {
	cfg := &AppConfig{
		Theme:          "frappe",
		CurrentContext: "prod",
		Contexts: map[string]Context{
			"prod":    {URI: "mongodb://prod:27017", Theme: "mocha"},
			"staging": {URI: "mongodb://staging:27017"},
		},
	}

	tests := []struct {
		name        string
		flagTheme   string
		flagContext string
		want        string
	}{
		{
			name:      "explicit flag wins",
			flagTheme: "latte",
			want:      "latte",
		},
		{
			name: "context override",
			want: "mocha", // prod context has theme: mocha
		},
		{
			name:        "context without override uses global",
			flagContext: "staging",
			want:        "frappe",
		},
		{
			name:        "flag wins over context",
			flagTheme:   "macchiato",
			flagContext: "prod",
			want:        "macchiato",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.ResolveTheme(tt.flagTheme, tt.flagContext)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("all empty returns default", func(t *testing.T) {
		empty := &AppConfig{}
		assert.Equal(t, "default", empty.ResolveTheme("", ""))
	})
}

func TestResolveReadonly(t *testing.T) {
	readonly := true
	notReadonly := false
	cfg := &AppConfig{
		Readonly:       false,
		CurrentContext: "prod",
		Contexts: map[string]Context{
			"prod":    {URI: "mongodb://prod:27017", Readonly: &readonly},
			"staging": {URI: "mongodb://staging:27017", Readonly: &notReadonly},
			"local":   {URI: "mongodb://local:27017"}, // nil Readonly
		},
	}

	t.Run("explicit flag wins", func(t *testing.T) {
		flag := false
		assert.False(t, cfg.ResolveReadonly(&flag, ""))
	})

	t.Run("context override true", func(t *testing.T) {
		assert.True(t, cfg.ResolveReadonly(nil, ""))
	})

	t.Run("context override false", func(t *testing.T) {
		assert.False(t, cfg.ResolveReadonly(nil, "staging"))
	})

	t.Run("context nil inherits global", func(t *testing.T) {
		assert.False(t, cfg.ResolveReadonly(nil, "local"))
	})

	t.Run("flag overrides context", func(t *testing.T) {
		flag := false
		assert.False(t, cfg.ResolveReadonly(&flag, "prod"))
	})
}

func TestValidateURI(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantErr string
	}{
		{
			name: "valid mongodb",
			uri:  "mongodb://localhost:27017",
		},
		{
			name: "valid mongodb+srv",
			uri:  "mongodb+srv://cluster.example.com",
		},
		{
			name: "valid with auth and options",
			uri:  "mongodb://user:pass@host1:27017,host2:27017/admin?replicaSet=rs0",
		},
		{
			name:    "no scheme",
			uri:     "localhost:27017",
			wantErr: "invalid URI scheme",
		},
		{
			name:    "wrong scheme",
			uri:     "http://localhost:27017",
			wantErr: "invalid URI scheme",
		},
		{
			name:    "empty string",
			uri:     "",
			wantErr: "invalid URI scheme",
		},
		{
			name:    "scheme only no host",
			uri:     "mongodb://",
			wantErr: "missing host",
		},
		{
			name:    "garbage",
			uri:     "garbage",
			wantErr: "invalid URI scheme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURI(tt.uri)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}
