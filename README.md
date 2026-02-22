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
pbmate --uri <mongodb-uri> [--theme <theme>]
```

| Flag | Description | Default |
|---|---|---|
| `--uri` | MongoDB connection URI (required) | — |
| `--theme` | Color theme: `default`, `mocha`, `latte`, `frappe`, `macchiato` | `default` |

### Keybindings

| Key | Action |
|---|---|
| `1` `2` `3` | Switch tabs |
| `]` `[` | Cycle panel focus |
| `j`/`k` or arrows | Navigate lists |
| `s` | Start backup (quick) |
| `S` | Start backup (configure) |
| `c` | Cancel running backup |
| `d` | Delete selected backup |
| `e` | Apply YAML configuration |
| `p` | New storage profile |
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
bk, _ := client.Backups.Wait(ctx, result.Name, sdk.BackupWaitOptions{})
```

## Project Structure

```
pbmate/
  sdk/              Standalone SDK module (github.com/jcechace/pbmate/sdk/v2)
  internal/tui/     Terminal UI (BubbleTea)
  main.go           TUI entry point
```

The SDK is a separate Go module with its own `go.mod`. The TUI depends on the SDK via a `replace` directive for local development. Consumers import the SDK independently — no dependency on the TUI.

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
