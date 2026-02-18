package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	sdk "github.com/jcechace/pbmate/sdk/v2"

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

	ctx := context.Background()
	client, err := sdk.NewClient(ctx, sdk.WithMongoURI(*uri))
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer func() { _ = client.Close(ctx) }()

	theme := tui.ThemeByName(*themeName)
	m := tui.New(client, theme)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
