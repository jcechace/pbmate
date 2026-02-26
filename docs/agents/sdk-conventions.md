# SDK Conventions

## Module Layout

PBMate is a monorepo with three Go modules:

```
sdk/  (github.com/jcechace/pbmate/sdk/v2)  — Standalone SDK wrapping PBM internals
mcp/  (github.com/jcechace/pbmate/mcp)     — MCP server (placeholder)
root  (github.com/jcechace/pbmate)          — TUI application (BubbleTea)
```

Dependency chain (strict — no cycles, no skipping):

```
TUI  -> SDK -> PBM internals
MCP  -> SDK -> PBM internals
```

The SDK exposes clean domain types. PBM types do not leak to consumers.

The root module uses a `replace` directive (`replace github.com/jcechace/pbmate/sdk/v2 => ./sdk`) for local development. Agents doing dependency work should be aware of this.

## File Convention

Every service follows this pattern:

| File | Contents | PBM imports? |
|------|----------|-------------|
| `<service>.go` | Public interface + domain types | No |
| `<service>_impl.go` | Unexported implementation struct | Yes |
| `<service>_convert.go` | PBM -> SDK type conversion | Yes |
| `<service>_convert_test.go` | Conversion unit tests | Yes |

Shared helpers: `convert.go` / `convert_test.go`, `wait.go` / `wait_test.go`.

## Services

| Service | Interface | Key Methods |
|---------|-----------|-------------|
| BackupService | `backup.go` | List, Get, Start, Wait, Cancel, Delete, CanDelete |
| RestoreService | `restore.go` | List, Get, Start, Wait |
| ConfigService | `config.go` | Get, GetYAML, ListProfiles, GetProfile, SetProfile, Resync |
| ClusterService | `cluster.go` | Members, Agents, RunningOperations, CheckLock, ServerInfo |
| PITRService | `pitr.go` | Status, Timelines, Bases, Enable, Disable, Delete |
| LogService | `log.go` | Get, Follow |

## Sealed Command Pattern

Operations with distinct variants use sealed interfaces (unexported marker method). Each variant is a concrete struct with only the fields valid for that variant.

| Sealed Interface | Variants |
|-----------------|----------|
| `StartBackupCommand` | `StartLogicalBackup`, `StartPhysicalBackup`, `StartIncrementalBackup` |
| `StartRestoreCommand` | `StartSnapshotRestore`, `StartPITRRestore` |
| `DeleteBackupCommand` | `DeleteBackupByName`, `DeleteBackupsBefore` |
| `DeletePITRCommand` | `DeletePITRBefore`, `DeletePITRAll` |
| `ResyncCommand` | `ResyncMain`, `ResyncProfile`, `ResyncAllProfiles` |

All commands implement `Validate() error`. Service methods call Validate before lock checks or dispatch. Type switches in `Start`/`Delete`/`Resync` methods use `default: panic("unreachable: ...")` to catch missing branches at development time.

Standalone commands (not part of any sealed interface): `AddProfileCommand`, `RemoveProfileCommand`, `CancelBackupCommand`. These are simple single-variant types with their own `Validate()` methods.

## Enum Types (Value Objects)

All enums use DDD-style value objects: a struct with an unexported `value` field, exported singleton instances, and a `Parse*()` function.

Types: `Status`, `BackupType`, `CompressionType`, `StorageType`, `NodeRole`, `LogSeverity`, `ConfigName`, `CommandType`.

```go
// Pattern:
type BackupType struct{ value string }
var BackupTypeLogical = BackupType{value: "logical"}
func ParseBackupType(s string) (BackupType, error) { ... }
```

External code cannot construct invalid values (`BackupType{value: "garbage"}` doesn't compile — `value` is unexported).

## PBM Context

- PBM docs: https://docs.percona.com/percona-backup-mongodb/
- PBM source: https://github.com/percona/percona-backup-mongodb
- Always consult PBM docs when working with PBM concepts.
- To inspect PBM source locally, read from the Go module cache (`~/go/pkg/mod/github.com/percona/percona-backup-mongodb@.../`). Check `sdk/go.mod` for the pinned version. Do not modify files in the module cache.
- PBM is MongoDB-centric: uses MongoDB collections for coordination, config, locks, command dispatch, agent status, and metadata.

Key PBM internal packages:

| Package | Purpose |
|---------|---------|
| `pbm/connect` | MongoDB client interface |
| `pbm/ctrl` | Command dispatch |
| `pbm/backup` | Backup metadata and orchestration |
| `pbm/restore` | Restore metadata and orchestration |
| `pbm/config` | Configuration and storage profiles |
| `pbm/defs` | Constants, enums, status codes |
| `pbm/topo` | Cluster topology, agent status |
| `pbm/oplog` | PITR oplog chunk metadata |
| `pbm/lock` | Distributed lock management |
| `pbm/storage` | Storage backends (S3, Azure, GCS, filesystem) |
| `pbm/log` | MongoDB-stored log access |

## Coding Conventions

### Go Style
- `gofmt` for formatting (not `gofumpt`).
- Naming: standard Go conventions (exported = PascalCase, unexported = camelCase).
- Interfaces describe behavior, not data. Keep them small and focused.
- Error handling: wrap with context using `fmt.Errorf("verb noun: %w", err)`.
- No magic constants. All literals that control behavior must be named constants.

### SDK-Specific
- **Design for arbitrary consumers.** The SDK is a standalone library — not just a backend for PBMate's TUI. API decisions (types, method signatures, convenience functions) must make sense for any Go program importing the SDK.
- Prefer PBM's exported internal APIs over direct MongoDB queries. If it means fetching more data and filtering in memory, do that — data volumes (backups, restores, agents) are always small.
- Direct DB interaction only when no reasonable PBM API exists (currently: command dispatch, since `ctrl.sendCommand` is unexported).
- Mark PBM workarounds with `TODO(pbm-fix)`.
- Unknown PBM enum values in conversions: log `slog.Warn`, do not crash. The SDK pins to a specific PBM version; unknown enums appear only on version mismatch.
- `nil`-means-default for optional fields: `*int` and `*bool` for tuning knobs.
- Interface compliance guards on every impl: `var _ BackupService = (*backupServiceImpl)(nil)`.

### Testing
- Use `testify` for all assertions.
  - `require` — preconditions that must hold for the test to continue.
  - `assert` — actual test assertions.
- Test files: `*_test.go` in the same package (white-box — `package sdk`, not `package sdk_test`).
- SDK services are interfaces — mock them in consumer tests without MongoDB.

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.26 |
| Testing | `testify` (require + assert) |
| TUI framework | `bubbletea` (charmbracelet) |
| TUI components | `bubbles`, `huh`, `lipgloss` |
| Syntax highlighting | `chroma/v2` (YAML in config tab) |
| Theming | `catppuccin-go` |
| Build runner | Taskfile (taskfile.dev) |
| Linting | `golangci-lint` v2 (config format differs from v1) |

## Project Structure

```
pbmate/
├── sdk/
│   ├── go.mod              # SDK module: github.com/jcechace/pbmate/sdk/v2
│   ├── README.md           # SDK API docs and examples
│   ├── doc.go              # Package-level documentation
│   ├── client.go           # NewClient, Close, functional options
│   ├── types.go            # Shared types: Timestamp, Status, BackupType, etc.
│   ├── errors.go           # ErrNotFound, ConcurrentOperationError, etc.
│   ├── convert.go          # Shared conversion helpers
│   ├── wait.go             # Generic waitForTerminal helper
│   ├── backup.go           # BackupService interface + types
│   ├── backup_impl.go      # Implementation (incl. CanDelete)
│   ├── backup_convert.go   # PBM -> SDK conversion
│   ├── backup_chain.go     # BackupChain grouping utilities
│   ├── restore.go          # RestoreService interface + types
│   ├── restore_impl.go     # Implementation
│   ├── restore_convert.go  # PBM -> SDK conversion
│   ├── command.go          # Sealed command types + Validate()
│   ├── command_impl.go     # Internal command dispatch
│   ├── command_convert.go  # SDK -> PBM command conversion
│   ├── config.go           # ConfigService interface + types
│   ├── config_impl.go      # Implementation
│   ├── config_convert.go   # PBM -> SDK conversion
│   ├── cluster.go          # ClusterService interface + types
│   ├── cluster_impl.go     # Implementation
│   ├── cluster_convert.go  # PBM -> SDK conversion
│   ├── pitr.go             # PITRService interface + types
│   ├── pitr_impl.go        # Implementation
│   ├── pitr_convert.go     # PBM -> SDK conversion
│   ├── log.go              # LogService interface + types
│   ├── log_impl.go         # Implementation
│   ├── log_convert.go      # PBM -> SDK conversion
│   └── cmd/smoketest/      # Manual smoke test binary
├── internal/
│   ├── config/             # App config: XDG path, Load/Save, context resolution, field helpers
│   └── tui/                # TUI implementation (see docs/tui-conventions.md)
├── mcp/
│   └── go.mod              # MCP module placeholder
├── main.go                 # CLI entry point (kong): TUI + context + config commands
├── go.mod                  # Root module: github.com/jcechace/pbmate
└── Taskfile.yaml           # Build runner config
```
