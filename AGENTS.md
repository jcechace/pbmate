# AGENTS.md - AI Coding Agent Instructions

## Project Overview

PBMate is a TUI companion for PBM (Percona Backup for MongoDB). It provides a
terminal user interface for monitoring and managing MongoDB backups through PBM.

## Workflow

- Never make code changes without first presenting the plan and getting explicit
  approval from the user. This applies regardless of any auto-apply or build
  mode settings. Always discuss before editing.

## Architecture

PBMate consists of three Go modules in a single repository:

```
sdk/  (github.com/jcechace/pbmate/sdk/v2)  - Standalone SDK wrapping PBM internals
mcp/  (github.com/jcechace/pbmate/mcp)  - MCP server, usable standalone or in-TUI
root  (github.com/jcechace/pbmate)      - TUI application (BubbleTea)
```

Dependency chain:

```
TUI  -> SDK -> PBM internals
MCP  -> SDK -> PBM internals
```

Rules:
- Only the SDK imports PBM (`github.com/percona/percona-backup-mongodb`) packages.
- TUI and MCP import ONLY the SDK, never PBM directly.
- The SDK exposes clean domain types; PBM types do not leak to consumers.

## PBM Context

- PBM docs: https://docs.percona.com/percona-backup-mongodb/
- PBM source: https://github.com/percona/percona-backup-mongodb
- Always consult the PBM docs when working with PBM concepts or behavior.
- PBM has its own `sdk/` package -- **DO NOT USE IT**. It is incomplete and
  poorly designed. Our SDK works directly with PBM's internal `pbm/` packages.
- The long-term goal is for `pbmate/sdk` to fully replace PBM's `sdk/` package.
- PBM is MongoDB-centric: uses MongoDB collections for coordination, config,
  locks, command dispatch, agent status, and metadata storage.
- Key PBM concepts: Backups (logical/physical/incremental/external), Restores,
  PITR (point-in-time recovery), Oplog slicing, Storage profiles, Agent
  coordination via distributed locks.
- Key PBM internal packages to depend on:
  - `pbm/connect` -- Client interface for MongoDB access
  - `pbm/ctrl` -- Command dispatch (Cmd, Command types)
  - `pbm/backup` -- Backup metadata and orchestration
  - `pbm/restore` -- Restore metadata and orchestration
  - `pbm/config` -- Configuration and storage profiles
  - `pbm/defs` -- Constants, enums, status codes, backup types
  - `pbm/topo` -- Cluster topology, agent status
  - `pbm/oplog` -- PITR oplog chunk metadata
  - `pbm/lock` -- Distributed lock management
  - `pbm/storage` -- Storage interface and backends (S3, Azure, GCS, filesystem, etc.)
  - `pbm/log` -- MongoDB-stored log access

## SDK Design

The SDK follows the **godo pattern** (DigitalOcean's Go client): a concrete
`Client` struct with interface-typed fields for each domain service.

Services:
- `BackupService` -- list, get, start, and cancel backups
- `RestoreService` -- list, get, and start restores
- `CommandService` -- send commands (backup, restore, cancel) with lock checking
- `ConfigService` -- read configuration and storage profiles
- `ClusterService` -- cluster topology, agents, running operations
- `PITRService` -- PITR status and oplog timelines
- `LogService` -- query and follow PBM logs

Each service has:
- A public interface definition + domain types in `<service>.go`
- An unexported implementation struct in `<service>_impl.go`
- Pure conversion functions (PBM types to SDK types) in `<service>_convert.go`
- Conversion unit tests in `<service>_convert_test.go`

Shared conversion helpers (Timestamp, Status, BackupType, etc.) live in
`convert.go` with tests in `convert_test.go`.

Domain types are owned by the SDK (not aliased from PBM). This isolates
consumers from PBM internal changes and enables testing without MongoDB.

### Enum types

All enum types (Status, BackupType, CompressionType, StorageType, NodeRole,
LogSeverity, ConfigName, CommandType) use **DDD-style value objects**: a struct
with an unexported `value` field, exported singleton instances, and a
`Parse*()` function.

## Tech Stack

- Go 1.26
- Testing: `testify` (require for preconditions, assert for assertions)
- TUI: `bubbletea` (charmbracelet/bubbletea)
- Task runner: Taskfile (taskfile.dev)

## Coding Conventions

- Use `testify` for all test assertions.
  - `require` for preconditions that must hold for the test to continue.
  - `assert` for the actual test assertions.
- Test files use `*_test.go` in the same package.
- Error handling: wrap errors with context using `fmt.Errorf("...: %w", err)`.
- Naming: follow standard Go conventions (exported = PascalCase,
  unexported = camelCase).
- Interfaces describe behavior, not data. Keep them small and focused.
- Use `gofmt` for formatting (not `gofumpt`).
- Under no circumstances should `primitive.Timestamp` or any other
  BSON/MongoDB driver type leak into the SDK's public API. `sdk.Timestamp{T,
  I uint32}` is the clean domain equivalent.
- Prefer PBM's exported internal APIs over direct MongoDB queries. If it means
  fetching a larger data set and filtering in memory, do that -- the data
  volumes (backups, restores, agents) are always small. Direct DB interaction
  is acceptable only when no reasonable exported PBM API exists (currently the
  only exception is command dispatch, since `ctrl.sendCommand` is unexported).

## Project Structure

```
pbmate/
├── .gitignore
├── AGENTS.md               # This file
├── PROGRESS.md             # Progress tracking (keep updated)
├── go.mod                  # TUI module: github.com/jcechace/pbmate
├── Taskfile.yaml           # Task runner config
├── sdk/
│   ├── go.mod              # SDK module: github.com/jcechace/pbmate/sdk/v2
│   ├── client.go           # Client struct, NewClient, Close
│   ├── types.go            # Shared types: Timestamp, Status, BackupType, etc.
│   ├── errors.go           # ErrNotFound, ConcurrentOperationError
│   ├── convert.go          # Shared conversion helpers (Timestamp, Status, etc.)
│   ├── convert_test.go     # Tests for shared conversion helpers
│   ├── backup.go           # BackupService interface + types
│   ├── backup_impl.go      # backupServiceImpl
│   ├── backup_convert.go   # PBM BackupMeta -> SDK Backup conversion
│   ├── backup_convert_test.go
│   ├── restore.go          # RestoreService interface + types
│   ├── restore_impl.go     # restoreServiceImpl
│   ├── restore_convert.go  # PBM RestoreMeta -> SDK Restore conversion
│   ├── restore_convert_test.go
│   ├── command.go          # CommandService interface, Command types
│   ├── command_impl.go     # commandServiceImpl (lock check + dispatch)
│   ├── command_convert.go  # SDK Command -> PBM ctrl.Cmd conversion
│   ├── command_convert_test.go
│   ├── config.go           # ConfigService interface + types
│   ├── config_impl.go      # configServiceImpl
│   ├── config_convert.go   # PBM Config -> SDK Config conversion
│   ├── config_convert_test.go
│   ├── cluster.go          # ClusterService interface + types
│   ├── cluster_impl.go     # clusterServiceImpl
│   ├── cluster_convert.go  # PBM topo/lock -> SDK Agent/Operation conversion
│   ├── cluster_convert_test.go
│   ├── pitr.go             # PITRService interface + types
│   ├── pitr_impl.go        # pitrServiceImpl
│   ├── pitr_convert.go     # PBM oplog.Timeline -> SDK Timeline conversion
│   ├── pitr_convert_test.go
│   ├── log.go              # LogService interface + types
│   ├── log_impl.go         # logServiceImpl
│   ├── log_convert.go      # PBM log.Entry -> SDK LogEntry conversion
│   └── log_convert_test.go
├── mcp/
│   └── go.mod              # MCP module: github.com/jcechace/pbmate/mcp
└── ...                     # TUI source (future)
```

## Testing

- Run SDK tests: `task test` or `go test ./...` from `sdk/`.
- Use `require` for preconditions, `assert` for actual test assertions.
- SDK services are interfaces -- mock them in consumer tests without MongoDB.

## Build & Tasks

- Use `task <name>` (Taskfile) for all build, test, and lint operations.
- Key tasks:
  - `task check` -- build, vet, lint, and test all modules (use after changes)
  - `task build` -- build all modules
  - `task test` -- run all tests
  - `task lint` -- run golangci-lint on all modules
  - `task fmt` -- auto-fix formatting
  - Module-specific tasks are prefixed: `sdk:build`, `sdk:test`, etc.
- Run `task --list` for the full list.
- AI agents must run `task check` after making changes instead of running
  go commands directly. Always consult the user before modifying AGENTS.md.

## Git Practices

- Commit in small increments -- one logical change per commit.
- Commit messages: first line is a brief summary (imperative mood, ~50 chars).
  For complex changes, follow with an empty line and a multiline description.
- Never push to remote. Pushing is always done by the human developer.

## TUI Design

- See `TUI.md` for the TUI design document, layout, and architecture decisions.

## Progress Tracking

- See `PROGRESS.md` for current project status and milestones.
- Keep `PROGRESS.md` updated as work is completed.
