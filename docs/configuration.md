# Configuration

PBMate stores its configuration in a single YAML file and supports named
connection contexts for switching between clusters.

## Config File Location

```
$XDG_CONFIG_HOME/pbmate/config.yaml    # if XDG_CONFIG_HOME is set
~/.config/pbmate/config.yaml           # fallback (XDG default)
```

Override the path with `--config <path>` or the `PBMATE_CONFIG` environment
variable. To see the resolved path:

```bash
pbmate config path
```

If the file doesn't exist, PBMate starts with empty defaults.

## File Format

```yaml
theme: mocha
readonly: false
editor: vim

current-context: production

contexts:
  production:
    uri: mongodb://prod-host:27017
    readonly: true
  staging:
    uri: mongodb://staging-host:27017
    theme: latte
  local:
    uri: mongodb://localhost:27017
```

### Global Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `theme` | string | `"default"` | Color theme for the TUI |
| `readonly` | bool | `false` | Disable all mutation actions |
| `editor` | string | — | External editor command (e.g. `"vim"`, `"code -w"`) |
| `current-context` | string | — | Name of the active connection context |
| `contexts` | map | — | Named connection contexts (see below) |

### Context Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `uri` | string | Yes | MongoDB connection URI |
| `theme` | string | No | Theme override for this context (empty = inherit global) |
| `readonly` | *bool | No | Readonly override (omit to inherit global setting) |

## Contexts

Contexts are named connection profiles. Each stores a MongoDB URI and optional
per-context overrides for theme and readonly mode.

### Managing Contexts

```bash
# Add a new context
pbmate context add prod --uri mongodb://prod-host:27017
pbmate context add prod --uri mongodb://prod-host:27017 --theme mocha --readonly

# List all contexts (* marks the active one)
pbmate context list

# Switch active context
pbmate context use prod

# Show current context name and URI
pbmate context current

# Remove a context
pbmate context remove staging
```

### Using Contexts

```bash
# Launch TUI with the active context
pbmate

# One-time override (doesn't change config)
pbmate --context staging

# Direct URI bypasses contexts entirely
pbmate --uri mongodb://localhost:27017
```

## Flag Precedence

Settings are resolved in this order (first match wins):

```
CLI flag  >  context setting  >  global config  >  built-in default
```

Examples:

- **Theme**: `--theme mocha` > `contexts.prod.theme` > top-level `theme` > `"default"`
- **Readonly**: `--readonly` > `contexts.prod.readonly` > top-level `readonly` > `false`

If no URI is available (no `--uri`, no context, no `current-context`), PBMate
prints a help message directing you to `pbmate context add`.

## Themes

PBMate supports [Catppuccin](https://catppuccin.com/) color themes:

| Theme | Description |
|-------|-------------|
| `default` | Adaptive — adjusts to your terminal's light/dark mode |
| `mocha` | Dark, warm (Catppuccin Mocha) |
| `latte` | Light (Catppuccin Latte) |
| `frappe` | Dark, cool (Catppuccin Frappe) |
| `macchiato` | Dark, medium (Catppuccin Macchiato) |

Set globally, per-context, or as a CLI flag:

```bash
pbmate config set theme mocha
pbmate config set theme latte --context=prod
pbmate --theme frappe
```

## Readonly Mode

When readonly is active, all mutation actions are disabled: starting backups,
restoring, cancelling, deleting, modifying config, resyncing, and toggling PITR.
The TUI operates as a monitoring-only dashboard.

A bold yellow `READONLY` badge appears in the bottom status bar. The help
overlay (`?`) hides mutation keybindings in readonly mode.

Enable it globally, per-context, or as a CLI flag:

```bash
pbmate config set readonly true
pbmate config set readonly true --context=prod
pbmate --readonly
```

## Editor

PBMate can open PBM configuration in an external editor (press `e` on the
Config tab). The editor is resolved in this order:

1. `PBMATE_EDITOR` environment variable
2. `editor` field in the config file
3. `VISUAL` environment variable
4. `EDITOR` environment variable
5. `vi` (fallback)

Compound commands with flags are supported (e.g. `"code -w"`, `"nvim --clean"`).

```bash
# Set via config
pbmate config set editor "code -w"

# Or via environment
export PBMATE_EDITOR="code -w"
```

The editor flow works like `kubectl edit`: PBMate writes the current YAML to a
temp file, opens the editor, and applies changes when the editor exits. If the
YAML is invalid, the temp file is preserved and its path is shown in the error
message for recovery.

## CLI Config Commands

The `pbmate config` subcommand manages configuration settings without manually
editing the YAML file.

```bash
# Print the full config as YAML
pbmate config show

# Print a single context
pbmate config show --context=prod

# Set a global value
pbmate config set theme mocha
pbmate config set readonly true

# Set a per-context value
pbmate config set theme latte --context=staging
pbmate config set readonly true --context=prod

# Reset a value to its default
pbmate config unset theme
pbmate config unset readonly --context=prod

# Print the config file path
pbmate config path
```

Supported keys for `set`/`unset`: `theme`, `readonly`, `editor`,
`current-context`.
Per-context keys (with `--context`): `uri`, `theme`, `readonly`.
