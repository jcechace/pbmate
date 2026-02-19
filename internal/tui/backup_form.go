package tui

import (
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// Form layout constants.
const (
	backupFormWidth  = 50
	backupFormHeight = 16
)

// backupFormResult holds the user's selections from the start backup form.
type backupFormResult struct {
	backupType  string
	compression string
}

// toOptions converts the form result into SDK StartBackupOptions.
func (r backupFormResult) toOptions() sdk.StartBackupOptions {
	opts := sdk.StartBackupOptions{}

	switch r.backupType {
	case "logical":
		opts.Type = sdk.BackupTypeLogical
	case "physical":
		opts.Type = sdk.BackupTypePhysical
	case "incremental":
		opts.Type = sdk.BackupTypeIncremental
	}

	switch r.compression {
	case "gzip":
		opts.Compression = sdk.CompressionTypeGZIP
	case "pgzip":
		opts.Compression = sdk.CompressionTypePGZIP
	case "snappy":
		opts.Compression = sdk.CompressionTypeSNAPPY
	case "lz4":
		opts.Compression = sdk.CompressionTypeLZ4
	case "s2":
		opts.Compression = sdk.CompressionTypeS2
	case "zstd":
		opts.Compression = sdk.CompressionTypeZSTD
	}
	// "default" → zero value, server decides.

	return opts
}

// newBackupForm creates a huh.Form for starting a backup.
func newBackupForm() (*huh.Form, *backupFormResult) {
	result := &backupFormResult{
		backupType:  "logical",
		compression: "default",
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Backup Type").
				Options(
					huh.NewOption("Logical", "logical"),
					huh.NewOption("Physical", "physical"),
					huh.NewOption("Incremental", "incremental"),
				).
				Value(&result.backupType),

			huh.NewSelect[string]().
				Title("Compression").
				Options(
					huh.NewOption("Server default", "default"),
					huh.NewOption("s2", "s2"),
					huh.NewOption("zstd", "zstd"),
					huh.NewOption("gzip", "gzip"),
					huh.NewOption("pgzip", "pgzip"),
					huh.NewOption("snappy", "snappy"),
					huh.NewOption("lz4", "lz4"),
					huh.NewOption("None", "none"),
				).
				Value(&result.compression),
		),
	).WithShowHelp(false).WithShowErrors(false)

	return form, result
}

// renderFormOverlay renders the form centered over the content area inside
// a bordered panel.
func renderFormOverlay(form *huh.Form, styles Styles, contentW, contentH int) string {
	formView := form.View()

	// Wrap the form in a bordered panel.
	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.FocusedBorderColor).
		Padding(1, 2).
		Width(backupFormWidth).
		Render(formView)

	// Build a title line.
	title := " Start Backup "
	borderColor := styles.FocusedBorderColor
	titleStyle := lipgloss.NewStyle().Foreground(borderColor).Bold(true)
	panel = embedTitle(panel, titleStyle.Render(title))

	// Center the panel in the content area.
	return lipgloss.Place(contentW, contentH,
		lipgloss.Center, lipgloss.Center,
		panel)
}

// embedTitle replaces part of the first line's border with a title string.
// Assumes the panel uses RoundedBorder where the top-left is "╭" and
// the horizontal is "─".
func embedTitle(panel, title string) string {
	lines := strings.Split(panel, "\n")
	if len(lines) == 0 {
		return panel
	}

	top := lines[0]
	titleWidth := lipgloss.Width(title)
	topWidth := lipgloss.Width(top)

	// Need at least: "╭─" + title + "─╮"
	const minPrefixLen = 2 // "╭─"
	if topWidth < titleWidth+minPrefixLen+1 {
		return panel
	}

	// Build: prefix ("╭─") + title + remaining border + "╮"
	runes := []rune(top)
	prefix := string(runes[:minPrefixLen])
	suffix := string(runes[minPrefixLen+titleWidth:])
	lines[0] = prefix + title + suffix

	return strings.Join(lines, "\n")
}
