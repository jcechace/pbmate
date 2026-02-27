package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// --- parseOptionalInt ---

func TestParseOptionalInt(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantN  *int
		wantOK bool // true means non-nil result
	}{
		{"empty string", "", nil, false},
		{"whitespace", "  ", nil, false},
		{"non-numeric", "abc", nil, false},
		{"zero", "0", nil, false},
		{"negative", "-1", nil, false},
		{"positive", "4", intPtr(4), true},
		{"leading space", " 8 ", intPtr(8), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseOptionalInt(tt.input)
			if tt.wantOK {
				require.NotNil(t, got)
				assert.Equal(t, *tt.wantN, *got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

// --- backupFormResult.toCommand ---

func TestBackupFormResultToCommandLogical(t *testing.T) {
	t.Run("default logical backup", func(t *testing.T) {
		r := backupFormResult{
			backupType:  "logical",
			compression: "default",
			configName:  defaultConfigName,
		}
		cmd := r.toCommand()
		logical, ok := cmd.(sdk.StartLogicalBackup)
		require.True(t, ok, "expected StartLogicalBackup")
		assert.True(t, logical.ConfigName.IsZero(), "main config should be zero ConfigName")
		assert.True(t, logical.Compression.IsZero(), "default compression should be zero")
		assert.Nil(t, logical.Namespaces)
		assert.Nil(t, logical.NumParallelColls)
	})

	t.Run("logical with namespaces", func(t *testing.T) {
		r := backupFormResult{
			backupType:  "logical",
			compression: "default",
			configName:  defaultConfigName,
			namespaces:  "db1.col1, db2.col2",
		}
		cmd := r.toCommand()
		logical, ok := cmd.(sdk.StartLogicalBackup)
		require.True(t, ok)
		assert.Equal(t, []string{"db1.col1", "db2.col2"}, logical.Namespaces)
	})

	t.Run("logical with wildcard namespaces treated as full", func(t *testing.T) {
		r := backupFormResult{
			backupType:  "logical",
			compression: "default",
			configName:  defaultConfigName,
			namespaces:  "*.*",
		}
		cmd := r.toCommand()
		logical, ok := cmd.(sdk.StartLogicalBackup)
		require.True(t, ok)
		assert.Nil(t, logical.Namespaces)
	})

	t.Run("logical with non-default compression", func(t *testing.T) {
		r := backupFormResult{
			backupType:  "logical",
			compression: "zstd",
			configName:  defaultConfigName,
		}
		cmd := r.toCommand()
		logical, ok := cmd.(sdk.StartLogicalBackup)
		require.True(t, ok)
		assert.Equal(t, "zstd", logical.Compression.String())
	})

	t.Run("logical with non-main profile", func(t *testing.T) {
		r := backupFormResult{
			backupType:  "logical",
			compression: "default",
			configName:  "my-profile",
		}
		cmd := r.toCommand()
		logical, ok := cmd.(sdk.StartLogicalBackup)
		require.True(t, ok)
		assert.Equal(t, "my-profile", logical.ConfigName.String())
	})

	t.Run("logical with parallel colls", func(t *testing.T) {
		r := backupFormResult{
			backupType:    "logical",
			compression:   "default",
			configName:    defaultConfigName,
			parallelColls: "8",
		}
		cmd := r.toCommand()
		logical, ok := cmd.(sdk.StartLogicalBackup)
		require.True(t, ok)
		require.NotNil(t, logical.NumParallelColls)
		assert.Equal(t, 8, *logical.NumParallelColls)
	})
}

func TestBackupFormResultToCommandIncremental(t *testing.T) {
	t.Run("incremental backup", func(t *testing.T) {
		r := backupFormResult{
			backupType:  "incremental",
			compression: "s2",
			configName:  defaultConfigName,
			incrBase:    true,
		}
		cmd := r.toCommand()
		incr, ok := cmd.(sdk.StartIncrementalBackup)
		require.True(t, ok, "expected StartIncrementalBackup")
		assert.True(t, incr.Base)
		assert.Equal(t, "s2", incr.Compression.String())
	})

	t.Run("incremental extend chain", func(t *testing.T) {
		r := backupFormResult{
			backupType:  "incremental",
			compression: "default",
			configName:  defaultConfigName,
			incrBase:    false,
		}
		cmd := r.toCommand()
		incr, ok := cmd.(sdk.StartIncrementalBackup)
		require.True(t, ok)
		assert.False(t, incr.Base)
	})
}

func TestBackupFormResultToCommandPhysical(t *testing.T) {
	t.Run("default physical backup", func(t *testing.T) {
		r := backupFormResult{
			backupType:  "physical",
			compression: "default",
			configName:  defaultConfigName,
		}
		cmd := r.toCommand()
		physical, ok := cmd.(sdk.StartPhysicalBackup)
		require.True(t, ok, "expected StartPhysicalBackup")
		assert.True(t, physical.ConfigName.IsZero(), "main config should be zero ConfigName")
		assert.True(t, physical.Compression.IsZero(), "default compression should be zero")
	})

	t.Run("physical with compression and profile", func(t *testing.T) {
		r := backupFormResult{
			backupType:  "physical",
			compression: "zstd",
			configName:  "my-s3",
		}
		cmd := r.toCommand()
		physical, ok := cmd.(sdk.StartPhysicalBackup)
		require.True(t, ok)
		assert.Equal(t, "zstd", physical.Compression.String())
		assert.Equal(t, "my-s3", physical.ConfigName.String())
	})
}

// --- hasIncrementalChain ---

func TestHasIncrementalChain(t *testing.T) {
	profileCN, _ := sdk.NewConfigName("my-profile")

	mainIncrDone := sdk.Backup{
		Name:       "2026-01-01T00:00:00Z",
		Type:       sdk.BackupTypeIncremental,
		Status:     sdk.StatusDone,
		ConfigName: sdk.MainConfig,
	}
	profileIncrDone := sdk.Backup{
		Name:       "2026-01-02T00:00:00Z",
		Type:       sdk.BackupTypeIncremental,
		Status:     sdk.StatusDone,
		ConfigName: profileCN,
	}
	mainLogicalDone := sdk.Backup{
		Name:       "2026-01-03T00:00:00Z",
		Type:       sdk.BackupTypeLogical,
		Status:     sdk.StatusDone,
		ConfigName: sdk.MainConfig,
	}
	mainIncrError := sdk.Backup{
		Name:       "2026-01-04T00:00:00Z",
		Type:       sdk.BackupTypeIncremental,
		Status:     sdk.StatusError,
		ConfigName: sdk.MainConfig,
	}

	tests := []struct {
		name       string
		backups    []sdk.Backup
		configName string
		want       bool
	}{
		{
			name:       "no backups",
			backups:    nil,
			configName: defaultConfigName,
			want:       false,
		},
		{
			name:       "incremental on main profile",
			backups:    []sdk.Backup{mainIncrDone},
			configName: defaultConfigName,
			want:       true,
		},
		{
			name:       "incremental on named profile",
			backups:    []sdk.Backup{profileIncrDone},
			configName: "my-profile",
			want:       true,
		},
		{
			name:       "incremental on wrong profile",
			backups:    []sdk.Backup{profileIncrDone},
			configName: defaultConfigName,
			want:       false,
		},
		{
			name:       "main profile no match on named",
			backups:    []sdk.Backup{mainIncrDone},
			configName: "my-profile",
			want:       false,
		},
		{
			name:       "only logical on main",
			backups:    []sdk.Backup{mainLogicalDone},
			configName: defaultConfigName,
			want:       false,
		},
		{
			name:       "only errored incremental",
			backups:    []sdk.Backup{mainIncrError},
			configName: defaultConfigName,
			want:       false,
		},
		{
			name:       "mixed backups finds incremental",
			backups:    []sdk.Backup{mainLogicalDone, mainIncrError, mainIncrDone},
			configName: defaultConfigName,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasIncrementalChain(tt.backups, tt.configName)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- newFullBackupForm chain awareness ---

func TestFullBackupFormIncrementalNoChain(t *testing.T) {
	// When no incremental chain exists, the form should force incrBase = true.
	_, result := newFullBackupForm(nil, nil, nil, &backupFormResult{
		backupType:  "incremental",
		compression: "default",
		configName:  defaultConfigName,
	})
	assert.True(t, result.incrBase, "incrBase should be forced true when no chain exists")
}

func TestFullBackupFormIncrementalWithChain(t *testing.T) {
	// When a chain exists, the form should preserve the user's incrBase choice.
	backups := []sdk.Backup{{
		Name:       "2026-01-01T00:00:00Z",
		Type:       sdk.BackupTypeIncremental,
		Status:     sdk.StatusDone,
		ConfigName: sdk.MainConfig,
	}}
	_, result := newFullBackupForm(nil, nil, backups, &backupFormResult{
		backupType:  "incremental",
		compression: "default",
		configName:  defaultConfigName,
		incrBase:    false,
	})
	assert.False(t, result.incrBase, "incrBase should remain false when chain exists")
}

// --- formOverlayInnerWidth ---

func TestFormOverlayInnerWidth(t *testing.T) {
	tests := []struct {
		name     string
		termW    int
		wantMin  int
		wantMax  int
		wantExac int // -1 if we just want range check
	}{
		{"very narrow terminal", 40, formOverlayMinWidth, formOverlayMinWidth, formOverlayMinWidth},
		{"very wide terminal", 200, formOverlayMaxWidth, formOverlayMaxWidth, formOverlayMaxWidth},
		{"typical terminal 100", 100, formOverlayMinWidth, formOverlayMaxWidth, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formOverlayInnerWidth(tt.termW)
			if tt.wantExac >= 0 {
				assert.Equal(t, tt.wantExac, got)
			} else {
				assert.GreaterOrEqual(t, got, tt.wantMin)
				assert.LessOrEqual(t, got, tt.wantMax)
			}
		})
	}
}

func intPtr(n int) *int {
	return &n
}
