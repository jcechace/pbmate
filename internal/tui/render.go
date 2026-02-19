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
	header := lipgloss.NewStyle().Bold(true).Foreground(styles.FocusedBorderColor)
	b.WriteString(header.Render("Backup"))
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
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("  Replica Sets"))
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
