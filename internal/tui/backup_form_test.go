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

func TestBackupFormResult_ToCommand_Logical(t *testing.T) {
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

func TestBackupFormResult_ToCommand_Incremental(t *testing.T) {
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
