package tui

import "github.com/charmbracelet/bubbles/key"

// globalKeyMap defines keybindings available in all views.
type globalKeyMap struct {
	Quit      key.Binding
	Tab1      key.Binding
	Tab2      key.Binding
	Tab3      key.Binding
	NextPanel key.Binding
	PrevPanel key.Binding
	Help      key.Binding
	Back      key.Binding
	Up        key.Binding
	Down      key.Binding
	Delete    key.Binding
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
		key.WithHelp("3", "config"),
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
	NextPanel: key.NewBinding(
		key.WithKeys("]"),
		key.WithHelp("]", "next panel"),
	),
	PrevPanel: key.NewBinding(
		key.WithKeys("["),
		key.WithHelp("[", "prev panel"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
}

// overviewKeyMap defines keybindings specific to the Overview tab.
type overviewKeyMap struct {
	Toggle key.Binding
	Follow key.Binding
	Wrap   key.Binding
}

var overviewKeys = overviewKeyMap{
	Toggle: key.NewBinding(
		key.WithKeys(" ", "enter"),
		key.WithHelp("space", "expand/collapse"),
	),
	Follow: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "follow logs"),
	),
	Wrap: key.NewBinding(
		key.WithKeys("w"),
		key.WithHelp("w", "wrap logs"),
	),
}

// backupKeyMap defines keybindings specific to the Backups tab.
type backupKeyMap struct {
	Start           key.Binding
	StartCustom     key.Binding
	Cancel          key.Binding
	Restore         key.Binding
	RestoreSelected key.Binding
	Toggle          key.Binding
}

// configKeyMap defines keybindings specific to the Config tab.
type configKeyMap struct {
	SetConfig         key.Binding
	SetConfigSelected key.Binding
	Resync            key.Binding
	ResyncSelected    key.Binding
}

var configKeys = configKeyMap{
	SetConfig: key.NewBinding(
		key.WithKeys("C"),
		key.WithHelp("C", "set config"),
	),
	SetConfigSelected: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "set config selected"),
	),
	Resync: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "resync"),
	),
	ResyncSelected: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "resync selected"),
	),
}

var backupKeys = backupKeyMap{
	Start: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "backup"),
	),
	StartCustom: key.NewBinding(
		key.WithKeys("S"),
		key.WithHelp("S", "custom backup"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("X"),
		key.WithHelp("X", "cancel backup"),
	),
	Restore: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "restore"),
	),
	RestoreSelected: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "restore selected"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "backups/restores"),
	),
}
