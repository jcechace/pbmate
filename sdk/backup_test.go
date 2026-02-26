package sdk

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBackupIsLogical(t *testing.T) {
	tests := []struct {
		name   string
		backup Backup
		want   bool
	}{
		{name: "logical backup", backup: Backup{Type: BackupTypeLogical}, want: true},
		{name: "physical backup", backup: Backup{Type: BackupTypePhysical}, want: false},
		{name: "incremental backup", backup: Backup{Type: BackupTypeIncremental}, want: false},
		{name: "zero type", backup: Backup{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.backup.IsLogical())
		})
	}
}

func TestBackupIsPhysical(t *testing.T) {
	tests := []struct {
		name   string
		backup Backup
		want   bool
	}{
		{name: "physical backup", backup: Backup{Type: BackupTypePhysical}, want: true},
		{name: "logical backup", backup: Backup{Type: BackupTypeLogical}, want: false},
		{name: "incremental backup", backup: Backup{Type: BackupTypeIncremental}, want: false},
		{name: "zero type", backup: Backup{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.backup.IsPhysical())
		})
	}
}

func TestBackupIsIncremental(t *testing.T) {
	tests := []struct {
		name   string
		backup Backup
		want   bool
	}{
		{
			name:   "incremental backup",
			backup: Backup{Type: BackupTypeIncremental},
			want:   true,
		},
		{
			name:   "logical backup",
			backup: Backup{Type: BackupTypeLogical},
			want:   false,
		},
		{
			name:   "physical backup",
			backup: Backup{Type: BackupTypePhysical},
			want:   false,
		},
		{
			name:   "zero type",
			backup: Backup{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.backup.IsIncremental())
		})
	}
}

func TestBackupIsIncrementalBase(t *testing.T) {
	tests := []struct {
		name   string
		backup Backup
		want   bool
	}{
		{
			name:   "incremental base (no parent)",
			backup: Backup{Type: BackupTypeIncremental, SrcBackup: ""},
			want:   true,
		},
		{
			name:   "incremental child (has parent)",
			backup: Backup{Type: BackupTypeIncremental, SrcBackup: "2026-02-19T20:00:00Z"},
			want:   false,
		},
		{
			name:   "logical backup (not incremental)",
			backup: Backup{Type: BackupTypeLogical, SrcBackup: ""},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.backup.IsIncrementalBase())
		})
	}
}

func TestBackupIsSelective(t *testing.T) {
	tests := []struct {
		name   string
		backup Backup
		want   bool
	}{
		{
			name:   "full backup (nil namespaces)",
			backup: Backup{},
			want:   false,
		},
		{
			name:   "full backup (empty namespaces)",
			backup: Backup{Namespaces: []string{}},
			want:   false,
		},
		{
			name:   "selective backup",
			backup: Backup{Namespaces: []string{"mydb.mycoll"}},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.backup.IsSelective())
		})
	}
}

func TestBackupInProgress(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{name: "running", status: StatusRunning, want: true},
		{name: "starting", status: StatusStarting, want: true},
		{name: "done", status: StatusDone, want: false},
		{name: "error", status: StatusError, want: false},
		{name: "cancelled", status: StatusCancelled, want: false},
		{name: "partly done", status: StatusPartlyDone, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bk := Backup{Status: tt.status}
			assert.Equal(t, tt.want, bk.InProgress())
		})
	}
}

func TestBackupDuration(t *testing.T) {
	start := time.Date(2026, 2, 19, 20, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		backup Backup
		want   time.Duration
	}{
		{
			name: "completed backup",
			backup: Backup{
				Status:           StatusDone,
				StartTS:          start,
				LastTransitionTS: start.Add(5 * time.Minute),
			},
			want: 5 * time.Minute,
		},
		{
			name: "failed backup",
			backup: Backup{
				Status:           StatusError,
				StartTS:          start,
				LastTransitionTS: start.Add(2 * time.Minute),
			},
			want: 2 * time.Minute,
		},
		{
			name: "still running (non-terminal)",
			backup: Backup{
				Status:           StatusRunning,
				StartTS:          start,
				LastTransitionTS: start.Add(3 * time.Minute),
			},
			want: 0,
		},
		{
			name: "zero start time",
			backup: Backup{
				Status:           StatusDone,
				LastTransitionTS: start.Add(5 * time.Minute),
			},
			want: 0,
		},
		{
			name: "zero transition time",
			backup: Backup{
				Status:  StatusDone,
				StartTS: start,
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.backup.Duration())
		})
	}
}
