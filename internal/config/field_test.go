package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetByPath(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }

	tests := []struct {
		name    string
		target  any
		path    string
		value   string
		check   func(t *testing.T, target any)
		wantErr string
	}{
		{
			name:   "AppConfig string field",
			target: &AppConfig{},
			path:   "theme",
			value:  "mocha",
			check: func(t *testing.T, target any) {
				assert.Equal(t, "mocha", target.(*AppConfig).Theme)
			},
		},
		{
			name:   "AppConfig bool field",
			target: &AppConfig{},
			path:   "readonly",
			value:  "true",
			check: func(t *testing.T, target any) {
				assert.True(t, target.(*AppConfig).Readonly)
			},
		},
		{
			name:   "AppConfig current-context",
			target: &AppConfig{},
			path:   "current-context",
			value:  "prod",
			check: func(t *testing.T, target any) {
				assert.Equal(t, "prod", target.(*AppConfig).CurrentContext)
			},
		},
		{
			name:   "Context string field",
			target: &Context{},
			path:   "uri",
			value:  "mongodb://localhost:27017",
			check: func(t *testing.T, target any) {
				assert.Equal(t, "mongodb://localhost:27017", target.(*Context).URI)
			},
		},
		{
			name:   "Context pointer bool field true",
			target: &Context{},
			path:   "readonly",
			value:  "true",
			check: func(t *testing.T, target any) {
				ctx := target.(*Context)
				require.NotNil(t, ctx.Readonly)
				assert.True(t, *ctx.Readonly)
			},
		},
		{
			name:   "Context pointer bool field false",
			target: &Context{Readonly: boolPtr(true)},
			path:   "readonly",
			value:  "false",
			check: func(t *testing.T, target any) {
				ctx := target.(*Context)
				require.NotNil(t, ctx.Readonly)
				assert.False(t, *ctx.Readonly)
			},
		},
		{
			name:    "invalid bool value",
			target:  &AppConfig{},
			path:    "readonly",
			value:   "notabool",
			wantErr: "invalid bool value",
		},
		{
			name:    "invalid pointer bool value",
			target:  &Context{},
			path:    "readonly",
			value:   "notabool",
			wantErr: "invalid bool value",
		},
		{
			name:    "unknown key on AppConfig",
			target:  &AppConfig{},
			path:    "nonexistent",
			value:   "x",
			wantErr: "unknown key: nonexistent",
		},
		{
			name:    "unknown key on Context",
			target:  &Context{},
			path:    "nonexistent",
			value:   "x",
			wantErr: "unknown key: nonexistent",
		},
		{
			name:    "cross-struct key uri on AppConfig",
			target:  &AppConfig{},
			path:    "uri",
			value:   "mongodb://localhost",
			wantErr: "unknown key: uri",
		},
		{
			name:    "cross-struct key current-context on Context",
			target:  &Context{},
			path:    "current-context",
			value:   "prod",
			wantErr: "unknown key: current-context",
		},
		{
			name:    "map field not settable",
			target:  &AppConfig{},
			path:    "contexts",
			value:   "anything",
			wantErr: "cannot set composite key",
		},
		{
			name:    "non-pointer target",
			target:  AppConfig{},
			path:    "theme",
			value:   "mocha",
			wantErr: "target must be a pointer to struct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SetByPath(tt.target, tt.path, tt.value)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			tt.check(t, tt.target)
		})
	}
}

func TestGetByPath(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }

	tests := []struct {
		name    string
		target  any
		path    string
		want    string
		wantErr string
	}{
		{
			name:   "string field",
			target: &AppConfig{Theme: "mocha"},
			path:   "theme",
			want:   "mocha",
		},
		{
			name:   "bool field true",
			target: &AppConfig{Readonly: true},
			path:   "readonly",
			want:   "true",
		},
		{
			name:   "bool field false",
			target: &AppConfig{},
			path:   "readonly",
			want:   "false",
		},
		{
			name:   "pointer bool set",
			target: &Context{Readonly: boolPtr(true)},
			path:   "readonly",
			want:   "true",
		},
		{
			name:   "pointer bool nil",
			target: &Context{},
			path:   "readonly",
			want:   "",
		},
		{
			name:   "empty string field",
			target: &AppConfig{},
			path:   "theme",
			want:   "",
		},
		{
			name:    "unknown key",
			target:  &AppConfig{},
			path:    "nonexistent",
			wantErr: "unknown key: nonexistent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetByPath(tt.target, tt.path)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUnsetByPath(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }

	tests := []struct {
		name    string
		target  any
		path    string
		check   func(t *testing.T, target any)
		wantErr string
	}{
		{
			name:   "unset string field",
			target: &AppConfig{Theme: "mocha"},
			path:   "theme",
			check: func(t *testing.T, target any) {
				assert.Equal(t, "", target.(*AppConfig).Theme)
			},
		},
		{
			name:   "unset bool field",
			target: &AppConfig{Readonly: true},
			path:   "readonly",
			check: func(t *testing.T, target any) {
				assert.False(t, target.(*AppConfig).Readonly)
			},
		},
		{
			name:   "unset pointer bool from true",
			target: &Context{Readonly: boolPtr(true)},
			path:   "readonly",
			check: func(t *testing.T, target any) {
				assert.Nil(t, target.(*Context).Readonly)
			},
		},
		{
			name:   "unset pointer bool from nil is idempotent",
			target: &Context{},
			path:   "readonly",
			check: func(t *testing.T, target any) {
				assert.Nil(t, target.(*Context).Readonly)
			},
		},
		{
			name:    "unknown key",
			target:  &AppConfig{},
			path:    "nonexistent",
			wantErr: "unknown key: nonexistent",
		},
		{
			name:    "map field not unsettable",
			target:  &AppConfig{},
			path:    "contexts",
			wantErr: "cannot unset composite key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UnsetByPath(tt.target, tt.path)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			tt.check(t, tt.target)
		})
	}
}

func TestSetGetRoundTrip(t *testing.T) {
	cfg := &AppConfig{}

	require.NoError(t, SetByPath(cfg, "theme", "frappe"))
	got, err := GetByPath(cfg, "theme")
	require.NoError(t, err)
	assert.Equal(t, "frappe", got)

	require.NoError(t, SetByPath(cfg, "readonly", "true"))
	got, err = GetByPath(cfg, "readonly")
	require.NoError(t, err)
	assert.Equal(t, "true", got)
}
