package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// newPanelViewport creates a viewport for use inside a panel. Keybindings
// and mouse are disabled so they don't conflict with global navigation.
func newPanelViewport() viewport.Model {
	vp := viewport.New(0, 0)
	vp.KeyMap = viewport.KeyMap{}
	vp.MouseWheelEnabled = false
	return vp
}

// replaceTitleBorder replaces the top border of a rendered lipgloss panel
// with a titled version: ╭─ Title ────────────╮
// outerW is the full panel width including borders. The title is rendered
// in bold with the border color.
func replaceTitleBorder(rendered, title string, outerW int,
	border lipgloss.Border, borderColor lipgloss.TerminalColor,
) string {
	styled := lipgloss.NewStyle().Bold(true).Foreground(borderColor).Render(title)
	return replaceStyledTitleBorder(rendered, styled, outerW, border, borderColor)
}

// replaceStyledTitleBorder replaces the top border of a rendered lipgloss
// panel with a pre-styled title string. The caller is responsible for all
// title styling; this function handles the border line layout.
// outerW is the full panel width including borders.
func replaceStyledTitleBorder(rendered, styledTitle string, outerW int,
	border lipgloss.Border, borderColor lipgloss.TerminalColor,
) string {
	bc := lipgloss.NewStyle().Foreground(borderColor)
	paddedTitle := " " + styledTitle + " "
	titleW := lipgloss.Width(paddedTitle)

	// Layout: corner(1) + pad(1) + title(titleW) + fill + corner(1) = outerW
	fill := outerW - 3 - titleW
	if fill < 0 {
		fill = 0
	}

	topLine := bc.Render(border.TopLeft+border.Top) +
		paddedTitle +
		bc.Render(strings.Repeat(border.Top, fill)+border.TopRight)

	lines := strings.SplitN(rendered, "\n", 2)
	if len(lines) == 2 {
		return topLine + "\n" + lines[1]
	}
	return topLine
}

// renderTitledPanel renders content inside a bordered panel with a title
// embedded in the top border line: ╭─ Title ────────────╮
// The title and border share the same color, which highlights on focus.
func renderTitledPanel(title, content string, style lipgloss.Style,
	width, height int, border lipgloss.Border,
	borderColor lipgloss.TerminalColor,
) string {
	panelStyle := style.Width(width).Height(height).BorderForeground(borderColor)
	rendered := panelStyle.Render(content)

	if title == "" {
		return rendered
	}

	outerW := width + panelBorderH
	return replaceTitleBorder(rendered, title, outerW, border, borderColor)
}

// helpColumnGap is the horizontal gap between the two help columns.
const helpColumnGap = 3

// helpEntry is a single key→description pair in the help overlay.
type helpEntry struct {
	key  string
	desc string
}

// helpFromBinding creates a helpEntry from a key.Binding's Help metadata.
func helpFromBinding(b key.Binding) helpEntry {
	h := b.Help()
	return helpEntry{key: h.Key, desc: h.Desc}
}

// helpSection is a titled group of keybinding entries.
type helpSection struct {
	title   string
	entries []helpEntry
}

// helpCombined creates a helpEntry with two bindings shown as "a / b desc",
// using the lowercase/uppercase convention (e.g. "s / S backup").
// Pass the lowercase binding first.
func helpCombined(a, b key.Binding, desc string) helpEntry {
	return helpEntry{
		key:  a.Help().Key + " / " + b.Help().Key,
		desc: desc,
	}
}

// helpColumns returns the help overlay content organized into two columns.
// Left: Navigation, Global, General. Right: tab-specific sections.
// Entries are derived from the actual key.Binding definitions in keys.go,
// so the help overlay stays in sync with keybinding changes.
// When readonly is true, mutation entries (backup, cancel, delete, restore,
// set config, resync) are omitted.
func helpColumns(readonly bool) (left, right []helpSection) {
	left = []helpSection{
		{"Navigation", []helpEntry{
			{
				key:  globalKeys.NextPanel.Help().Key + " / " + globalKeys.PrevPanel.Help().Key,
				desc: "next / prev panel",
			},
			helpFromBinding(globalKeys.Up),
			helpFromBinding(globalKeys.Down),
			{"1-3", "jump to tab"},
		}},
	}
	if !readonly {
		left = append(left, helpSection{"Global", []helpEntry{
			helpCombined(backupKeys.Start, backupKeys.StartCustom, "backup"),
			helpFromBinding(backupKeys.Cancel),
			helpFromBinding(globalKeys.Delete),
			helpFromBinding(globalKeys.PITRToggle),
		}})
	}
	left = append(left, helpSection{"General", []helpEntry{
		helpFromBinding(globalKeys.Help),
		{key: globalKeys.Back.Help().Key, desc: "back / dismiss"},
		helpFromBinding(globalKeys.Quit),
	}})

	right = []helpSection{
		{"1:Overview", []helpEntry{
			helpFromBinding(overviewKeys.Toggle),
			helpFromBinding(overviewKeys.Follow),
			helpFromBinding(overviewKeys.Wrap),
			helpFromBinding(overviewKeys.Filter),
		}},
	}

	if readonly {
		// Backups tab: only navigation (toggle), no mutations.
		right = append(right, helpSection{"2:Backups", []helpEntry{
			helpFromBinding(backupKeys.Toggle),
		}})
		// Config tab: no mutation bindings, section omitted entirely.
	} else {
		right = append(right, helpSection{"2:Backups", []helpEntry{
			helpFromBinding(backupKeys.Toggle),
			helpCombined(backupKeys.RestoreSelected, backupKeys.Restore, "restore"),
			helpFromBinding(backupKeys.BulkDelete),
		}})
		right = append(right, helpSection{"3:Config", []helpEntry{
			helpCombined(configKeys.SetConfigSelected, configKeys.SetConfig, "set config"),
			helpCombined(configKeys.ResyncSelected, configKeys.Resync, "resync"),
			helpFromBinding(configKeys.Edit),
		}})
	}
	return
}

// renderHelpColumn renders a slice of help sections into a single column string.
func renderHelpColumn(sections []helpSection, keyStyle, descStyle, sectionStyle lipgloss.Style) string {
	var b strings.Builder
	for i, section := range sections {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(sectionStyle.Render(section.title))
		b.WriteByte('\n')
		for _, entry := range section.entries {
			b.WriteString(fmt.Sprintf("  %s  %s", keyStyle.Render(entry.key), descStyle.Render(entry.desc)))
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// renderHelpOverlay renders a centered two-column help panel showing all
// keybindings organized by category. Left column has navigation, global, and
// general bindings. Right column has tab-specific sections.
// When readonly is true, mutation entries are omitted.
func renderHelpOverlay(styles *Styles, contentW, contentH int, readonly bool) string {
	keyStyle := styles.HintKey
	descStyle := lipgloss.NewStyle().Foreground(styles.FocusedBorderColor)
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.FocusedBorderColor)

	left, right := helpColumns(readonly)
	leftStr := renderHelpColumn(left, keyStyle, descStyle, sectionStyle)
	rightStr := renderHelpColumn(right, keyStyle, descStyle, sectionStyle)

	gap := strings.Repeat(" ", helpColumnGap)
	body := lipgloss.JoinHorizontal(lipgloss.Top, leftStr, gap, rightStr)

	border := lipgloss.RoundedBorder()
	borderColor := styles.FocusedBorderColor

	// Let lipgloss measure the body width; add padding.
	bodyWidth := lipgloss.Width(body)
	panelWidth := bodyWidth + panelPaddingH

	panel := lipgloss.NewStyle().
		Border(border).
		BorderForeground(borderColor).
		Padding(1, 1).
		Width(panelWidth).
		Render(body)

	outerW := panelWidth + panelBorderH
	panel = replaceTitleBorder(panel, "Help", outerW, border, borderColor)

	return lipgloss.Place(contentW, contentH,
		lipgloss.Center, lipgloss.Center,
		panel)
}

// renderCursorList renders a list of pre-rendered lines with cursor highlighting.
// The line at cursor index gets a ▶ prefix when focused, or bold-only when
// unfocused. All other lines get a plain two-space indent to align with the
// cursor prefix.
func renderCursorList(lines []string, cursor int, focused bool, styles *Styles) string {
	cursorStyle := lipgloss.NewStyle().Foreground(styles.FocusedBorderColor)

	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i == cursor {
			if focused {
				line = cursorStyle.Render("▶ ") + styles.Bold.Render(line)
			} else {
				line = "  " + styles.Bold.Render(line)
			}
		} else {
			line = "  " + line
		}
		b.WriteString(line)
	}
	return b.String()
}

// humanBytes formats a byte count into a human-readable string.
func humanBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1fGB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1fMB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1fKB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

// relativeTime returns a human-readable relative time string.
func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
}
