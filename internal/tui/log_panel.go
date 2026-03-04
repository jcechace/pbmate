package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// logPanel is a reusable scrollable log viewer with follow, pin, and wrap
// support. It wraps a viewport and manages its own content rebuilding.
type logPanel struct {
	vp        viewport.Model
	styles    *Styles
	entries   []sdk.LogEntry
	following bool // follow mode is active (streaming new entries)
	pinned    bool // auto-scroll to bottom on new content
	wrap      bool // word-wrap lines to viewport width
	lineCount int  // total lines in rendered content
}

// newLogPanel creates a log panel with auto-scroll pinned by default.
func newLogPanel(styles *Styles) logPanel {
	return logPanel{
		vp:     newPanelViewport(),
		styles: styles,
		pinned: true,
	}
}

// setEntries replaces the displayed log entries and rebuilds the viewport.
// When the user has scrolled up (not pinned) the update is skipped — there
// is no point refreshing content the user is actively reading. The viewport
// resumes updating as soon as the user scrolls back to the bottom.
func (p *logPanel) setEntries(entries []sdk.LogEntry) {
	if !p.pinned {
		return
	}
	p.entries = entries
	p.rebuildContent()
}

// setFollowing updates the follow mode flag and rebuilds the mode indicator.
// Pinning is set when starting follow so the viewport snaps to the bottom.
// When stopping follow, the scroll position is preserved.
func (p *logPanel) setFollowing(following bool) {
	p.following = following
	if following {
		p.pinned = true
	}
	p.rebuildContent()
}

// scroll moves the log viewport by delta lines and updates the pinned state.
func (p *logPanel) scroll(delta int) {
	scrollViewport(&p.vp, delta)
	p.pinned = p.atBottom()
}

// toggleWrap flips word-wrapping and rebuilds the content.
func (p *logPanel) toggleWrap() {
	p.wrap = !p.wrap
	p.rebuildContent()
}

// view returns the viewport's rendered output (exactly Height lines).
func (p *logPanel) view() string {
	return p.vp.View()
}

// atBottom reports whether the viewport is scrolled to the bottom.
func (p *logPanel) atBottom() bool {
	maxY := p.lineCount - p.vp.Height
	if maxY <= 0 {
		return true // content fits in viewport
	}
	return p.vp.YOffset >= maxY
}

// rebuildContent reconstructs the viewport content from log entries and the
// mode indicator, applying wrap and auto-scroll as configured.
func (p *logPanel) rebuildContent() {
	var b strings.Builder

	if len(p.entries) == 0 {
		b.WriteString(p.styles.StatusMuted.Render(" No log entries"))
	} else {
		for i, entry := range p.entries {
			if i > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(p.formatEntry(entry))
		}
	}

	// Mode indicator as the last line.
	b.WriteByte('\n')
	mode := "auto-refresh"
	if p.following {
		mode = "following"
	}
	if p.wrap {
		mode += "+wrap"
	}
	if p.following {
		b.WriteString(p.styles.StatusWarning.Render(" [" + mode + "]"))
	} else {
		b.WriteString(p.styles.StatusMuted.Render(" [" + mode + "]"))
	}

	content := b.String()

	// When wrapping is enabled, word-wrap lines to viewport width.
	if p.wrap && p.vp.Width > 0 {
		content = lipgloss.NewStyle().Width(p.vp.Width).Render(content)
	}

	p.lineCount = strings.Count(content, "\n") + 1
	p.vp.SetContent(content)

	// Auto-scroll to bottom when pinned.
	if p.vp.Height > 0 && p.pinned {
		p.vp.GotoBottom()
	}
}

// formatEntry formats a single log entry for display.
// Includes RS/node prefix from the entry's structured attributes when available.
func (p *logPanel) formatEntry(entry sdk.LogEntry) string {
	ts := entry.Timestamp.UTC().Format("15:04:05")
	sev := p.severityStyle(entry.Severity).Render(entry.Severity.String()[:1])
	source := formatLogSource(entry.Attrs)
	if source != "" {
		source = p.styles.StatusMuted.Render("["+source+"]") + " "
	}
	return fmt.Sprintf(" %s %s %s%s", p.styles.StatusMuted.Render(ts), sev, source, entry.Message)
}

// formatLogSource builds a compact source identifier from log entry attributes.
// Returns "rs/node" when both are available, "rs" when only RS is set, or ""
// when neither is available.
func formatLogSource(attrs map[string]any) string {
	rs, _ := attrs[sdk.LogKeyReplicaSet].(string)
	node, _ := attrs[sdk.LogKeyNode].(string)

	if rs == "" {
		return ""
	}
	if node == "" {
		return rs
	}
	// Shorten the node to just the port or short hostname for compactness.
	// Node format is typically "hostname:port".
	short := node
	if idx := strings.LastIndex(node, ":"); idx >= 0 {
		short = node[idx+1:]
	}
	return rs + "/" + short
}

// severityStyle returns the style for a log severity level.
func (p *logPanel) severityStyle(sev sdk.LogSeverity) lipgloss.Style {
	switch {
	case sev.Equal(sdk.LogSeverityError), sev.Equal(sdk.LogSeverityFatal):
		return p.styles.StatusError
	case sev.Equal(sdk.LogSeverityWarning):
		return p.styles.StatusWarning
	case sev.Equal(sdk.LogSeverityDebug):
		return p.styles.StatusMuted
	default:
		return lipgloss.NewStyle()
	}
}
