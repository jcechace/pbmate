package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// --- toSnapshotCommand ---

func TestToSnapshotCommand(t *testing.T) {
	t.Run("full scope omits namespaces and usersAndRoles", func(t *testing.T) {
		r := restoreFormResult{
			scope:         restoreScopeFull,
			namespaces:    "db.col1, db.col2", // stale value from previous scope switch
			usersAndRoles: true,               // stale
			parallelColls: "4",
		}
		cmd := r.toSnapshotCommand("backup-2026")
		assert.Equal(t, "backup-2026", cmd.BackupName)
		assert.Nil(t, cmd.Namespaces)
		assert.False(t, cmd.UsersAndRoles)
		assert.NotNil(t, cmd.NumParallelColls)
		assert.Equal(t, 4, *cmd.NumParallelColls)
	})

	t.Run("selective scope includes namespaces and usersAndRoles", func(t *testing.T) {
		r := restoreFormResult{
			scope:         restoreScopeSelective,
			namespaces:    "db.col1, db.col2",
			usersAndRoles: true,
		}
		cmd := r.toSnapshotCommand("backup-2026")
		assert.Equal(t, []string{"db.col1", "db.col2"}, cmd.Namespaces)
		assert.True(t, cmd.UsersAndRoles)
	})

	t.Run("optional int fields are nil when empty", func(t *testing.T) {
		r := restoreFormResult{
			scope:            restoreScopeFull,
			parallelColls:    "",
			insertionWorkers: "",
		}
		cmd := r.toSnapshotCommand("bk")
		assert.Nil(t, cmd.NumParallelColls)
		assert.Nil(t, cmd.NumInsertionWorkers)
	})
}

// --- toPITRCommand ---

func TestToPITRCommand(t *testing.T) {
	// 2026-02-20T14:30:00 UTC as time.Time and expected unix seconds.
	targetTime := time.Date(2026, 2, 20, 14, 30, 0, 0, time.UTC)
	const targetUnix = uint32(1771597800)

	t.Run("preset target uses pitrPreset", func(t *testing.T) {
		r := restoreFormResult{
			scope:            restoreScopeFull,
			pitrPreset:       "2026-02-20T14:30:00",
			insertionWorkers: "8",
		}
		cmd := r.toPITRCommand("base-backup")
		assert.Equal(t, "base-backup", cmd.BackupName)
		assert.Equal(t, targetUnix, cmd.Target.T)
		assert.Nil(t, cmd.Namespaces)
		assert.NotNil(t, cmd.NumInsertionWorkers)
		assert.Equal(t, 8, *cmd.NumInsertionWorkers)
	})

	t.Run("selective scope includes namespaces", func(t *testing.T) {
		r := restoreFormResult{
			scope:      restoreScopeSelective,
			pitrPreset: "2026-02-20T14:30:00",
			namespaces: "mydb.mycol",
		}
		cmd := r.toPITRCommand("base")
		assert.Equal(t, []string{"mydb.mycol"}, cmd.Namespaces)
	})

	t.Run("full scope prevents namespace leak", func(t *testing.T) {
		r := restoreFormResult{
			scope:      restoreScopeFull,
			pitrPreset: "2026-02-20T14:30:00",
			namespaces: "stale.value", // stale from scope switch
		}
		cmd := r.toPITRCommand("base")
		assert.Nil(t, cmd.Namespaces)
		assert.False(t, cmd.UsersAndRoles)
	})

	t.Run("custom preset uses pitrTarget time.Time", func(t *testing.T) {
		r := restoreFormResult{
			scope:      restoreScopeFull,
			pitrPreset: pitrPresetCustom,
			pitrTarget: targetTime,
		}
		cmd := r.toPITRCommand("base")
		assert.Equal(t, targetUnix, cmd.Target.T)
	})
}

// --- parseNamespaces (via restoreFormResult) ---

func TestParseNamespaces(t *testing.T) {
	tests := []struct {
		name       string
		scope      string
		namespaces string
		want       []string
	}{
		{
			name:       "full scope returns nil",
			scope:      restoreScopeFull,
			namespaces: "db.col1, db.col2",
			want:       nil,
		},
		{
			name:       "selective with valid namespaces",
			scope:      restoreScopeSelective,
			namespaces: "db.col1, db.col2",
			want:       []string{"db.col1", "db.col2"},
		},
		{
			name:       "selective with single namespace",
			scope:      restoreScopeSelective,
			namespaces: "mydb.mycol",
			want:       []string{"mydb.mycol"},
		},
		{
			name:       "selective with empty string returns nil",
			scope:      restoreScopeSelective,
			namespaces: "",
			want:       nil,
		},
		{
			name:       "selective with whitespace only returns nil",
			scope:      restoreScopeSelective,
			namespaces: "  ,  ,  ",
			want:       nil,
		},
		{
			name:       "selective with trailing comma filters empty",
			scope:      restoreScopeSelective,
			namespaces: "db.col1, db.col2,",
			want:       []string{"db.col1", "db.col2"},
		},
		{
			name:       "selective trims whitespace",
			scope:      restoreScopeSelective,
			namespaces: " db.col1 , db.col2 ",
			want:       []string{"db.col1", "db.col2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &restoreFormResult{scope: tt.scope, namespaces: tt.namespaces}
			got := r.parseNamespaces()
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- backupContextDescription ---

func TestBackupContextDescription(t *testing.T) {
	t.Run("logical backup with size and config", func(t *testing.T) {
		cn, _ := sdk.NewConfigName("s3-west")
		bk := &sdk.Backup{
			Name:       "2026-02-20T14:30:00Z",
			Type:       sdk.BackupTypeLogical,
			Status:     sdk.StatusDone,
			Size:       1024 * 1024 * 50, // 50 MiB
			ConfigName: cn,
		}
		desc := backupContextDescription(bk)
		assert.Contains(t, desc, "logical")
		assert.Contains(t, desc, "done")
		assert.Contains(t, desc, "s3-west")
	})

	t.Run("incremental backup shows chain parent", func(t *testing.T) {
		bk := &sdk.Backup{
			Name:       "2026-02-20T15:00:00Z",
			Type:       sdk.BackupTypeIncremental,
			Status:     sdk.StatusDone,
			SrcBackup:  "2026-02-20T14:00:00Z",
			ConfigName: sdk.MainConfig,
		}
		desc := backupContextDescription(bk)
		assert.Contains(t, desc, "incremental")
		assert.Contains(t, desc, "Chain parent: 2026-02-20T14:00:00Z")
	})

	t.Run("backup with zero size omits size", func(t *testing.T) {
		bk := &sdk.Backup{
			Name:       "2026-02-20T14:30:00Z",
			Type:       sdk.BackupTypeLogical,
			Status:     sdk.StatusRunning,
			ConfigName: sdk.MainConfig,
		}
		desc := backupContextDescription(bk)
		assert.Contains(t, desc, "logical")
		assert.Contains(t, desc, "running")
		// Should not contain byte formatting artifacts.
		assert.NotContains(t, desc, "0 B")
	})
}

// --- physicalRestoreWarning ---

func TestPhysicalRestoreWarning(t *testing.T) {
	t.Run("snapshot restore", func(t *testing.T) {
		req := physicalRestoreConfirmRequest{
			backupName: "2026-02-20T14:30:00Z",
			backupType: "physical",
			isPITR:     false,
		}
		warning := physicalRestoreWarning(req)
		assert.Contains(t, warning, "physical restore")
		assert.Contains(t, warning, "shut down mongod")
		assert.Contains(t, warning, "2026-02-20T14:30:00Z")
		assert.NotContains(t, warning, "base backup")
	})

	t.Run("PITR restore with physical base", func(t *testing.T) {
		req := physicalRestoreConfirmRequest{
			backupName: "2026-02-20T14:00:00Z",
			backupType: "incremental",
			isPITR:     true,
		}
		warning := physicalRestoreWarning(req)
		assert.Contains(t, warning, "base backup")
		assert.Contains(t, warning, "incremental")
		assert.Contains(t, warning, "shut down mongod")
		assert.Contains(t, warning, "2026-02-20T14:00:00Z")
	})
}

// --- findBackupByName ---

func TestFindBackupByName(t *testing.T) {
	backups := []sdk.Backup{
		{Name: "bk1"},
		{Name: "bk2"},
		{Name: "bk3"},
	}

	t.Run("found", func(t *testing.T) {
		got := findBackupByName(backups, "bk2")
		require.NotNil(t, got)
		assert.Equal(t, "bk2", got.Name)
	})

	t.Run("not found", func(t *testing.T) {
		got := findBackupByName(backups, "nonexistent")
		assert.Nil(t, got)
	})

	t.Run("empty slice", func(t *testing.T) {
		got := findBackupByName(nil, "bk1")
		assert.Nil(t, got)
	})
}
