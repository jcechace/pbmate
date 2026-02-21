package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/jcechace/pbmate/sdk/v2"
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

func TestStatusIndicator(t *testing.T) {
	styles := testStyles()

	tests := []struct {
		name   string
		status sdk.Status
	}{
		{"done", sdk.StatusDone},
		{"error", sdk.StatusError},
		{"partly done", sdk.StatusPartlyDone},
		{"cancelled", sdk.StatusCancelled},
		{"running", sdk.StatusRunning},
		{"starting", sdk.StatusStarting},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := statusIndicator(tt.status, &styles)
			assert.NotEmpty(t, result)
			assert.Contains(t, result, "●")
		})
	}

	// Verify that all branches produce non-empty, dot-containing output.
	// (Color differentiation is visual and not testable without a real terminal.)
}

func TestAgentIndicator(t *testing.T) {
	styles := testStyles()

	tests := []struct {
		name    string
		agent   sdk.Agent
		wantDot string // "●" for filled, "○" for stale
	}{
		{"healthy", sdk.Agent{OK: true}, "●"},
		{"stale", sdk.Agent{Stale: true, OK: true}, "○"},
		{"errors", sdk.Agent{OK: false, Errors: []string{"err"}}, "●"},
		{"not ok", sdk.Agent{OK: false}, "●"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agentIndicator(&tt.agent, &styles)
			assert.Contains(t, result, tt.wantDot)
		})
	}

	// Healthy uses filled dot, stale uses hollow dot — structurally different.
	healthy := agentIndicator(&sdk.Agent{OK: true}, &styles)
	stale := agentIndicator(&sdk.Agent{Stale: true, OK: true}, &styles)

	assert.NotEqual(t, healthy, stale, "healthy (●) vs stale (○) should differ")
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

func TestHelpSections(t *testing.T) {
	sections := helpSections()

	// Verify we have the expected section titles.
	require.True(t, len(sections) >= 5, "expected at least 5 help sections")

	titles := make([]string, len(sections))
	for i, s := range sections {
		titles[i] = s.title
	}

	assert.Contains(t, titles, "Navigation")
	assert.Contains(t, titles, "Actions")
	assert.Contains(t, titles, "Overview")
	assert.Contains(t, titles, "General")

	// Every section should have at least one entry.
	for _, s := range sections {
		assert.NotEmpty(t, s.entries, "section %q should have entries", s.title)
		for _, e := range s.entries {
			assert.NotEmpty(t, e.key, "entry in %q should have a key", s.title)
			assert.NotEmpty(t, e.desc, "entry in %q should have a desc", s.title)
		}
	}
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
