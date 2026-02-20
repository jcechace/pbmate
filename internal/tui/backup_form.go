package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

const (
	// backupFormInnerWidth is the content width inside the form panel,
	// excluding border and padding.
	backupFormInnerWidth = 40

	// defaultConfigName is the name of the default (main) storage profile.
	defaultConfigName = "main"
)

// backupFormKind distinguishes between the quick confirm and the full wizard.
type backupFormKind int

const (
	backupFormQuick backupFormKind = iota
	backupFormFull
)

// backupFormResult holds the user's selections from the backup form.
type backupFormResult struct {
	backupType  string
	compression string
	configName  string
	namespaces  string
	incrBase    bool
	confirmed   bool // true = start, false = customize (quick form only)

	// Profiles are stored for handoff from quick → full form.
	profiles []sdk.StorageProfile
}

// toOptions converts the form result into SDK StartBackupOptions.
func (r backupFormResult) toOptions() sdk.StartBackupOptions {
	opts := sdk.StartBackupOptions{}

	switch r.backupType {
	case "logical":
		opts.Type = sdk.BackupTypeLogical
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
	// "default" / "none" → zero value, server decides.

	if r.configName != defaultConfigName {
		cn, err := sdk.NewConfigName(r.configName)
		if err == nil {
			opts.ConfigName = cn
		}
	}

	if r.namespaces != "" && r.namespaces != "*.*" {
		opts.Namespaces = strings.Split(r.namespaces, ",")
		for i := range opts.Namespaces {
			opts.Namespaces[i] = strings.TrimSpace(opts.Namespaces[i])
		}
	}

	opts.IncrBase = r.incrBase

	return opts
}

// --- Quick backup form ---

// newQuickBackupForm creates a compact confirm form for starting a backup
// with defaults. The user can confirm ("Start") or choose to customize.
func newQuickBackupForm() (*huh.Form, *backupFormResult) {
	result := &backupFormResult{
		backupType:  "logical",
		compression: "default",
		configName:  defaultConfigName,
		confirmed:   true,
	}

	theme := huh.ThemeCatppuccin()
	theme.Focused.Base = theme.Focused.Base.BorderStyle(lipgloss.HiddenBorder())

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Start Backup").
				Description("Logical backup to **Main** storage."),

			huh.NewConfirm().
				Affirmative("Start").
				Negative("Customize (c)").
				Value(&result.confirmed),
		),
	).
		WithTheme(theme).
		WithWidth(backupFormInnerWidth).
		WithShowHelp(false).
		WithShowErrors(false).
		WithKeyMap(backupFormKeyMap())

	return form, result
}

// --- Full backup wizard ---

// newFullBackupForm creates a multi-group wizard form for configuring a backup.
// initialResult carries values from the quick form (or defaults if opened
// directly via S). profiles is the list of named storage profiles.
func newFullBackupForm(profiles []sdk.StorageProfile, initial *backupFormResult) (*huh.Form, *backupFormResult) {
	result := &backupFormResult{
		backupType:  "logical",
		compression: "default",
		configName:  defaultConfigName,
		confirmed:   true,
		profiles:    profiles,
	}
	if initial != nil {
		result.backupType = initial.backupType
		result.compression = initial.compression
		result.configName = initial.configName
	}

	// Profile options: Main is always first.
	profileOpts := []huh.Option[string]{
		huh.NewOption("Main", defaultConfigName),
	}
	for _, p := range profiles {
		profileOpts = append(profileOpts, huh.NewOption(p.Name.String(), p.Name.String()))
	}

	form := huh.NewForm(
		// Group 1: Type & Profile.
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Backup Type").
				Options(
					huh.NewOption("Logical", "logical"),
					huh.NewOption("Incremental", "incremental"),
				).
				Value(&result.backupType),

			huh.NewSelect[string]().
				Title("Profile").
				Options(profileOpts...).
				Value(&result.configName),
		),

		// Group 2: Compression.
		huh.NewGroup(
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

		// Group 3: Advanced — logical-specific options.
		huh.NewGroup(
			huh.NewInput().
				Title("Namespaces").
				Placeholder("*.*  (all)").
				Value(&result.namespaces),
		).WithHideFunc(func() bool {
			return result.backupType != "logical"
		}),

		// Group 4: Advanced — incremental-specific options.
		huh.NewGroup(
			huh.NewConfirm().
				Title("Use as incremental base?").
				Description("Starts a new incremental backup chain.").
				Affirmative("Yes").
				Negative("No").
				Value(&result.incrBase),
		).WithHideFunc(func() bool {
			return result.backupType != "incremental"
		}),

		// Group 5: Final confirmation.
		huh.NewGroup(
			huh.NewConfirm().
				TitleFunc(func() string {
					return fmt.Sprintf("Start %s backup?", result.backupType)
				}, &result.backupType).
				Affirmative("Start").
				Negative("Cancel").
				Value(&result.confirmed),
		),
	).
		WithTheme(huh.ThemeCatppuccin()).
		WithWidth(backupFormInnerWidth).
		WithShowHelp(false).
		WithShowErrors(false).
		WithKeyMap(backupFormKeyMap())

	return form, result
}

// --- Confirm form ---

// confirmFormResult holds the user's selection from a confirmation form.
type confirmFormResult struct {
	confirmed bool
}

// newConfirmForm creates a compact confirmation overlay with a description
// and Yes/No (or custom) buttons.
func newConfirmForm(description, affirmative, negative string) (*huh.Form, *confirmFormResult) {
	result := &confirmFormResult{confirmed: true}

	theme := huh.ThemeCatppuccin()
	theme.Focused.Base = theme.Focused.Base.BorderStyle(lipgloss.HiddenBorder())

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Description(description),

			huh.NewConfirm().
				Affirmative(affirmative).
				Negative(negative).
				Value(&result.confirmed),
		),
	).
		WithTheme(theme).
		WithWidth(backupFormInnerWidth).
		WithShowHelp(false).
		WithShowErrors(false).
		WithKeyMap(backupFormKeyMap())

	return form, result
}

// --- Shared ---

// backupFormKeyMap returns a huh KeyMap with ] and [ added to field
// navigation alongside the default tab/shift+tab/enter bindings.
func backupFormKeyMap() *huh.KeyMap {
	km := huh.NewDefaultKeyMap()

	km.Select.Next = key.NewBinding(key.WithKeys("enter", "tab", "]"))
	km.Select.Prev = key.NewBinding(key.WithKeys("shift+tab", "["))
	km.Confirm.Next = key.NewBinding(key.WithKeys("enter", "tab", "]"))
	km.Confirm.Prev = key.NewBinding(key.WithKeys("shift+tab", "["))
	km.Note.Next = key.NewBinding(key.WithKeys("enter", "tab", "]"))
	km.Note.Prev = key.NewBinding(key.WithKeys("shift+tab", "["))
	km.Input.Next = key.NewBinding(key.WithKeys("enter", "tab", "]"))
	km.Input.Prev = key.NewBinding(key.WithKeys("shift+tab", "["))

	return km
}

// renderFormOverlay renders the form centered over the content area inside
// a bordered panel with a title in the top border, using the same approach
// as renderTitledPanel.
func renderFormOverlay(form *huh.Form, title string, styles *Styles, contentW, contentH int) string {
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

	outerW := panelWidth + panelBorderH
	panel = replaceTitleBorder(panel, title, outerW, border, borderColor)

	// Center the panel in the content area.
	return lipgloss.Place(contentW, contentH,
		lipgloss.Center, lipgloss.Center,
		panel)
}
