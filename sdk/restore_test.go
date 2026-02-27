package sdk

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnwaitableRestoreResult(t *testing.T) {
	r := &unwaitableRestoreResult{name: "test-restore", opid: "abc123"}

	assert.Equal(t, "test-restore", r.Name())
	assert.Equal(t, "abc123", r.OPID())
	assert.False(t, r.Waitable())

	restore, err := r.Wait(context.Background(), RestoreWaitOptions{})
	require.ErrorIs(t, err, ErrRestoreUnwaitable)
	assert.Nil(t, restore)
}

func TestWaitableRestoreResult(t *testing.T) {
	r := &waitableRestoreResult{name: "test-restore", opid: "abc123"}

	assert.Equal(t, "test-restore", r.Name())
	assert.Equal(t, "abc123", r.OPID())
	assert.True(t, r.Waitable())
}

func TestRestoreCommandBackupName(t *testing.T) {
	tests := []struct {
		name string
		cmd  StartRestoreCommand
		want string
	}{
		{
			name: "snapshot restore",
			cmd:  StartSnapshotRestore{BackupName: "2026-02-19T20:28:16Z"},
			want: "2026-02-19T20:28:16Z",
		},
		{
			name: "PITR restore",
			cmd:  StartPITRRestore{BackupName: "2026-02-19T20:28:16Z", Target: Timestamp{T: 1}},
			want: "2026-02-19T20:28:16Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, restoreCommandBackupName(tt.cmd))
		})
	}
}

func TestRestoreInProgress(t *testing.T) {
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
			r := Restore{Status: tt.status}
			assert.Equal(t, tt.want, r.InProgress())
		})
	}
}

func TestRestoreDuration(t *testing.T) {
	start := time.Date(2026, 2, 19, 20, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		restore Restore
		want    time.Duration
	}{
		{
			name: "completed restore",
			restore: Restore{
				Status:           StatusDone,
				StartTS:          start,
				LastTransitionTS: start.Add(10 * time.Minute),
			},
			want: 10 * time.Minute,
		},
		{
			name: "failed restore",
			restore: Restore{
				Status:           StatusError,
				StartTS:          start,
				LastTransitionTS: start.Add(3 * time.Minute),
			},
			want: 3 * time.Minute,
		},
		{
			name: "still running (non-terminal)",
			restore: Restore{
				Status:           StatusRunning,
				StartTS:          start,
				LastTransitionTS: start.Add(7 * time.Minute),
			},
			want: 0,
		},
		{
			name: "zero start time",
			restore: Restore{
				Status:           StatusDone,
				LastTransitionTS: start.Add(5 * time.Minute),
			},
			want: 0,
		},
		{
			name: "zero transition time",
			restore: Restore{
				Status:  StatusDone,
				StartTS: start,
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.restore.Duration())
		})
	}
}

func TestRestoreElapsed(t *testing.T) {
	start := time.Date(2026, 2, 19, 20, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		restore Restore
		check   func(t *testing.T, d time.Duration)
	}{
		{
			name:    "zero start returns zero",
			restore: Restore{Status: StatusRunning},
			check: func(t *testing.T, d time.Duration) {
				assert.Equal(t, time.Duration(0), d)
			},
		},
		{
			name: "terminal status returns final duration",
			restore: Restore{
				Status:           StatusDone,
				StartTS:          start,
				LastTransitionTS: start.Add(10 * time.Minute),
			},
			check: func(t *testing.T, d time.Duration) {
				assert.Equal(t, 10*time.Minute, d)
			},
		},
		{
			name: "in-progress returns positive live elapsed",
			restore: Restore{
				Status:  StatusRunning,
				StartTS: time.Now().Add(-3 * time.Second),
			},
			check: func(t *testing.T, d time.Duration) {
				assert.Greater(t, d, 2*time.Second)
				assert.Less(t, d, 10*time.Second)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, tt.restore.Elapsed())
		})
	}
}
