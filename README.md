# PBMate

A companion toolkit for [Percona Backup for MongoDB (PBM)](https://github.com/percona/percona-backup-mongodb).

PBMate provides a **Go SDK** with stable public interfaces and a **terminal UI** for monitoring and managing PBM clusters. The SDK wraps PBM's internal packages behind a conversion boundary, so consumers are insulated from internal changes.

## Why

PBM is a distributed backup tool for MongoDB. The SDK provides a stable programmatic interface for PBM operations — clean domain types, service interfaces, and a conversion layer that decouples consumers from PBM's internal package structure.

PBMate provides:

- **SDK** — a stable Go API with domain-typed services, sealed command interfaces, and a conversion layer that absorbs internal changes
- **TUI** — a real-time terminal dashboard for monitoring and operating PBM clusters

## TUI

The terminal UI provides a real-time dashboard with three tabs:

- **Overview** — cluster topology, agent health, PITR status, running operations, live logs
- **Backups** — backup list grouped by storage profile, incremental chain display, restore history
- **Config** — configuration viewer, storage profile management, YAML editor

![Overview tab](docs/screens/overview.png)

### Usage

```
pbmate                              # start TUI with current context
pbmate --uri <mongodb-uri>          # start TUI with explicit URI
pbmate --theme mocha --readonly     # theme and readonly overrides
```

| Flag | Description | Default |
|---|---|---|
| `--uri` | MongoDB connection URI (overrides context) | — |
| `--context` | Use a named context (overrides current-context) | — |
| `--theme` | Color theme: `default`, `mocha`, `latte`, `frappe`, `macchiato` | `default` |
| `--readonly` | Disable mutation actions in the TUI | `false` |
| `--config` | Config file path (or set `PBMATE_CONFIG`) | `~/.config/pbmate/config.yaml` |

### Context Management

Named connection contexts avoid repeating URIs:

```
pbmate context add prod --uri=mongodb://prod:27017 --theme=mocha
pbmate context use prod
pbmate context list
```

### Configuration

View and modify settings from the command line:

```
pbmate config show                          # print full config as YAML
pbmate config set theme mocha               # set global theme
pbmate config set readonly true --context=prod  # per-context override
pbmate config unset readonly --context=prod # remove override (inherit global)
pbmate config path                          # print config file path
```

### Keybindings

| Key | Action |
|---|---|
| `1` `2` `3` | Switch tabs |
| `]` `[` | Cycle panel focus |
| `j`/`k` or arrows | Navigate lists |
| `s` / `S` | Start backup (quick / configure) |
| `X` | Cancel running backup |
| `d` | Delete selected item (backup or profile) |
| `R` / `r` | Restore (wizard / from cursor) |
| `C` / `c` | Set config (wizard / on Config tab) |
| `R` / `r` | Resync (on Config tab) |
| `x` | Delete profile (Config tab) |
| `f` | Toggle log follow mode |
| `w` | Toggle log word wrap |
| `tab` | Toggle backups / restores |
| `esc` | Back / dismiss overlay |
| `?` | Help overlay |
| `q` | Quit |

## SDK

The SDK is a standalone Go module for programmatic access to PBM operations. See the [SDK README](sdk/README.md) for API documentation, examples, and design details.

```go
import sdk "github.com/jcechace/pbmate/sdk/v2"

client, err := sdk.NewClient(ctx, sdk.WithMongoURI("mongodb://localhost:27017"))
defer client.Close(ctx)

// List recent backups.
backups, _ := client.Backups.List(ctx, sdk.ListBackupsOptions{Limit: 5})

// Start a logical backup.
result, _ := client.Backups.Start(ctx, sdk.StartLogicalBackup{})

// Wait for completion.
bk, _ := result.Wait(ctx, sdk.BackupWaitOptions{})
```

## Project Structure

```
pbmate/
  sdk/              Standalone SDK module (github.com/jcechace/pbmate/sdk/v2)
  internal/tui/     Terminal UI (BubbleTea)
  internal/config/  App configuration (Load/Save, contexts, field helpers)
  main.go           CLI entry point (kong): TUI + context + config commands
```

The SDK is a separate Go module with its own `go.mod`. The TUI depends on the SDK via a `replace` directive for local development. Consumers import the SDK independently — no dependency on the TUI.

## Installation

### Homebrew (macOS / Linux)

```bash
brew tap jcechace/tap
brew install pbmate
```

### Binary Download

Download pre-built binaries from the [GitHub Releases](https://github.com/jcechace/pbmate/releases) page.

### From Source

```bash
go install github.com/jcechace/pbmate@latest
```

## Building

PBMate uses [Task](https://taskfile.dev) as its build runner.

```bash
task build    # build all modules; produces ./pbmate binary
task test     # run all tests
task check    # build + vet + lint + test
```

## Requirements

- Go 1.26+
- A running MongoDB cluster with PBM configured
- Network access to the MongoDB cluster from the machine running PBMate
