package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// --- parsePITRTarget ---

func TestParsePITRTarget(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantT   uint32
		wantErr bool
	}{
		{
			name:  "ISO format",
			input: "2026-02-20T14:30:00",
			wantT: 1771597800,
		},
		{
			name:  "space-separated format",
			input: "2026-02-20 14:30:00",
			wantT: 1771597800,
		},
		{
			name:  "leading/trailing whitespace",
			input: "  2026-02-20T14:30:00  ",
			wantT: 1771597800,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   "not-a-date",
			wantErr: true,
		},
		{
			name:    "date without time",
			input:   "2026-02-20",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, err := parsePITRTarget(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantT, ts.T)
			assert.Equal(t, uint32(0), ts.I)
		})
	}
}

// --- findBaseBackup ---

func TestFindBaseBackup(t *testing.T) {
	mkBackup := func(name string, status sdk.Status, lastWriteT uint32) sdk.Backup {
		return sdk.Backup{
			Name:        name,
			Status:      status,
			LastWriteTS: sdk.Timestamp{T: lastWriteT},
		}
	}

	backups := []sdk.Backup{
		mkBackup("bk-old", sdk.StatusDone, 1000),
		mkBackup("bk-mid", sdk.StatusDone, 2000),
		mkBackup("bk-new", sdk.StatusDone, 3000),
		mkBackup("bk-future", sdk.StatusDone, 5000),
		mkBackup("bk-err", sdk.StatusError, 1500), // non-done
	}

	tests := []struct {
		name     string
		targetT  uint32
		backups  []sdk.Backup
		wantName string
		wantErr  bool
	}{
		{
			name:     "selects latest before target",
			targetT:  2500,
			backups:  backups,
			wantName: "bk-mid",
		},
		{
			name:     "exact match on lastWriteTS",
			targetT:  3000,
			backups:  backups,
			wantName: "bk-new",
		},
		{
			name:    "target before all backups",
			targetT: 500,
			backups: backups,
			wantErr: true,
		},
		{
			name:    "empty backup list",
			targetT: 3000,
			backups: nil,
			wantErr: true,
		},
		{
			name:    "skips non-done backups",
			targetT: 1800,
			backups: []sdk.Backup{
				mkBackup("bk-err", sdk.StatusError, 1500),
				mkBackup("bk-running", sdk.StatusRunning, 1200),
			},
			wantErr: true,
		},
		{
			name:    "skips zero lastWriteTS",
			targetT: 3000,
			backups: []sdk.Backup{
				{Name: "bk-zero", Status: sdk.StatusDone, LastWriteTS: sdk.Timestamp{}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := sdk.Timestamp{T: tt.targetT}
			name, err := findBaseBackup(target, tt.backups)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, name)
		})
	}
}

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
	t.Run("valid PITR command", func(t *testing.T) {
		r := restoreFormResult{
			scope:            restoreScopeFull,
			pitrPreset:       "2026-02-20T14:30:00",
			pitrTarget:       "2026-02-20T14:30:00",
			insertionWorkers: "8",
		}
		cmd, err := r.toPITRCommand("base-backup")
		require.NoError(t, err)
		assert.Equal(t, "base-backup", cmd.BackupName)
		assert.Equal(t, uint32(1771597800), cmd.Target.T)
		assert.Nil(t, cmd.Namespaces)
		assert.NotNil(t, cmd.NumInsertionWorkers)
		assert.Equal(t, 8, *cmd.NumInsertionWorkers)
	})

	t.Run("selective scope includes namespaces", func(t *testing.T) {
		r := restoreFormResult{
			scope:      restoreScopeSelective,
			pitrPreset: "2026-02-20T14:30:00",
			pitrTarget: "2026-02-20T14:30:00",
			namespaces: "mydb.mycol",
		}
		cmd, err := r.toPITRCommand("base")
		require.NoError(t, err)
		assert.Equal(t, []string{"mydb.mycol"}, cmd.Namespaces)
	})

	t.Run("full scope prevents namespace leak", func(t *testing.T) {
		r := restoreFormResult{
			scope:      restoreScopeFull,
			pitrPreset: "2026-02-20T14:30:00",
			pitrTarget: "2026-02-20T14:30:00",
			namespaces: "stale.value", // stale from scope switch
		}
		cmd, err := r.toPITRCommand("base")
		require.NoError(t, err)
		assert.Nil(t, cmd.Namespaces)
		assert.False(t, cmd.UsersAndRoles)
	})

	t.Run("custom preset uses pitrTarget", func(t *testing.T) {
		r := restoreFormResult{
			scope:      restoreScopeFull,
			pitrPreset: pitrPresetCustom,
			pitrTarget: "2026-02-20T14:30:00",
		}
		cmd, err := r.toPITRCommand("base")
		require.NoError(t, err)
		assert.Equal(t, uint32(1771597800), cmd.Target.T)
	})

	t.Run("invalid target returns error", func(t *testing.T) {
		r := restoreFormResult{
			scope:      restoreScopeFull,
			pitrPreset: pitrPresetCustom,
			pitrTarget: "not-a-date",
		}
		_, err := r.toPITRCommand("base")
		assert.Error(t, err)
	})
}

// --- latestTimeline ---

func TestLatestTimeline(t *testing.T) {
	t.Run("empty returns nil", func(t *testing.T) {
		assert.Nil(t, latestTimeline(nil))
	})

	t.Run("single timeline", func(t *testing.T) {
		tl := []sdk.Timeline{{End: sdk.Timestamp{T: 100}}}
		got := latestTimeline(tl)
		require.NotNil(t, got)
		assert.Equal(t, uint32(100), got.End.T)
	})

	t.Run("multiple timelines returns latest end", func(t *testing.T) {
		tls := []sdk.Timeline{
			{End: sdk.Timestamp{T: 100}},
			{End: sdk.Timestamp{T: 300}},
			{End: sdk.Timestamp{T: 200}},
		}
		got := latestTimeline(tls)
		require.NotNil(t, got)
		assert.Equal(t, uint32(300), got.End.T)
	})
}

// --- completedBackupProfiles ---

func TestCompletedBackupProfiles(t *testing.T) {
	cn := func(name string) sdk.ConfigName {
		if name == "main" {
			return sdk.MainConfig
		}
		c, _ := sdk.NewConfigName(name)
		return c
	}

	mkBackup := func(name string, status sdk.Status, profile sdk.ConfigName) sdk.Backup {
		return sdk.Backup{Name: name, Status: status, ConfigName: profile}
	}

	t.Run("empty backups", func(t *testing.T) {
		opts := completedBackupProfiles(nil)
		assert.Empty(t, opts)
	})

	t.Run("only non-done backups", func(t *testing.T) {
		backups := []sdk.Backup{
			mkBackup("bk1", sdk.StatusError, cn("main")),
		}
		opts := completedBackupProfiles(backups)
		assert.Empty(t, opts)
	})

	t.Run("main only", func(t *testing.T) {
		backups := []sdk.Backup{
			mkBackup("bk1", sdk.StatusDone, cn("main")),
			mkBackup("bk2", sdk.StatusDone, cn("main")),
		}
		opts := completedBackupProfiles(backups)
		require.Len(t, opts, 1)
		assert.Equal(t, "main", opts[0].Value)
		assert.Equal(t, "Main", opts[0].Key)
	})

	t.Run("main first then named profiles", func(t *testing.T) {
		backups := []sdk.Backup{
			mkBackup("bk1", sdk.StatusDone, cn("s3-west")),
			mkBackup("bk2", sdk.StatusDone, cn("main")),
			mkBackup("bk3", sdk.StatusDone, cn("gcs-east")),
			mkBackup("bk4", sdk.StatusDone, cn("s3-west")), // duplicate
		}
		opts := completedBackupProfiles(backups)
		require.Len(t, opts, 3)
		assert.Equal(t, "main", opts[0].Value)
		assert.Equal(t, "s3-west", opts[1].Value)
		assert.Equal(t, "gcs-east", opts[2].Value)
	})
}

// --- completedBackupOptions ---

func TestCompletedBackupOptions(t *testing.T) {
	cn := func(name string) sdk.ConfigName {
		if name == "main" {
			return sdk.MainConfig
		}
		c, _ := sdk.NewConfigName(name)
		return c
	}

	backups := []sdk.Backup{
		{Name: "bk-main-1", Status: sdk.StatusDone, ConfigName: cn("main")},
		{Name: "bk-main-2", Status: sdk.StatusDone, ConfigName: cn("main")},
		{Name: "bk-s3", Status: sdk.StatusDone, ConfigName: cn("s3")},
		{Name: "bk-err", Status: sdk.StatusError, ConfigName: cn("main")},
	}

	t.Run("filters by profile", func(t *testing.T) {
		opts := completedBackupOptions(backups, "main")
		require.Len(t, opts, 2)
		assert.Equal(t, "bk-main-1", opts[0].Value)
		assert.Equal(t, "bk-main-2", opts[1].Value)
	})

	t.Run("different profile", func(t *testing.T) {
		opts := completedBackupOptions(backups, "s3")
		require.Len(t, opts, 1)
		assert.Equal(t, "bk-s3", opts[0].Value)
	})

	t.Run("no matches", func(t *testing.T) {
		opts := completedBackupOptions(backups, "nonexistent")
		assert.Empty(t, opts)
	})

	t.Run("skips non-done", func(t *testing.T) {
		opts := completedBackupOptions(backups, "main")
		for _, o := range opts {
			assert.NotEqual(t, "bk-err", o.Value)
		}
	})
}

// --- hasOptionValue ---

func TestHasOptionValue(t *testing.T) {
	opts := completedBackupOptions([]sdk.Backup{
		{Name: "bk1", Status: sdk.StatusDone, ConfigName: sdk.MainConfig},
		{Name: "bk2", Status: sdk.StatusDone, ConfigName: sdk.MainConfig},
	}, "main")

	assert.True(t, hasOptionValue(opts, "bk1"))
	assert.True(t, hasOptionValue(opts, "bk2"))
	assert.False(t, hasOptionValue(opts, "bk3"))
	assert.False(t, hasOptionValue(nil, "bk1"))
}
