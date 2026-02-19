package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	sdk "github.com/jcechace/pbmate/sdk/v2"
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
// outerW is the full panel width including borders.
func replaceTitleBorder(rendered, title string, outerW int,
	border lipgloss.Border, borderColor lipgloss.TerminalColor,
) string {
	bc := lipgloss.NewStyle().Foreground(borderColor)
	tc := lipgloss.NewStyle().Bold(true).Foreground(borderColor)
	titleStr := tc.Render(" " + title + " ")
	titleW := lipgloss.Width(titleStr)

	// Layout: corner(1) + pad(1) + title(titleW) + fill + corner(1) = outerW
	fill := outerW - 3 - titleW
	if fill < 0 {
		fill = 0
	}

	topLine := bc.Render(border.TopLeft+border.Top) +
		titleStr +
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

// helpOverlayWidth is the content width inside the help overlay panel.
const helpOverlayWidth = 38

// renderHelpOverlay renders a centered help panel showing all keybindings
// organized by category.
func renderHelpOverlay(styles Styles, contentW, contentH int) string {
	keyStyle := styles.HintKey
	descStyle := lipgloss.NewStyle().Foreground(styles.FocusedBorderColor)
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.FocusedBorderColor)

	line := func(k, desc string) string {
		return fmt.Sprintf("  %s  %s", keyStyle.Render(k), descStyle.Render(desc))
	}

	var b strings.Builder

	b.WriteString(sectionStyle.Render("Navigation"))
	b.WriteByte('\n')
	b.WriteString(line("] / [", "next / prev panel"))
	b.WriteByte('\n')
	b.WriteString(line("up/k", "up"))
	b.WriteByte('\n')
	b.WriteString(line("down/j", "down"))
	b.WriteByte('\n')
	b.WriteString(line("tab", "next tab"))
	b.WriteByte('\n')
	b.WriteString(line("1-4", "jump to tab"))
	b.WriteByte('\n')

	b.WriteByte('\n')
	b.WriteString(sectionStyle.Render("Actions"))
	b.WriteByte('\n')
	b.WriteString(line("s", "start backup"))
	b.WriteByte('\n')
	b.WriteString(line("S", "custom backup"))
	b.WriteByte('\n')
	b.WriteString(line("c", "cancel backup"))
	b.WriteByte('\n')
	b.WriteString(line("d", "delete"))
	b.WriteByte('\n')

	b.WriteByte('\n')
	b.WriteString(sectionStyle.Render("Overview"))
	b.WriteByte('\n')
	b.WriteString(line("space", "expand / collapse"))
	b.WriteByte('\n')
	b.WriteString(line("f", "follow logs"))
	b.WriteByte('\n')
	b.WriteString(line("w", "wrap logs"))
	b.WriteByte('\n')

	b.WriteByte('\n')
	b.WriteString(sectionStyle.Render("General"))
	b.WriteByte('\n')
	b.WriteString(line("?", "help"))
	b.WriteByte('\n')
	b.WriteString(line("esc", "back / dismiss"))
	b.WriteByte('\n')
	b.WriteString(line("q", "quit"))

	body := b.String()
	border := lipgloss.RoundedBorder()
	borderColor := styles.FocusedBorderColor

	panelWidth := helpOverlayWidth + panelPaddingH

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

// statusIndicator returns a colored status dot for a PBM status.
func statusIndicator(s sdk.Status, styles *Styles) string {
	switch {
	case s.Equal(sdk.StatusDone):
		return styles.StatusOK.Render("●")
	case s.Equal(sdk.StatusError), s.Equal(sdk.StatusPartlyDone):
		return styles.StatusError.Render("●")
	case s.Equal(sdk.StatusCancelled):
		return styles.StatusMuted.Render("●")
	case s.IsTerminal():
		return styles.StatusMuted.Render("●")
	default:
		// Running / in-progress states.
		return styles.StatusWarning.Render("●")
	}
}

// agentIndicator returns a colored status dot for an agent.
func agentIndicator(a *sdk.Agent, styles *Styles) string {
	if a.Stale {
		return styles.StatusMuted.Render("○")
	}
	if !a.OK || len(a.Errors) > 0 {
		return styles.StatusError.Render("●")
	}
	return styles.StatusOK.Render("●")
}

// renderBackupDetail writes full backup detail to the builder.
func renderBackupDetail(b *strings.Builder, bk *sdk.Backup, styles *Styles) {
	b.WriteString(styles.SectionHeader.Render("Backup"))
	b.WriteByte('\n')

	fmt.Fprintf(b, "  Name:        %s\n", bk.Name)
	fmt.Fprintf(b, "  Type:        %s\n", bk.Type)

	ind := statusIndicator(bk.Status, styles)
	fmt.Fprintf(b, "  Status:      %s %s\n", ind, bk.Status)

	if bk.Size > 0 {
		fmt.Fprintf(b, "  Size:        %s", humanBytes(bk.Size))
		if bk.SizeUncompressed > 0 {
			fmt.Fprintf(b, " (%s uncompressed)", humanBytes(bk.SizeUncompressed))
		}
		b.WriteByte('\n')
	}

	if !bk.Compression.IsZero() {
		fmt.Fprintf(b, "  Compression: %s\n", bk.Compression)
	}
	if !bk.ConfigName.IsZero() {
		fmt.Fprintf(b, "  Config:      %s\n", bk.ConfigName)
	}
	if !bk.StartTS.IsZero() {
		fmt.Fprintf(b, "  Started:     %s\n", bk.StartTS.Format("2006-01-02 15:04:05"))
	}
	if !bk.LastTransitionTS.IsZero() && !bk.StartTS.IsZero() {
		dur := bk.LastTransitionTS.Sub(bk.StartTS).Truncate(time.Second)
		if dur > 0 {
			fmt.Fprintf(b, "  Duration:    %s\n", dur)
		}
	}

	if bk.Error != "" {
		fmt.Fprintf(b, "  Error:       %s\n", styles.StatusError.Render(bk.Error))
	}

	if len(bk.Replsets) > 0 {
		b.WriteByte('\n')
		b.WriteString(styles.Bold.Render("  Replica Sets"))
		b.WriteByte('\n')
		for _, rs := range bk.Replsets {
			rsInd := statusIndicator(rs.Status, styles)
			node := rs.Node
			if node == "" {
				node = "-"
			}
			fmt.Fprintf(b, "  %s %s: %s  (%s)\n", rsInd, rs.Name, rs.Status, node)
		}
	}
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
