package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

const (
	// formOverlayMinWidth is the minimum content width for form overlays.
	formOverlayMinWidth = 40

	// formOverlayMaxWidth is the maximum content width for form overlays.
	formOverlayMaxWidth = 60

	// formOverlayWidthPct is the percentage of terminal width used for overlays.
	formOverlayWidthPct = 50

	// formOverlayDefaultWidth is the form construction-time width.
	// renderFormOverlay may adapt the actual panel width at render time.
	formOverlayDefaultWidth = formOverlayMaxWidth

	// defaultConfigName is the name of the default (main) storage profile.
	defaultConfigName = "main"
)

// formOverlayInnerWidth computes an adaptive content width for form overlays
// based on the available terminal width. The result is clamped between
// formOverlayMinWidth and formOverlayMaxWidth.
func formOverlayInnerWidth(terminalW int) int {
	w := terminalW * formOverlayWidthPct / 100
	w -= panelBorderH + panelPaddingH // account for overlay chrome
	return max(min(w, formOverlayMaxWidth), formOverlayMinWidth)
}

// backupFormKind distinguishes between the quick confirm and the full wizard.
type backupFormKind int

const (
	backupFormQuick backupFormKind = iota
	backupFormFull
)

// backupFormResult holds the user's selections from the backup form.
type backupFormResult struct {
	backupType    string
	compression   string
	configName    string
	namespaces    string
	parallelColls string // number of parallel collections; "" = server default
	incrBase      bool
	confirmed     bool // true = start, false = customize (quick form only)

	// Profiles are stored for handoff from quick → full form.
	profiles []sdk.StorageProfile
}

// toCommand converts the form result into a sealed SDK StartBackupCommand.
func (r backupFormResult) toCommand() sdk.StartBackupCommand {
	configName := sdk.ConfigName{}
	if r.configName != defaultConfigName {
		if cn, err := sdk.NewConfigName(r.configName); err == nil {
			configName = cn
		}
	}

	compression := sdk.CompressionType{}
	// "default" leaves Compression as zero value (server decides).
	// All other values including "none" are parsed to their SDK equivalents.
	if r.compression != "default" {
		if ct, err := sdk.ParseCompressionType(r.compression); err == nil {
			compression = ct
		}
	}

	numParallelColls := parseOptionalInt(r.parallelColls)

	if r.backupType == "incremental" {
		return sdk.StartIncrementalBackup{
			ConfigName:       configName,
			Compression:      compression,
			Base:             r.incrBase,
			NumParallelColls: numParallelColls,
		}
	}

	// Default: logical backup.
	cmd := sdk.StartLogicalBackup{
		ConfigName:       configName,
		Compression:      compression,
		NumParallelColls: numParallelColls,
	}

	if r.namespaces != "" && r.namespaces != "*.*" {
		nss := strings.Split(r.namespaces, ",")
		for i := range nss {
			nss[i] = strings.TrimSpace(nss[i])
		}
		cmd.Namespaces = nss
	}

	return cmd
}

// parseOptionalInt parses a string to *int. Returns nil for empty or
// non-numeric input (which means "use server default").
func parseOptionalInt(s string) *int {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return nil
	}
	return &n
}

// --- Quick backup form ---

// newQuickBackupForm creates a compact confirm form for starting a backup
// with defaults. The user can confirm ("Start") or choose to customize.
func newQuickBackupForm(formTheme *huh.Theme) (*huh.Form, *backupFormResult) {
	result := &backupFormResult{
		backupType:  "logical",
		compression: "default",
		configName:  defaultConfigName,
		confirmed:   true,
	}

	form := newBorderlessForm([]*huh.Group{
		huh.NewGroup(
			huh.NewNote().
				Title("Start Backup").
				Description("Logical backup to **Main** storage."),

			huh.NewConfirm().
				Affirmative("Start").
				Negative("Customize (c)").
				Value(&result.confirmed),
		),
	}, formTheme)

	return form, result
}

// --- Full backup form ---

// newFullBackupForm creates a single-screen form for configuring a backup.
// All essential fields are visible at once — no multi-step wizard.
// initialResult carries values from the quick form (or defaults if opened
// directly via S). profiles is the list of named storage profiles.
func newFullBackupForm(formTheme *huh.Theme, profiles []sdk.StorageProfile, initial *backupFormResult) (*huh.Form, *backupFormResult) {
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
		result.namespaces = initial.namespaces
		result.parallelColls = initial.parallelColls
		result.incrBase = initial.incrBase
	}

	// Profile options: Main is always first.
	profileOpts := []huh.Option[string]{
		huh.NewOption("Main", defaultConfigName),
	}
	for _, p := range profiles {
		profileOpts = append(profileOpts, huh.NewOption(p.Name.String(), p.Name.String()))
	}

	// Build groups dynamically based on backup type.
	// The form is rebuilt when the type changes (see backupFormOverlay.rebuildForm),
	// so we include only the groups relevant to the current type.
	groups := []*huh.Group{
		// Core fields — all inline selectors on one screen.
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Type").
				Options(
					huh.NewOption("Logical", "logical"),
					huh.NewOption("Incremental", "incremental"),
				).
				Inline(true).
				Value(&result.backupType),

			huh.NewSelect[string]().
				Title("Profile").
				Options(profileOpts...).
				Inline(true).
				Value(&result.configName),

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
				Inline(true).
				Value(&result.compression),
		),
	}

	if result.backupType == "logical" {
		groups = append(groups, huh.NewGroup(
			huh.NewInput().
				Title("Namespaces").
				Placeholder("*.*  (all)").
				Value(&result.namespaces),
		))
	}

	if result.backupType == "incremental" {
		groups = append(groups, huh.NewGroup(
			huh.NewConfirm().
				Title("Start new chain?").
				Inline(true).
				WithButtonAlignment(lipgloss.Right).
				Affirmative("Yes").
				Negative("No").
				Value(&result.incrBase),
		))
	}

	groups = append(groups,
		// Tuning + confirmation.
		huh.NewGroup(
			huh.NewInput().
				Title("Parallel Collections").
				Placeholder("server default").
				Value(&result.parallelColls),

			huh.NewConfirm().
				Title(fmt.Sprintf("Start %s backup?", result.backupType)).
				WithButtonAlignment(lipgloss.Left).
				Affirmative("Start").
				Negative("Cancel").
				Value(&result.confirmed),
		),
	)

	form := newStandardForm(groups, formTheme)

	return form, result
}

// --- Confirm form ---

// confirmFormResult holds the user's selection from a confirmation form.
type confirmFormResult struct {
	confirmed bool
}

// newConfirmForm creates a compact confirmation overlay with a description
// and Yes/No (or custom) buttons.
func newConfirmForm(formTheme *huh.Theme, description, affirmative, negative string) (*huh.Form, *confirmFormResult) {
	result := &confirmFormResult{confirmed: true}

	form := newBorderlessForm([]*huh.Group{
		huh.NewGroup(
			huh.NewNote().
				Description(description),

			huh.NewConfirm().
				Affirmative(affirmative).
				Negative(negative).
				Value(&result.confirmed),
		),
	}, formTheme)

	return form, result
}

// --- Form construction helpers ---

// newStandardForm creates a huh.Form with PBMate's standard configuration:
// stack layout, default overlay width, no help/errors, custom keymap.
func newStandardForm(groups []*huh.Group, theme *huh.Theme) *huh.Form {
	return huh.NewForm(groups...).
		WithTheme(theme).
		WithLayout(huh.LayoutStack).
		WithWidth(formOverlayDefaultWidth).
		WithShowHelp(false).
		WithShowErrors(false).
		WithKeyMap(formKeyMap())
}

// newBorderlessForm creates a standard form with hidden group borders.
// Used for compact overlays (confirms, quick backup) where the overlay
// panel border is sufficient.
func newBorderlessForm(groups []*huh.Group, theme *huh.Theme) *huh.Form {
	t := *theme
	t.Focused.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
	return newStandardForm(groups, &t)
}

// --- Shared overlay helpers ---

// dismissOverlay returns true if the message is a key press matching back or
// quit. Overlay Update methods use this to dismiss on Esc/q.
func dismissOverlay(msg tea.Msg, back, quit key.Binding) bool {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return key.Matches(keyMsg, back) || key.Matches(keyMsg, quit)
	}
	return false
}

// updateFormModel forwards a tea.Msg to a huh.Form, writing the (possibly new)
// *huh.Form pointer back through form. Returns the tea.Cmd from the update.
func updateFormModel(form **huh.Form, msg tea.Msg) tea.Cmd {
	formModel, cmd := (*form).Update(msg)
	if f, ok := formModel.(*huh.Form); ok {
		*form = f
	}
	return cmd
}

// initFormWithAdvance calls form.Init() and optionally advances one field
// (form.NextField). Used after dynamic form rebuilds where the focus should
// land on a field other than the first interactive one.
func initFormWithAdvance(form *huh.Form, advance bool) tea.Cmd {
	initCmd := form.Init()
	if advance {
		advanceCmd := form.NextField()
		return tea.Batch(initCmd, advanceCmd)
	}
	return initCmd
}

// --- Shared ---

// formKeyMap returns a huh KeyMap with ] and [ added to field
// navigation alongside the default tab/shift+tab/enter bindings.
// Shared by all form overlays (backup, restore, confirm, profile name).
func formKeyMap() *huh.KeyMap {
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
// as renderTitledPanel. Panel width adapts to terminal width.
func renderFormOverlay(form *huh.Form, title string, styles *Styles, contentW, contentH int) string {
	innerW := formOverlayInnerWidth(contentW)
	formView := form.View()
	border := lipgloss.RoundedBorder()
	borderColor := styles.FocusedBorderColor

	// panelWidth is the lipgloss Width value (content + padding, inside border).
	panelWidth := innerW + panelPaddingH

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
