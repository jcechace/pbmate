package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/alecthomas/kong"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jcechace/pbmate/internal/config"
	"github.com/jcechace/pbmate/internal/tui"
)

// version is set at build time by GoReleaser via -ldflags.
var version = "dev"

// configFilePath is a named type to avoid kong DI type collisions with
// plain string parameters. It holds the resolved path to the config file.
type configFilePath string

// cli is the root kong CLI struct for PBMate.
type cli struct {
	Config  string           `help:"Config file path." type:"path" env:"PBMATE_CONFIG"`
	Version kong.VersionFlag `help:"Print version and exit."`
	TUI     tuiCmd           `cmd:"" default:"withargs" help:"Start the TUI (default)."`
	Context contextCmd       `cmd:"" help:"Manage connection contexts."`
}

// tuiCmd starts the TUI with the resolved connection settings.
type tuiCmd struct {
	URI      string `help:"MongoDB URI (overrides context)." optional:""`
	Context  string `help:"Use a named context (overrides current-context)." optional:"" name:"context"`
	Theme    string `help:"Color theme override (default, mocha, latte, frappe, macchiato)." optional:""`
	Readonly *bool  `help:"Readonly mode (disable mutations)." optional:"" negatable:""`
}

func (cmd *tuiCmd) Run(cfg *config.AppConfig) error {
	uri, err := cfg.ResolveURI(cmd.URI, cmd.Context)
	if err != nil {
		return err
	}

	themeName := cfg.ResolveTheme(cmd.Theme, cmd.Context)
	theme := tui.ThemeByName(themeName)

	// Determine the context name for the header display.
	// Only show it when a named context is used, not for direct --uri.
	contextName := cmd.Context
	if contextName == "" && cmd.URI == "" {
		contextName = cfg.CurrentContext
	}

	readonly := cfg.ResolveReadonly(cmd.Readonly, cmd.Context)

	m := tui.New(tui.Options{
		URI:         uri,
		Theme:       theme,
		ContextName: contextName,
		Readonly:    readonly,
	})
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

// contextCmd is the parent for context management subcommands.
type contextCmd struct {
	List    contextListCmd    `cmd:"" help:"List all contexts."`
	Current contextCurrentCmd `cmd:"" help:"Print the current context."`
	Use     contextUseCmd     `cmd:"" help:"Switch the active context."`
	Add     contextAddCmd     `cmd:"" help:"Add a new context."`
	Remove  contextRemoveCmd  `cmd:"" help:"Remove a context."`
}

// contextListCmd lists all configured contexts.
type contextListCmd struct{}

func (cmd *contextListCmd) Run(cfg *config.AppConfig) error {
	if len(cfg.Contexts) == 0 {
		fmt.Println("No contexts configured. Add one with: pbmate context add <name> --uri=<uri>")
		return nil
	}

	// Sort names for stable output.
	names := make([]string, 0, len(cfg.Contexts))
	for name := range cfg.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		ctx := cfg.Contexts[name]
		marker := "  "
		if name == cfg.CurrentContext {
			marker = "* "
		}
		line := fmt.Sprintf("%s%-20s %s", marker, name, ctx.URI)
		if ctx.Theme != "" {
			line += fmt.Sprintf("  (theme: %s)", ctx.Theme)
		}
		if ctx.Readonly != nil && *ctx.Readonly {
			line += "  (readonly)"
		}
		fmt.Println(line)
	}
	return nil
}

// contextCurrentCmd prints the current context name and URI.
type contextCurrentCmd struct{}

func (cmd *contextCurrentCmd) Run(cfg *config.AppConfig) error {
	if cfg.CurrentContext == "" {
		fmt.Println("No current context set. Use: pbmate context use <name>")
		return nil
	}

	ctx := cfg.CurrentCtx()
	if ctx == nil {
		return fmt.Errorf("current context %q not found in config", cfg.CurrentContext)
	}

	fmt.Printf("%s (%s)\n", cfg.CurrentContext, ctx.URI)
	return nil
}

// contextUseCmd switches the active context.
type contextUseCmd struct {
	Name string `arg:"" help:"Context name to activate."`
}

func (cmd *contextUseCmd) Run(cfg *config.AppConfig, path configFilePath) error {
	if _, ok := cfg.Contexts[cmd.Name]; !ok {
		return fmt.Errorf("context %q not found; available: %s", cmd.Name, contextNameList(cfg))
	}

	cfg.CurrentContext = cmd.Name
	if err := cfg.Save(string(path)); err != nil {
		return err
	}

	fmt.Printf("Switched to context %q.\n", cmd.Name)
	return nil
}

// contextAddCmd adds a new context.
type contextAddCmd struct {
	Name     string `arg:"" help:"Context name."`
	URI      string `required:"" help:"MongoDB connection URI."`
	Theme    string `optional:"" help:"Theme override for this context."`
	Readonly *bool  `optional:"" help:"Readonly mode for this context." negatable:""`
}

func (cmd *contextAddCmd) Run(cfg *config.AppConfig, path configFilePath) error {
	if cfg.Contexts == nil {
		cfg.Contexts = make(map[string]config.Context)
	}

	if _, exists := cfg.Contexts[cmd.Name]; exists {
		return fmt.Errorf("context %q already exists; remove it first or edit the config file", cmd.Name)
	}

	ctx := config.Context{
		URI:      cmd.URI,
		Theme:    cmd.Theme,
		Readonly: cmd.Readonly,
	}
	cfg.Contexts[cmd.Name] = ctx

	// If this is the first context, make it current.
	if cfg.CurrentContext == "" {
		cfg.CurrentContext = cmd.Name
	}

	if err := cfg.Save(string(path)); err != nil {
		return err
	}

	if cfg.CurrentContext == cmd.Name {
		fmt.Printf("Added and activated context %q.\n", cmd.Name)
	} else {
		fmt.Printf("Added context %q.\n", cmd.Name)
	}
	return nil
}

// contextRemoveCmd removes a context.
type contextRemoveCmd struct {
	Name string `arg:"" help:"Context name to remove."`
}

func (cmd *contextRemoveCmd) Run(cfg *config.AppConfig, path configFilePath) error {
	if _, ok := cfg.Contexts[cmd.Name]; !ok {
		return fmt.Errorf("context %q not found", cmd.Name)
	}

	delete(cfg.Contexts, cmd.Name)

	// Clear current-context if it was the removed one.
	if cfg.CurrentContext == cmd.Name {
		cfg.CurrentContext = ""
	}

	if err := cfg.Save(string(path)); err != nil {
		return err
	}

	fmt.Printf("Removed context %q.\n", cmd.Name)
	return nil
}

// contextNameList returns a comma-separated list of context names for error messages.
func contextNameList(cfg *config.AppConfig) string {
	if len(cfg.Contexts) == 0 {
		return "(none)"
	}
	names := make([]string, 0, len(cfg.Contexts))
	for name := range cfg.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func main() {
	var c cli
	kongCtx := kong.Parse(&c,
		kong.Name("pbmate"),
		kong.Description("TUI companion for Percona Backup for MongoDB."),
		kong.UsageOnError(),
		kong.Vars{"version": version},
	)

	// Resolve config file path.
	configPath := c.Config
	if configPath == "" {
		path, err := config.DefaultPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		configPath = path
	}

	// Load config.
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Bind config and path for dependency injection into Run() methods.
	err = kongCtx.Run(cfg, configFilePath(configPath))
	kongCtx.FatalIfErrorf(err)
}
