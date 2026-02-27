package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

func TestDeduplicateLogEntries(t *testing.T) {
	makeEntry := func(ts time.Time, msg string) sdk.LogEntry {
		return sdk.LogEntry{Timestamp: ts, Message: msg, Severity: sdk.LogSeverityInfo}
	}

	t0 := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	t1 := t0.Add(1 * time.Second)
	t2 := t0.Add(2 * time.Second)

	tests := []struct {
		name     string
		existing []sdk.LogEntry
		incoming []sdk.LogEntry
		want     []sdk.LogEntry
	}{
		{
			name:     "both empty",
			existing: nil,
			incoming: nil,
			want:     nil,
		},
		{
			name:     "no existing passes all through",
			existing: nil,
			incoming: []sdk.LogEntry{makeEntry(t0, "a"), makeEntry(t1, "b")},
			want:     []sdk.LogEntry{makeEntry(t0, "a"), makeEntry(t1, "b")},
		},
		{
			name:     "no incoming returns nil",
			existing: []sdk.LogEntry{makeEntry(t0, "a")},
			incoming: nil,
			want:     nil,
		},
		{
			name:     "all new past boundary",
			existing: []sdk.LogEntry{makeEntry(t0, "a")},
			incoming: []sdk.LogEntry{makeEntry(t1, "b"), makeEntry(t2, "c")},
			want:     []sdk.LogEntry{makeEntry(t1, "b"), makeEntry(t2, "c")},
		},
		{
			name:     "exact duplicate filtered",
			existing: []sdk.LogEntry{makeEntry(t0, "a")},
			incoming: []sdk.LogEntry{makeEntry(t0, "a")},
			want:     nil,
		},
		{
			name:     "same timestamp different message passes",
			existing: []sdk.LogEntry{makeEntry(t0, "a")},
			incoming: []sdk.LogEntry{makeEntry(t0, "b")},
			want:     []sdk.LogEntry{makeEntry(t0, "b")},
		},
		{
			name:     "mixed dups and new",
			existing: []sdk.LogEntry{makeEntry(t0, "a"), makeEntry(t1, "b")},
			incoming: []sdk.LogEntry{makeEntry(t1, "b"), makeEntry(t2, "c")},
			want:     []sdk.LogEntry{makeEntry(t2, "c")},
		},
		{
			name: "multiple entries at boundary",
			existing: []sdk.LogEntry{
				makeEntry(t0, "early"),
				makeEntry(t1, "x"),
				makeEntry(t1, "y"),
			},
			incoming: []sdk.LogEntry{
				makeEntry(t1, "x"),
				makeEntry(t1, "y"),
				makeEntry(t2, "z"),
			},
			want: []sdk.LogEntry{makeEntry(t2, "z")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deduplicateLogEntries(tt.existing, tt.incoming)
			if tt.want == nil {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestAppendLogEntries(t *testing.T) {
	makeEntry := func(msg string) sdk.LogEntry {
		return sdk.LogEntry{Message: msg, Severity: sdk.LogSeverityInfo}
	}

	newModel := func() overviewModel {
		s := testStyles()
		return overviewModel{
			styles: &s,
			logs:   newLogPanel(&s),
		}
	}

	t.Run("append within capacity", func(t *testing.T) {
		m := newModel()
		entries := []sdk.LogEntry{makeEntry("a"), makeEntry("b"), makeEntry("c")}
		m.appendLogEntries(entries)
		assert.Len(t, m.data.logEntries, 3)
		assert.Equal(t, "a", m.data.logEntries[0].Message)
		assert.Equal(t, "c", m.data.logEntries[2].Message)
	})

	t.Run("append trims to maxLogEntries", func(t *testing.T) {
		m := newModel()

		// Pre-fill with maxLogEntries - 1 entries.
		for i := range maxLogEntries - 1 {
			m.data.logEntries = append(m.data.logEntries, makeEntry("old"))
			_ = i
		}

		// Append 5 more — should trim to maxLogEntries from the tail.
		newEntries := make([]sdk.LogEntry, 5)
		for i := range newEntries {
			newEntries[i] = makeEntry("new")
		}
		m.appendLogEntries(newEntries)

		assert.Len(t, m.data.logEntries, maxLogEntries)
		// The last 5 entries should be the new ones.
		for i := maxLogEntries - 5; i < maxLogEntries; i++ {
			assert.Equal(t, "new", m.data.logEntries[i].Message)
		}
	})

	t.Run("append to empty", func(t *testing.T) {
		m := newModel()
		m.appendLogEntries([]sdk.LogEntry{makeEntry("first")})
		assert.Len(t, m.data.logEntries, 1)
		assert.Equal(t, "first", m.data.logEntries[0].Message)
	})

	t.Run("append nil is no-op", func(t *testing.T) {
		m := newModel()
		m.data.logEntries = []sdk.LogEntry{makeEntry("existing")}
		m.appendLogEntries(nil)
		assert.Len(t, m.data.logEntries, 1)
	})
}
