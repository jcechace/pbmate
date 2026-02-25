package tui

import (
	"fmt"
	"strings"
	"time"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

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

	if bk.IsIncrementalBase() {
		fmt.Fprintf(b, "  Source:      %s base\n", styles.StatusWarning.Render("⌂"))
	} else if bk.IsIncremental() {
		fmt.Fprintf(b, "  Source:      %s\n", bk.SrcBackup)
	}

	ind := statusIndicator(bk.Status, styles)
	fmt.Fprintf(b, "  Status:      %s %s\n", ind, bk.Status)

	if !bk.LastWriteTS.IsZero() {
		restoreTime := bk.LastWriteTS.Time().UTC().Format("2006-01-02 15:04:05")
		fmt.Fprintf(b, "  Restore To:  %s\n", styles.Bold.Render(restoreTime))
	}

	// Oplog range — show when both timestamps are available.
	if !bk.FirstWriteTS.IsZero() && !bk.LastWriteTS.IsZero() {
		first := bk.FirstWriteTS.Time().UTC().Format("2006-01-02 15:04:05")
		last := bk.LastWriteTS.Time().UTC().Format("2006-01-02 15:04:05")
		fmt.Fprintf(b, "  Oplog Range: %s → %s\n", first, last)
	}

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

	if bk.IsSelective() {
		fmt.Fprintf(b, "  Namespaces:  %s\n", strings.Join(bk.Namespaces, ", "))
	} else {
		fmt.Fprintf(b, "  Namespaces:  %s\n", styles.StatusMuted.Render("*.* (all)"))
	}

	if !bk.StartTS.IsZero() {
		fmt.Fprintf(b, "  Started:     %s\n", bk.StartTS.UTC().Format("2006-01-02 15:04:05"))
	}
	if dur := bk.Elapsed().Truncate(time.Second); dur > 0 {
		fmt.Fprintf(b, "  Duration:    %s\n", dur)
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
			rsName := rs.Name
			if rs.IsConfigSvr {
				rsName += " (configsvr)"
			}
			line := fmt.Sprintf("  %s %s: %s  (%s)", rsInd, rsName, rs.Status, node)
			if rs.Size > 0 {
				line += fmt.Sprintf("  %s", humanBytes(rs.Size))
				if rs.SizeUncompressed > 0 {
					line += fmt.Sprintf(" / %s", humanBytes(rs.SizeUncompressed))
				}
			}
			b.WriteString(line)
			b.WriteByte('\n')
			if rs.Error != "" {
				fmt.Fprintf(b, "      %s\n", styles.StatusError.Render(rs.Error))
			}
		}
	}
}

// renderRestoreDetail writes full restore detail to the builder.
func renderRestoreDetail(b *strings.Builder, rs *sdk.Restore, styles *Styles) {
	b.WriteString(styles.SectionHeader.Render("Restore"))
	b.WriteByte('\n')

	fmt.Fprintf(b, "  Name:        %s\n", rs.Name)
	fmt.Fprintf(b, "  Backup:      %s\n", rs.Backup)
	fmt.Fprintf(b, "  Type:        %s\n", rs.Type)

	ind := statusIndicator(rs.Status, styles)
	fmt.Fprintf(b, "  Status:      %s %s\n", ind, rs.Status)

	if !rs.StartTS.IsZero() {
		fmt.Fprintf(b, "  Started:     %s\n", rs.StartTS.UTC().Format("2006-01-02 15:04:05"))
	}
	if !rs.LastTransitionTS.IsZero() {
		fmt.Fprintf(b, "  Finished:    %s\n", rs.LastTransitionTS.UTC().Format("2006-01-02 15:04:05"))
	}
	if dur := rs.Elapsed().Truncate(time.Second); dur > 0 {
		fmt.Fprintf(b, "  Duration:    %s\n", dur)
	}

	if !rs.PITRTarget.IsZero() {
		fmt.Fprintf(b, "  PITR Target: %s\n", rs.PITRTarget.Time().UTC().Format("2006-01-02 15:04:05"))
	}

	if len(rs.Namespaces) > 0 {
		fmt.Fprintf(b, "  Namespaces:  %s\n", strings.Join(rs.Namespaces, ", "))
	}

	if rs.Error != "" {
		fmt.Fprintf(b, "  Error:       %s\n", styles.StatusError.Render(rs.Error))
	}

	if len(rs.Replsets) > 0 {
		b.WriteByte('\n')
		b.WriteString(styles.Bold.Render("  Replica Sets"))
		b.WriteByte('\n')
		for _, rrs := range rs.Replsets {
			rsInd := statusIndicator(rrs.Status, styles)
			fmt.Fprintf(b, "  %s %s: %s\n", rsInd, rrs.Name, rrs.Status)
			if rrs.Error != "" {
				fmt.Fprintf(b, "      %s\n", styles.StatusError.Render(rrs.Error))
			}
			for _, node := range rrs.Nodes {
				nodeInd := statusIndicator(node.Status, styles)
				fmt.Fprintf(b, "      %s %s: %s\n", nodeInd, node.Name, node.Status)
				if node.Error != "" {
					fmt.Fprintf(b, "          %s\n", styles.StatusError.Render(node.Error))
				}
			}
		}
	}
}
