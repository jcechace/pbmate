package sdk

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
