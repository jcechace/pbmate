package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// formOverlay is a modal overlay that captures all input while active.
// Update returns the next overlay state: self to continue, nil to dismiss,
// or a different overlay to transition (e.g. profile name → file picker).
type formOverlay interface {
	Update(msg tea.Msg, back, quit key.Binding) (formOverlay, tea.Cmd)
	View(styles *Styles, contentW, contentH int) string
}
