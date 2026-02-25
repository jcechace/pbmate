package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

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
