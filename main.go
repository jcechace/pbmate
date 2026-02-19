package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jcechace/pbmate/internal/tui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	uri := flag.String("uri", "", "MongoDB connection URI")
	themeName := flag.String("theme", "default", "Color theme (default, mocha, latte, frappe, macchiato)")
	flag.Parse()

	if *uri == "" {
		return fmt.Errorf("--uri is required")
	}

	theme := tui.ThemeByName(*themeName)
	m := tui.New(*uri, theme)
	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return err
	}
	if m, ok := result.(tui.Model); ok {
		m.Close()
	}
	return nil
}
