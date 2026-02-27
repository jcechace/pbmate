# Contributing

PBMate is a monorepo with two Go modules:

```
pbmate/
├── sdk/              Standalone SDK module (github.com/jcechace/pbmate/sdk/v2)
├── internal/
│   ├── tui/          Terminal UI (BubbleTea)
│   └── config/       App configuration (contexts, settings, XDG paths)
├── main.go           CLI entry point (kong)
├── go.mod            Root module (github.com/jcechace/pbmate)
├── Taskfile.yaml     Build runner config
└── .goreleaser.yaml  Release config
```

The SDK is a separate Go module with its own `go.mod`. The root module (TUI) depends on the SDK via a `replace` directive for local development. Consumers import the SDK independently.

## Prerequisites

- **Go 1.26+**
- **[Task](https://taskfile.dev)** — build runner (`brew install go-task` or see taskfile.dev)
- **[golangci-lint](https://golangci-lint.run) v2** — linting
- A running **MongoDB cluster with PBM configured** (for manual testing)

## Building and Testing

All build, test, and lint operations go through the Taskfile:

```bash
task check      # build + vet + lint + test (ALL modules) — run after every change
task build      # build all modules; produces ./pbmate binary
task test       # run all tests
task lint       # golangci-lint
task fmt        # auto-fix formatting
task --list     # show all available tasks
```

Module-specific tasks are available: `sdk:build`, `sdk:test`, `sdk:check`, `tui:build`, `tui:test`, `tui:check`.

Do not use `go build` or `go test` directly — always use Task.

## Module Dependency Chain

```
TUI  -> SDK -> PBM internals
```

Only the SDK imports PBM packages. The TUI imports only `github.com/jcechace/pbmate/sdk/v2`. PBM types must never appear in the SDK's public API.

## Git Conventions

- One logical change per commit.
- Commit message prefix: `[sdk]`, `[tui]`, or none for general changes.
- Imperative mood summary (~50 chars), optional body after blank line.

## Testing

- Use [testify](https://github.com/stretchr/testify) (`require` for preconditions, `assert` for assertions).
- Prefer table-driven tests with `t.Run` subtests.
- Test function names: `TestFunctionName` or `TestTypeMethod` (no underscores).
- SDK services are interfaces — mock them in consumer tests without MongoDB.

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.26 |
| Testing | testify (require + assert) |
| TUI framework | BubbleTea (charmbracelet) |
| TUI components | bubbles, huh, lipgloss |
| Theming | catppuccin-go |
| Build runner | Taskfile |
| Linting | golangci-lint v2 |
| CLI framework | kong |

## Further Reading

- [SDK README](sdk/README.md) — SDK API documentation and examples
- [docs/agents/](docs/agents/) — detailed architecture docs, conventions, and design decisions
