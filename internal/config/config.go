// Package config handles PBMate application configuration: loading, saving,
// XDG path resolution, and connection context management.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// appDirName is the directory name under XDG_CONFIG_HOME.
const appDirName = "pbmate"

// configFileName is the configuration file name.
const configFileName = "config.yaml"

// AppConfig is the root configuration for PBMate. It holds global TUI
// settings and named connection contexts.
type AppConfig struct {
	// Theme is the global color theme name (e.g. "default", "mocha").
	Theme string `yaml:"theme,omitempty"`

	// Readonly disables all mutation actions in the TUI when true.
	Readonly bool `yaml:"readonly,omitempty"`

	// CurrentContext is the name of the active connection context.
	CurrentContext string `yaml:"current-context,omitempty"`

	// Contexts maps context names to their connection settings.
	Contexts map[string]Context `yaml:"contexts,omitempty"`
}

// Context holds the connection settings for a named cluster.
type Context struct {
	// URI is the MongoDB connection URI (required).
	URI string `yaml:"uri"`

	// Theme overrides the global theme for this context. Empty means
	// inherit the global theme.
	Theme string `yaml:"theme,omitempty"`

	// Readonly overrides the global readonly setting for this context.
	// nil means inherit the global setting.
	Readonly *bool `yaml:"readonly,omitempty"`
}

// DefaultPath returns the default configuration file path following XDG
// conventions: $XDG_CONFIG_HOME/pbmate/config.yaml, falling back to
// ~/.config/pbmate/config.yaml.
func DefaultPath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, appDirName, configFileName), nil
}

// Load reads and parses the configuration file at the given path. If the
// file does not exist, an empty AppConfig is returned (not an error).
func Load(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &AppConfig{}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return &cfg, nil
}

// Save writes the configuration to the given path, creating parent
// directories as needed.
func (c *AppConfig) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// CurrentCtx returns the context for CurrentContext. Returns nil if
// CurrentContext is empty or not found.
func (c *AppConfig) CurrentCtx() *Context {
	if c.CurrentContext == "" {
		return nil
	}
	ctx, ok := c.Contexts[c.CurrentContext]
	if !ok {
		return nil
	}
	return &ctx
}

// ResolveURI returns the effective MongoDB URI by applying flag precedence:
// explicit URI flag > named context > current context. Returns an error if
// no URI can be resolved.
func (c *AppConfig) ResolveURI(flagURI, flagContext string) (string, error) {
	// Explicit --uri always wins.
	if flagURI != "" {
		return flagURI, nil
	}

	// --context flag overrides current-context.
	contextName := flagContext
	if contextName == "" {
		contextName = c.CurrentContext
	}

	if contextName == "" {
		return "", fmt.Errorf("no URI provided and no active context; use --uri or run: pbmate context add <name> --uri=<uri>")
	}

	ctx, ok := c.Contexts[contextName]
	if !ok {
		return "", fmt.Errorf("context %q not found; available contexts: %s", contextName, c.contextNames())
	}

	if ctx.URI == "" {
		return "", fmt.Errorf("context %q has no URI configured", contextName)
	}
	return ctx.URI, nil
}

// ResolveTheme returns the effective theme by applying flag precedence:
// explicit flag > context override > global config > "default".
func (c *AppConfig) ResolveTheme(flagTheme, flagContext string) string {
	if flagTheme != "" {
		return flagTheme
	}

	// Check context-level override.
	contextName := flagContext
	if contextName == "" {
		contextName = c.CurrentContext
	}
	if contextName != "" {
		if ctx, ok := c.Contexts[contextName]; ok && ctx.Theme != "" {
			return ctx.Theme
		}
	}

	if c.Theme != "" {
		return c.Theme
	}
	return "default"
}

// ResolveReadonly returns the effective readonly setting by applying flag
// precedence: explicit flag > context override > global config > false.
func (c *AppConfig) ResolveReadonly(flagReadonly *bool, flagContext string) bool {
	if flagReadonly != nil {
		return *flagReadonly
	}

	// Check context-level override.
	contextName := flagContext
	if contextName == "" {
		contextName = c.CurrentContext
	}
	if contextName != "" {
		if ctx, ok := c.Contexts[contextName]; ok && ctx.Readonly != nil {
			return *ctx.Readonly
		}
	}

	return c.Readonly
}

// contextNames returns a comma-separated list of context names for error
// messages. Returns "(none)" if no contexts are defined.
func (c *AppConfig) contextNames() string {
	if len(c.Contexts) == 0 {
		return "(none)"
	}
	names := ""
	for name := range c.Contexts {
		if names != "" {
			names += ", "
		}
		names += name
	}
	return names
}
