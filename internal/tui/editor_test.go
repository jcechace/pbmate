package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildEditorCmd(t *testing.T) {
	tests := []struct {
		name     string
		editor   string
		tmpPath  string
		wantArgs []string // full Args slice (Args[0] = command name)
	}{
		{
			name:     "simple editor",
			editor:   "vim",
			tmpPath:  "/tmp/pbmate-123.yaml",
			wantArgs: []string{"vim", "/tmp/pbmate-123.yaml"},
		},
		{
			name:     "editor with flag",
			editor:   "code -w",
			tmpPath:  "/tmp/pbmate-456.yaml",
			wantArgs: []string{"code", "-w", "/tmp/pbmate-456.yaml"},
		},
		{
			name:     "editor with multiple flags",
			editor:   "nvim --clean -u NONE",
			tmpPath:  "/tmp/test.yaml",
			wantArgs: []string{"nvim", "--clean", "-u", "NONE", "/tmp/test.yaml"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := buildEditorCmd(tt.editor, tt.tmpPath)
			assert.Equal(t, tt.wantArgs, cmd.Args)
		})
	}
}

func TestBuildEditorCmdPanicsOnEmpty(t *testing.T) {
	assert.Panics(t, func() {
		buildEditorCmd("", "/tmp/test.yaml")
	})
}
