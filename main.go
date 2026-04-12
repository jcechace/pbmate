package main

import (
	"fmt"
	"os"
	"sort"

	tea "charm.land/bubbletea/v2"
	"github.com/alecthomas/kong"

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
	Cfg     cfgCmd           `cmd:"" name:"config" help:"View and modify configuration."`
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
	editor := cfg.ResolveEditor()

	m := tui.New(tui.Options{
		URI:         uri,
		Theme:       theme,
		ContextName: contextName,
		Readonly:    readonly,
		Editor:      editor,
	})
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return err
	}
	if m, ok := result.(tui.Model); ok {
		m.Close()
		if msg := m.ExitMessage(); msg != "" {
			fmt.Println(msg)
		}
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
		return fmt.Errorf("context %q not found; available: %s", cmd.Name, cfg.ContextNames())
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
	if err := config.ValidateURI(cmd.URI); err != nil {
		return err
	}

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

// cfgCmd is the parent for configuration management subcommands.
type cfgCmd struct {
	Show  cfgShowCmd  `cmd:"" help:"Print current configuration."`
	Set   cfgSetCmd   `cmd:"" help:"Set a configuration value."`
	Unset cfgUnsetCmd `cmd:"" help:"Unset a configuration value (reset to default)."`
	Path  cfgPathCmd  `cmd:"" help:"Print config file path."`
}

// cfgShowCmd prints the full config or a single context's settings.
type cfgShowCmd struct {
	Context string `optional:"" help:"Show only this context's settings." name:"context"`
}

func (cmd *cfgShowCmd) Run(cfg *config.AppConfig) error {
	var target any = cfg
	if cmd.Context != "" {
		ctx, ok := cfg.Contexts[cmd.Context]
		if !ok {
			return fmt.Errorf("context %q not found; available: %s", cmd.Context, cfg.ContextNames())
		}
		target = ctx
	}

	out, err := config.FormatYAML(target)
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}

// cfgSetCmd sets a configuration value by key.
type cfgSetCmd struct {
	Key     string `arg:"" help:"Config key (e.g. theme, readonly)."`
	Value   string `arg:"" help:"Value to set."`
	Context string `optional:"" help:"Set on a named context instead of global." name:"context"`
}

func (cmd *cfgSetCmd) Run(cfg *config.AppConfig, path configFilePath) error {
	// Validate URI before persisting.
	if cmd.Key == "uri" {
		if err := config.ValidateURI(cmd.Value); err != nil {
			return err
		}
	}

	err := mutateConfig(cfg, cmd.Context, string(path), func(target any) error {
		return config.SetByPath(target, cmd.Key, cmd.Value)
	})
	if err != nil {
		return err
	}
	fmt.Printf("Set %s = %s\n", cmd.Key, cmd.Value)
	return nil
}

// cfgUnsetCmd resets a configuration value to its default (zero value).
type cfgUnsetCmd struct {
	Key     string `arg:"" help:"Config key to unset (e.g. theme, readonly)."`
	Context string `optional:"" help:"Unset on a named context instead of global." name:"context"`
}

func (cmd *cfgUnsetCmd) Run(cfg *config.AppConfig, path configFilePath) error {
	err := mutateConfig(cfg, cmd.Context, string(path), func(target any) error {
		return config.UnsetByPath(target, cmd.Key)
	})
	if err != nil {
		return err
	}
	fmt.Printf("Unset %s\n", cmd.Key)
	return nil
}

// cfgPathCmd prints the resolved config file path.
type cfgPathCmd struct{}

func (cmd *cfgPathCmd) Run(path configFilePath) error {
	fmt.Println(string(path))
	return nil
}

// mutateConfig applies fn to the appropriate target (the full config or a
// named context), writes the result back, and saves to disk. This is the
// shared helper for config set and config unset.
func mutateConfig(cfg *config.AppConfig, contextName, path string, fn func(any) error) error {
	if contextName != "" {
		ctx, ok := cfg.Contexts[contextName]
		if !ok {
			return fmt.Errorf("context %q not found; available: %s", contextName, cfg.ContextNames())
		}
		if err := fn(&ctx); err != nil {
			return err
		}
		cfg.Contexts[contextName] = ctx
	} else {
		if err := fn(cfg); err != nil {
			return err
		}
	}
	return cfg.Save(path)
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
