package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// --- pitrBaseOptions ---

func TestPitrBaseOptions(t *testing.T) {
	timelines := []sdk.Timeline{
		{Start: sdk.Timestamp{T: 1000}, End: sdk.Timestamp{T: 5000}},
	}

	mkBackup := func(name string, lastWriteT uint32) sdk.Backup {
		return sdk.Backup{
			Name:        name,
			Status:      sdk.StatusDone,
			Type:        sdk.BackupTypeLogical,
			ConfigName:  sdk.MainConfig,
			LastWriteTS: sdk.Timestamp{T: lastWriteT},
		}
	}

	tests := []struct {
		name       string
		target     sdk.Timestamp
		backups    []sdk.Backup
		timelines  []sdk.Timeline
		wantValues []string
	}{
		{
			name:       "no valid bases returns nil",
			target:     sdk.Timestamp{T: 946684810},
			backups:    nil,
			timelines:  timelines,
			wantValues: nil,
		},
		{
			name:   "returns options sorted by LastWriteTS desc",
			target: sdk.Timestamp{T: 946687800},
			backups: []sdk.Backup{
				mkBackup("bk-early", 1100),
				mkBackup("bk-late", 1300),
				mkBackup("bk-mid", 1200),
			},
			timelines: []sdk.Timeline{
				{Start: sdk.Timestamp{T: 1000}, End: sdk.Timestamp{T: 946687800}},
			},
			wantValues: []string{"bk-late", "bk-mid", "bk-early"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := pitrBaseOptions(tt.target, tt.backups, tt.timelines)
			if tt.wantValues == nil {
				assert.Nil(t, opts)
				return
			}
			require.Len(t, opts, len(tt.wantValues))
			for i, wantVal := range tt.wantValues {
				assert.Equal(t, wantVal, opts[i].Value, "opts[%d]", i)
			}
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

// --- resolvePITRTarget ---

func TestResolvePITRTarget(t *testing.T) {
	// 2026-02-20T14:30:00 UTC
	const expectedUnix = uint32(1771597800)
	customTime := time.Date(2026, 2, 20, 14, 30, 0, 0, time.UTC)

	tests := []struct {
		name         string
		preset       string
		customTarget time.Time
		wantT        uint32
	}{
		{
			name:         "non-custom preset parses timestamp string",
			preset:       "2026-02-20T14:30:00",
			customTarget: time.Time{}, // ignored
			wantT:        expectedUnix,
		},
		{
			name:         "custom preset uses time.Time value",
			preset:       pitrPresetCustom,
			customTarget: customTime,
			wantT:        expectedUnix,
		},
		{
			name:         "empty preset returns zero Timestamp",
			preset:       "",
			customTarget: customTime,
			wantT:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolvePITRTarget(tt.preset, tt.customTarget)
			assert.Equal(t, tt.wantT, got.T)
		})
	}
}

// --- parseNamespaces ---

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

// --- pitrPresetOptions ---

func TestPitrPresetOptions(t *testing.T) {
	t.Run("short timeline only has latest and custom", func(t *testing.T) {
		// 3-minute timeline — no relative presets fit.
		tl := &sdk.Timeline{
			Start: sdk.Timestamp{T: 1000},
			End:   sdk.Timestamp{T: 1180}, // 3 minutes
		}
		opts := pitrPresetOptions(tl)
		require.Len(t, opts, 2) // Latest + Custom
		assert.Contains(t, opts[0].Key, "Latest")
		assert.Equal(t, pitrPresetCustom, opts[len(opts)-1].Value)
	})

	t.Run("long timeline includes relative presets", func(t *testing.T) {
		// 2-hour timeline — should include -5m, -15m, -30m, -1h.
		start := uint32(1771590000)
		end := start + 7200 // 2 hours
		tl := &sdk.Timeline{
			Start: sdk.Timestamp{T: start},
			End:   sdk.Timestamp{T: end},
		}
		opts := pitrPresetOptions(tl)
		// Latest + -5m + -15m + -30m + -1h + Custom = 6
		require.Len(t, opts, 6)
		assert.Contains(t, opts[0].Key, "Latest")
		assert.Contains(t, opts[1].Key, "-5 min")
		assert.Contains(t, opts[2].Key, "-15 min")
		assert.Contains(t, opts[3].Key, "-30 min")
		assert.Contains(t, opts[4].Key, "-1 hour")
		assert.Equal(t, pitrPresetCustom, opts[5].Value)
	})

	t.Run("custom is always last", func(t *testing.T) {
		tl := &sdk.Timeline{
			Start: sdk.Timestamp{T: 1000},
			End:   sdk.Timestamp{T: 100000},
		}
		opts := pitrPresetOptions(tl)
		require.True(t, len(opts) >= 2)
		assert.Equal(t, pitrPresetCustom, opts[len(opts)-1].Value)
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
