package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterPITRBases(t *testing.T) {
	// Shared timeline: covers T=1000 to T=2000.
	timelines := []Timeline{
		{Start: Timestamp{T: 1000}, End: Timestamp{T: 2000}},
	}

	// Helper to build a valid base backup at the given LastWriteTS.
	validBackup := func(name string, lastWriteT uint32) Backup {
		return Backup{
			Name:        name,
			Status:      StatusDone,
			Type:        BackupTypeLogical,
			ConfigName:  MainConfig,
			LastWriteTS: Timestamp{T: lastWriteT},
		}
	}

	tests := []struct {
		name      string
		target    Timestamp
		backups   []Backup
		timelines []Timeline
		wantNames []string
	}{
		{
			name:      "no backups",
			target:    Timestamp{T: 1500},
			backups:   nil,
			timelines: timelines,
			wantNames: nil,
		},
		{
			name:   "single valid backup",
			target: Timestamp{T: 1500},
			backups: []Backup{
				validBackup("bk1", 1200),
			},
			timelines: timelines,
			wantNames: []string{"bk1"},
		},
		{
			name:   "sorted by LastWriteTS descending",
			target: Timestamp{T: 1800},
			backups: []Backup{
				validBackup("bk-early", 1100),
				validBackup("bk-late", 1600),
				validBackup("bk-mid", 1300),
			},
			timelines: timelines,
			wantNames: []string{"bk-late", "bk-mid", "bk-early"},
		},
		{
			name:   "excludes non-done status",
			target: Timestamp{T: 1500},
			backups: []Backup{
				validBackup("done", 1200),
				{
					Name:        "running",
					Status:      StatusRunning,
					Type:        BackupTypeLogical,
					ConfigName:  MainConfig,
					LastWriteTS: Timestamp{T: 1100},
				},
				{
					Name:        "error",
					Status:      StatusError,
					Type:        BackupTypeLogical,
					ConfigName:  MainConfig,
					LastWriteTS: Timestamp{T: 1100},
				},
			},
			timelines: timelines,
			wantNames: []string{"done"},
		},
		{
			name:   "excludes zero LastWriteTS",
			target: Timestamp{T: 1500},
			backups: []Backup{
				validBackup("valid", 1200),
				{
					Name:        "zero-ts",
					Status:      StatusDone,
					Type:        BackupTypeLogical,
					ConfigName:  MainConfig,
					LastWriteTS: Timestamp{},
				},
			},
			timelines: timelines,
			wantNames: []string{"valid"},
		},
		{
			name:   "excludes backup at or after target",
			target: Timestamp{T: 1500},
			backups: []Backup{
				validBackup("before", 1200),
				validBackup("at-target", 1500),
				validBackup("after", 1600),
			},
			timelines: timelines,
			wantNames: []string{"before"},
		},
		{
			name:   "excludes selective backups",
			target: Timestamp{T: 1500},
			backups: []Backup{
				validBackup("full", 1200),
				{
					Name:        "selective",
					Status:      StatusDone,
					Type:        BackupTypeLogical,
					ConfigName:  MainConfig,
					LastWriteTS: Timestamp{T: 1100},
					Namespaces:  []string{"db.col"},
				},
			},
			timelines: timelines,
			wantNames: []string{"full"},
		},
		{
			name:   "excludes external backups",
			target: Timestamp{T: 1500},
			backups: []Backup{
				validBackup("logical", 1200),
				{
					Name:        "external",
					Status:      StatusDone,
					Type:        BackupTypeExternal,
					ConfigName:  MainConfig,
					LastWriteTS: Timestamp{T: 1100},
				},
			},
			timelines: timelines,
			wantNames: []string{"logical"},
		},
		{
			name:   "excludes non-main config",
			target: Timestamp{T: 1500},
			backups: []Backup{
				validBackup("main", 1200),
				{
					Name:        "profile",
					Status:      StatusDone,
					Type:        BackupTypeLogical,
					ConfigName:  ConfigName{value: "archive"},
					LastWriteTS: Timestamp{T: 1100},
				},
			},
			timelines: timelines,
			wantNames: []string{"main"},
		},
		{
			name:   "excludes backup outside timeline coverage",
			target: Timestamp{T: 1500},
			backups: []Backup{
				validBackup("covered", 1200),
				validBackup("outside", 800), // before timeline start
			},
			timelines: timelines,
			wantNames: []string{"covered"},
		},
		{
			name:   "no timelines means no valid bases",
			target: Timestamp{T: 1500},
			backups: []Backup{
				validBackup("bk1", 1200),
			},
			timelines: nil,
			wantNames: nil,
		},
		{
			name:   "target outside all timelines",
			target: Timestamp{T: 3000},
			backups: []Backup{
				validBackup("bk1", 1200),
			},
			timelines: timelines, // ends at 2000
			wantNames: nil,
		},
		{
			name:   "multiple timelines with gap",
			target: Timestamp{T: 3500},
			backups: []Backup{
				validBackup("in-first", 1200),  // in first timeline but target not covered
				validBackup("in-second", 3100), // in second timeline, target covered
			},
			timelines: []Timeline{
				{Start: Timestamp{T: 1000}, End: Timestamp{T: 2000}},
				{Start: Timestamp{T: 3000}, End: Timestamp{T: 4000}},
			},
			wantNames: []string{"in-second"},
		},
		{
			name:   "physical and incremental bases are valid",
			target: Timestamp{T: 1800},
			backups: []Backup{
				validBackup("logical", 1200),
				{
					Name:        "physical",
					Status:      StatusDone,
					Type:        BackupTypePhysical,
					ConfigName:  MainConfig,
					LastWriteTS: Timestamp{T: 1300},
				},
				{
					Name:        "incremental",
					Status:      StatusDone,
					Type:        BackupTypeIncremental,
					ConfigName:  MainConfig,
					LastWriteTS: Timestamp{T: 1400},
				},
			},
			timelines: timelines,
			wantNames: []string{"incremental", "physical", "logical"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterPITRBases(tt.target, tt.backups, tt.timelines)

			if tt.wantNames == nil {
				assert.Nil(t, result)
				return
			}

			require.Len(t, result, len(tt.wantNames))
			for i, wantName := range tt.wantNames {
				assert.Equal(t, wantName, result[i].Name, "result[%d]", i)
			}
		})
	}
}

func TestTimelineCovers(t *testing.T) {
	tests := []struct {
		name      string
		backupTS  Timestamp
		target    Timestamp
		timelines []Timeline
		want      bool
	}{
		{
			name:      "no timelines",
			backupTS:  Timestamp{T: 1200},
			target:    Timestamp{T: 1500},
			timelines: nil,
			want:      false,
		},
		{
			name:     "both inside same timeline",
			backupTS: Timestamp{T: 1200},
			target:   Timestamp{T: 1800},
			timelines: []Timeline{
				{Start: Timestamp{T: 1000}, End: Timestamp{T: 2000}},
			},
			want: true,
		},
		{
			name:     "backup at timeline start",
			backupTS: Timestamp{T: 1000},
			target:   Timestamp{T: 1500},
			timelines: []Timeline{
				{Start: Timestamp{T: 1000}, End: Timestamp{T: 2000}},
			},
			want: true,
		},
		{
			name:     "target at timeline end",
			backupTS: Timestamp{T: 1200},
			target:   Timestamp{T: 2000},
			timelines: []Timeline{
				{Start: Timestamp{T: 1000}, End: Timestamp{T: 2000}},
			},
			want: true,
		},
		{
			name:     "backup before timeline",
			backupTS: Timestamp{T: 500},
			target:   Timestamp{T: 1500},
			timelines: []Timeline{
				{Start: Timestamp{T: 1000}, End: Timestamp{T: 2000}},
			},
			want: false,
		},
		{
			name:     "target after timeline",
			backupTS: Timestamp{T: 1200},
			target:   Timestamp{T: 2500},
			timelines: []Timeline{
				{Start: Timestamp{T: 1000}, End: Timestamp{T: 2000}},
			},
			want: false,
		},
		{
			name:     "in different timelines (gap between)",
			backupTS: Timestamp{T: 1200},
			target:   Timestamp{T: 3500},
			timelines: []Timeline{
				{Start: Timestamp{T: 1000}, End: Timestamp{T: 2000}},
				{Start: Timestamp{T: 3000}, End: Timestamp{T: 4000}},
			},
			want: false,
		},
		{
			name:     "both in second timeline",
			backupTS: Timestamp{T: 3100},
			target:   Timestamp{T: 3500},
			timelines: []Timeline{
				{Start: Timestamp{T: 1000}, End: Timestamp{T: 2000}},
				{Start: Timestamp{T: 3000}, End: Timestamp{T: 4000}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := timelineCovers(tt.backupTS, tt.target, tt.timelines)
			assert.Equal(t, tt.want, got)
		})
	}
}
