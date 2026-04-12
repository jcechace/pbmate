package tui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// filePickerAllowedTypes restricts the file picker to YAML files.
var filePickerAllowedTypes = []string{".yaml", ".yml"}

// filePickerHeight is the number of visible rows in the file picker.
// Fits comfortably in typical 24-row terminals with room for chrome.
const filePickerHeight = 18

// filePickerOverlay wraps a file picker for selecting YAML config files.
// When needsConfirm is true, selecting a file transitions to a confirmation
// overlay before applying (used when overriding existing config/profiles).
type filePickerOverlay struct {
	picker       filepicker.Model
	title        string
	profile      string // target profile ("" = main config)
	isNew        bool   // creating new vs overwriting existing
	needsConfirm bool   // show confirm overlay before applying
	formTheme    huh.Theme
	ctx          context.Context
	client       *sdk.Client
}

func newFilePickerOverlay(ctx context.Context, client *sdk.Client, profile string, isNew bool, needsConfirm bool, formTheme huh.Theme, title string) (*filePickerOverlay, tea.Cmd) {
	fp := filepicker.New()
	fp.AllowedTypes = filePickerAllowedTypes
	fp.AutoHeight = false
	fp.SetHeight(filePickerHeight)
	fp.ShowHidden = false
	fp.ShowPermissions = false
	fp.ShowSize = true
	fp.DirAllowed = false
	fp.FileAllowed = true

	// Start from an absolute path so Back (filepath.Dir) can navigate up.
	if wd, err := os.Getwd(); err == nil {
		fp.CurrentDirectory = wd
	}

	// Customize keybindings: remove esc from Back (used for dismiss),
	// and use h/backspace/left for parent directory navigation.
	km := filepicker.DefaultKeyMap()
	km.Back = key.NewBinding(
		key.WithKeys("h", "backspace", "left"),
		key.WithHelp("h", "back"),
	)
	fp.KeyMap = km

	o := &filePickerOverlay{
		picker:       fp,
		title:        title,
		profile:      profile,
		isNew:        isNew,
		needsConfirm: needsConfirm,
		formTheme:    formTheme,
		ctx:          ctx,
		client:       client,
	}
	return o, o.picker.Init()
}

func (o *filePickerOverlay) Update(msg tea.Msg, back, quit key.Binding) (formOverlay, tea.Cmd) {
	if dismissOverlay(msg, back, quit) {
		return nil, nil
	}

	fp, cmd := o.picker.Update(msg)
	o.picker = fp

	if didSelect, path := o.picker.DidSelectFile(msg); didSelect {
		applyCmd := o.buildApplyCmd(path)
		if o.needsConfirm {
			title := "Override Config"
			desc := "Override existing main config?"
			if o.profile != "" {
				title = "Override Profile"
				desc = fmt.Sprintf("Override profile %q config?", o.profile)
			}
			overlay, cmd := newConfirmOverlay(o.formTheme, title, desc, "Override", "Cancel", applyCmd)
			return overlay, cmd
		}
		return nil, applyCmd
	}

	return o, cmd
}

// buildApplyCmd returns the tea.Cmd that applies the selected YAML file
// to the appropriate target (main config, existing profile, or new profile).
func (o *filePickerOverlay) buildApplyCmd(path string) tea.Cmd {
	if o.isNew {
		return applyProfileCmd(o.ctx, o.client, o.profile, path, "create profile")
	}
	if o.profile == "" {
		return applyConfigCmd(o.ctx, o.client, path)
	}
	return applyProfileCmd(o.ctx, o.client, o.profile, path, "set profile")
}

func (o *filePickerOverlay) View(styles *Styles, contentW, contentH int) string {
	return renderFilePickerOverlay(&o.picker, o.title, styles, contentW, contentH)
}

// =============================================================================
// File picker rendering
// =============================================================================

// filePickerInnerWidth is the content width inside the file picker panel.
const filePickerInnerWidth = 60

// renderFilePickerOverlay renders the file picker centered over the content
// area inside a bordered panel with a title and current path breadcrumb.
func renderFilePickerOverlay(fp *filepicker.Model, title string, styles *Styles, contentW, contentH int) string {
	// Current directory path — truncate from the left if too wide.
	dir := fp.CurrentDirectory
	maxPathW := filePickerInnerWidth - 2 // leave room for "…/" prefix
	if len(dir) > maxPathW {
		dir = "\u2026" + dir[len(dir)-maxPathW:]
	}
	pathLine := styles.StatusMuted.Render(dir)

	// Hint line for navigation.
	hintLine := styles.StatusMuted.Render("h:back  l:open  enter:select  esc:cancel")

	body := pathLine + "\n" +
		styles.StatusMuted.Render(strings.Repeat("\u2500", filePickerInnerWidth)) + "\n" +
		fp.View() + "\n" +
		styles.StatusMuted.Render(strings.Repeat("\u2500", filePickerInnerWidth)) + "\n" +
		hintLine

	border := lipgloss.RoundedBorder()
	borderColor := styles.FocusedBorderColor

	panelWidth := filePickerInnerWidth + panelPaddingH

	panel := lipgloss.NewStyle().
		Border(border).
		BorderForeground(borderColor).
		Padding(1, 1).
		Width(panelWidth).
		Render(body)

	outerW := panelWidth + panelBorderH
	panel = replaceTitleBorder(panel, title, outerW, border, borderColor)

	return lipgloss.Place(contentW, contentH,
		lipgloss.Center, lipgloss.Center,
		panel)
}
