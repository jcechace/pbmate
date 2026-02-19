package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// backupFormInnerWidth is the content width inside the form panel,
// excluding border and padding.
const backupFormInnerWidth = 40

// backupFormResult holds the user's selections from the start backup form.
type backupFormResult struct {
	backupType  string
	compression string
	configName  string
	confirmed   bool
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

	if r.configName != "main" {
		cn, err := sdk.NewConfigName(r.configName)
		if err == nil {
			opts.ConfigName = cn
		}
	}

	return opts
}

// newBackupForm creates a huh.Form for starting a backup.
// profiles is the list of named storage profiles (may be empty).
func newBackupForm(profiles []sdk.StorageProfile) (*huh.Form, *backupFormResult) {
	result := &backupFormResult{
		backupType:  "logical",
		compression: "default",
		configName:  "main",
		confirmed:   true,
	}

	// Build profile options: Main is always first.
	profileOpts := []huh.Option[string]{
		huh.NewOption("Main", "main"),
	}
	for _, p := range profiles {
		profileOpts = append(profileOpts, huh.NewOption(p.Name.String(), p.Name.String()))
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

			huh.NewSelect[string]().
				Title("Profile").
				Options(profileOpts...).
				Value(&result.configName),

			huh.NewConfirm().
				TitleFunc(func() string {
					return fmt.Sprintf("Start %s backup?", result.backupType)
				}, &result.backupType).
				Value(&result.confirmed),
		),
	).WithShowHelp(false).WithShowErrors(false).WithKeyMap(backupFormKeyMap())

	return form, result
}

// backupFormKeyMap returns a huh KeyMap with ] and [ added to field
// navigation alongside the default tab/shift+tab/enter bindings.
func backupFormKeyMap() *huh.KeyMap {
	km := huh.NewDefaultKeyMap()

	km.Select.Next = key.NewBinding(key.WithKeys("enter", "tab", "]"))
	km.Select.Prev = key.NewBinding(key.WithKeys("shift+tab", "["))
	km.Confirm.Next = key.NewBinding(key.WithKeys("enter", "tab", "]"))
	km.Confirm.Prev = key.NewBinding(key.WithKeys("shift+tab", "["))

	return km
}

// renderFormOverlay renders the form centered over the content area inside
// a bordered panel with a title in the top border, using the same approach
// as renderTitledPanel.
func renderFormOverlay(form *huh.Form, styles Styles, contentW, contentH int) string {
	formView := form.View()
	border := lipgloss.RoundedBorder()
	borderColor := styles.FocusedBorderColor

	// panelWidth is the lipgloss Width value (content + padding, inside border).
	panelWidth := backupFormInnerWidth + panelPaddingH

	// Render the panel body (border + padding + content).
	panel := lipgloss.NewStyle().
		Border(border).
		BorderForeground(borderColor).
		Padding(1, 1).
		Width(panelWidth).
		Render(formView)

	// Build a titled top border line: ╭─ Start Backup ─────╮
	bc := lipgloss.NewStyle().Foreground(borderColor)
	tc := lipgloss.NewStyle().Bold(true).Foreground(borderColor)
	title := tc.Render(" Start Backup ")
	titleW := lipgloss.Width(title)

	outerW := panelWidth + panelBorderH
	fill := outerW - 3 - titleW // corner + pad + title + fill + corner
	if fill < 0 {
		fill = 0
	}

	topLine := bc.Render(border.TopLeft+border.Top) +
		title +
		bc.Render(strings.Repeat(border.Top, fill)+border.TopRight)

	// Replace the auto-generated top border with our titled one.
	lines := strings.SplitN(panel, "\n", 2)
	if len(lines) == 2 {
		panel = topLine + "\n" + lines[1]
	}

	// Center the panel in the content area.
	return lipgloss.Place(contentW, contentH,
		lipgloss.Center, lipgloss.Center,
		panel)
}
