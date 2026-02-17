# AGENTS.md - AI Coding Agent Instructions

## Project Overview

PBMate is a TUI companion for PBM (Percona Backup for MongoDB). It provides a
terminal user interface for monitoring and managing MongoDB backups through PBM.

## Architecture

PBMate consists of three Go modules in a single repository:

```
sdk/  (github.com/jcechace/pbmate/sdk)  - Standalone SDK wrapping PBM internals
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

Services (v1 is read-only):
- `BackupService` -- list and describe backups
- `RestoreService` -- list and describe restores
- `ConfigService` -- read configuration and storage profiles
- `ClusterService` -- cluster topology, agents, running operations
- `PITRService` -- PITR status and oplog timelines
- `LogService` -- query and follow PBM logs

Each service has:
- A public interface definition + domain types in `<service>.go`
- An unexported implementation struct in `<service>_impl.go`
- Conversion functions from PBM internal types to SDK domain types

Domain types are owned by the SDK (not aliased from PBM). This isolates
consumers from PBM internal changes and enables testing without MongoDB.

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

## Project Structure

```
pbmate/
├── .gitignore
├── AGENTS.md               # This file
├── PROGRESS.md             # Progress tracking (keep updated)
├── go.mod                  # TUI module: github.com/jcechace/pbmate
├── Taskfile.yaml           # Task runner config
├── sdk/
│   ├── go.mod              # SDK module: github.com/jcechace/pbmate/sdk
│   ├── client.go           # Client struct, NewClient, Close
│   ├── types.go            # Shared types: Timestamp, Status, BackupType, etc.
│   ├── backup.go           # BackupService interface + types
│   ├── backup_impl.go      # backupServiceImpl + conversion
│   ├── restore.go          # RestoreService interface + types
│   ├── restore_impl.go     # restoreServiceImpl + conversion
│   ├── config.go           # ConfigService interface + types
│   ├── config_impl.go      # configServiceImpl + conversion
│   ├── cluster.go          # ClusterService interface + types
│   ├── cluster_impl.go     # clusterServiceImpl + conversion
│   ├── pitr.go             # PITRService interface + types
│   ├── pitr_impl.go        # pitrServiceImpl + conversion
│   ├── log.go              # LogService interface + types
│   └── log_impl.go         # logServiceImpl + conversion
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
- See `Taskfile.yaml` for available tasks.

## Git Practices

- Commit in small increments -- one logical change per commit.
- Commit messages: first line is a brief summary (imperative mood, ~50 chars).
  For complex changes, follow with an empty line and a multiline description.
- Never push to remote. Pushing is always done by the human developer.

## Progress Tracking

- See `PROGRESS.md` for current project status and milestones.
- Keep `PROGRESS.md` updated as work is completed.
