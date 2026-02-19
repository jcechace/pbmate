package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHumanBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0B"},
		{1, "1B"},
		{512, "512B"},
		{1023, "1023B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.0MB"},
		{1572864, "1.5MB"},
		{1073741824, "1.0GB"},
		{1610612736, "1.5GB"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, humanBytes(tt.input), "humanBytes(%d)", tt.input)
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{"just now", now.Add(-30 * time.Second), "just now"},
		{"1 minute ago", now.Add(-90 * time.Second), "1m ago"},
		{"5 minutes ago", now.Add(-5 * time.Minute), "5m ago"},
		{"1 hour ago", now.Add(-90 * time.Minute), "1h ago"},
		{"3 hours ago", now.Add(-3 * time.Hour), "3h ago"},
		{"1 day ago", now.Add(-36 * time.Hour), "1d ago"},
		{"5 days ago", now.Add(-5 * 24 * time.Hour), "5d ago"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, relativeTime(tt.input), tt.name)
	}
}
