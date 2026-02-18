package tui

import "github.com/charmbracelet/bubbles/key"

// globalKeyMap defines keybindings available in all views.
type globalKeyMap struct {
	Quit    key.Binding
	Tab1    key.Binding
	Tab2    key.Binding
	Tab3    key.Binding
	Tab4    key.Binding
	Tab5    key.Binding
	NextTab key.Binding
	PrevTab key.Binding
	Help    key.Binding
	Back    key.Binding
	Up      key.Binding
	Down    key.Binding
	Left    key.Binding
	Right   key.Binding
}

var globalKeys = globalKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Tab1: key.NewBinding(
		key.WithKeys("1"),
		key.WithHelp("1", "overview"),
	),
	Tab2: key.NewBinding(
		key.WithKeys("2"),
		key.WithHelp("2", "backups"),
	),
	Tab3: key.NewBinding(
		key.WithKeys("3"),
		key.WithHelp("3", "restores"),
	),
	Tab4: key.NewBinding(
		key.WithKeys("4"),
		key.WithHelp("4", "config"),
	),
	Tab5: key.NewBinding(
		key.WithKeys("5"),
		key.WithHelp("5", "logs"),
	),
	NextTab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next tab"),
	),
	PrevTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev tab"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("up/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("down/j", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("left/h", "left panel"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("right/l", "right panel"),
	),
}

// backupKeyMap defines keybindings specific to the Backups tab.
type backupKeyMap struct {
	Start  key.Binding
	Cancel key.Binding
	Delete key.Binding
}

var backupKeys = backupKeyMap{
	Start: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "start backup"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "cancel backup"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete backup"),
	),
}

// ShortHelp returns the key bindings shown in the compact help bar.
func (k globalKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Help, k.Quit, k.NextTab, k.Tab1, k.Tab2, k.Tab3, k.Tab4, k.Tab5,
	}
}

// FullHelp returns key bindings for the expanded help view.
func (k globalKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.NextTab, k.PrevTab, k.Tab1, k.Tab2, k.Tab3, k.Tab4, k.Tab5},
		{k.Help, k.Back, k.Quit},
	}
}
