package tui

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// editConfigRequest is emitted by the config tab when the user presses 'e'
// to edit the selected config/profile in an external editor.
// profileName is empty for the main config.
type editConfigRequest struct {
	profileName string
}

// editConfigReadyMsg carries the fetched YAML for the config/profile to edit.
type editConfigReadyMsg struct {
	yaml        []byte
	profileName string // empty = main config
	err         error
}

// editorDoneMsg is sent after the external editor exits. It carries the
// original and edited YAML, the profile name, and the temp file path for
// cleanup.
type editorDoneMsg struct {
	original    []byte
	edited      []byte
	profileName string // empty = main config
	tmpPath     string
	err         error
}

// fetchEditYAMLCmd fetches the current YAML for the main config or a named
// profile, returning it in an editConfigReadyMsg.
func fetchEditYAMLCmd(ctx context.Context, client *sdk.Client, profileName string) tea.Cmd {
	return func() tea.Msg {
		var (
			yaml []byte
			err  error
		)
		if profileName == "" {
			yaml, err = client.Config.GetYAML(ctx, sdk.WithUnmasked())
		} else {
			yaml, err = client.Config.GetProfileYAML(ctx, profileName, sdk.WithUnmasked())
		}
		return editConfigReadyMsg{yaml: yaml, profileName: profileName, err: err}
	}
}

// buildEditorCmd constructs an exec.Cmd for the given editor command string
// and temp file path. The editor string may contain flags (e.g. "code -w").
// Panics if editor is empty — callers must ensure a valid editor
// (ResolveEditor guarantees at least "vi").
func buildEditorCmd(editor, tmpPath string) *exec.Cmd {
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		panic("buildEditorCmd: editor command must not be empty")
	}
	args := make([]string, len(parts)-1, len(parts))
	copy(args, parts[1:])
	args = append(args, tmpPath)
	return exec.Command(parts[0], args...)
}

// openEditorCmd writes the YAML to a temp file and returns a tea.Cmd that
// suspends the TUI and opens the external editor. When the editor exits,
// the callback reads the file back and sends an editorDoneMsg.
func openEditorCmd(editor string, yaml []byte, profileName string) tea.Cmd {
	// Write content to temp file before returning the Cmd.
	tmpFile, err := os.CreateTemp("", "pbmate-*.yaml")
	if err != nil {
		return func() tea.Msg {
			return editorDoneMsg{err: fmt.Errorf("create temp file: %w", err)}
		}
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(yaml); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return func() tea.Msg {
			return editorDoneMsg{err: fmt.Errorf("write temp file: %w", err)}
		}
	}
	_ = tmpFile.Close()

	cmd := buildEditorCmd(editor, tmpPath)

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			_ = os.Remove(tmpPath)
			return editorDoneMsg{err: fmt.Errorf("editor: %w", err)}
		}

		edited, readErr := os.ReadFile(tmpPath)
		if readErr != nil {
			_ = os.Remove(tmpPath)
			return editorDoneMsg{err: fmt.Errorf("read temp file: %w", readErr)}
		}

		return editorDoneMsg{
			original:    yaml,
			edited:      edited,
			profileName: profileName,
			tmpPath:     tmpPath,
		}
	})
}

// applyEditedConfigCmd returns a tea.Cmd that applies the edited YAML
// to the main config or a named profile. On success the temp file is
// deleted. On failure the temp file is preserved and its path is
// included in the error message so the user can recover their edits.
func applyEditedConfigCmd(ctx context.Context, client *sdk.Client, edited []byte, profileName, tmpPath string) tea.Cmd {
	return func() tea.Msg {
		r := bytes.NewReader(edited)
		var err error
		if profileName == "" {
			err = client.Config.SetYAML(ctx, r)
		} else {
			_, err = client.Config.SetProfile(ctx, profileName, r)
		}

		action := "edit config"
		if profileName != "" {
			action = fmt.Sprintf("edit profile %s", profileName)
		}

		if err != nil {
			// Preserve temp file so user can recover edits.
			return actionResultMsg{
				action: action,
				err:    fmt.Errorf("%w (edits saved to %s)", err, tmpPath),
			}
		}

		// Success — clean up temp file.
		_ = os.Remove(tmpPath)
		return actionResultMsg{action: action}
	}
}
