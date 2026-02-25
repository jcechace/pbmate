package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testStyles returns a Styles value suitable for unit tests.
func testStyles() Styles {
	return NewStyles(DefaultTheme())
}

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

func TestRenderCursorList(t *testing.T) {
	styles := testStyles()
	lines := []string{"alpha", "beta", "gamma"}

	t.Run("focused cursor renders arrow", func(t *testing.T) {
		result := renderCursorList(lines, 1, true, &styles)
		assert.Contains(t, result, "▶")
		// The cursor line should contain "beta".
		assert.Contains(t, result, "beta")
	})

	t.Run("unfocused cursor omits arrow", func(t *testing.T) {
		result := renderCursorList(lines, 0, false, &styles)
		assert.NotContains(t, result, "▶")
		assert.Contains(t, result, "alpha")
	})

	t.Run("all lines present", func(t *testing.T) {
		result := renderCursorList(lines, 0, true, &styles)
		for _, line := range lines {
			assert.Contains(t, result, line)
		}
	})

	t.Run("non-cursor lines indented", func(t *testing.T) {
		result := renderCursorList(lines, 0, true, &styles)
		resultLines := strings.Split(result, "\n")
		require.Len(t, resultLines, 3)
		// Lines 1 and 2 (non-cursor) should start with "  " (two-space indent).
		assert.True(t, strings.HasPrefix(resultLines[1], "  "), "non-cursor line should be indented")
		assert.True(t, strings.HasPrefix(resultLines[2], "  "), "non-cursor line should be indented")
	})

	t.Run("empty list", func(t *testing.T) {
		result := renderCursorList(nil, 0, true, &styles)
		assert.Empty(t, result)
	})
}

func TestHelpColumns(t *testing.T) {
	left, right := helpColumns()

	// Verify we have the expected number of sections per column.
	require.Len(t, left, 3, "left column should have 3 sections")
	require.Len(t, right, 3, "right column should have 3 sections")

	// Verify left column section titles.
	assert.Equal(t, "Navigation", left[0].title)
	assert.Equal(t, "Global", left[1].title)
	assert.Equal(t, "General", left[2].title)

	// Verify right column section titles.
	assert.Equal(t, "1:Overview", right[0].title)
	assert.Equal(t, "2:Backups", right[1].title)
	assert.Equal(t, "3:Config", right[2].title)

	// Every section should have at least one entry with non-empty key/desc.
	for _, sections := range [][]helpSection{left, right} {
		for _, s := range sections {
			assert.NotEmpty(t, s.entries, "section %q should have entries", s.title)
			for _, e := range s.entries {
				assert.NotEmpty(t, e.key, "entry in %q should have a key", s.title)
				assert.NotEmpty(t, e.desc, "entry in %q should have a desc", s.title)
			}
		}
	}
}

func TestHelpCombined(t *testing.T) {
	a := key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "backup"))
	b := key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "custom backup"))
	entry := helpCombined(a, b, "backup")
	assert.Equal(t, "s / S", entry.key)
	assert.Equal(t, "backup", entry.desc)
}

func TestHelpFromBinding(t *testing.T) {
	b := key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "do something"),
	)
	entry := helpFromBinding(b)
	assert.Equal(t, "x", entry.key)
	assert.Equal(t, "do something", entry.desc)
}
