# Agent Rules & Workflow

## First Steps

At the start of every session, read `docs/agents/progress.md` to understand
current status, backlog, and deferred items before doing any work.

## Rules

Non-negotiable. Violating any of these is a bug in agent behavior.

1. **Plan before code.** Never make code changes without presenting the plan and getting explicit approval. This applies regardless of auto-apply or build mode settings.
2. **Only the SDK imports PBM.** TUI and MCP import ONLY `github.com/jcechace/pbmate/sdk/v2`, never PBM packages directly.
3. **No PBM types in public API.** `primitive.Timestamp`, BSON types, and any PBM internal types must never appear in SDK public signatures. Use `sdk.Timestamp{T, I uint32}` and other domain types.
4. **Do not use PBM's own `sdk/` package.** It is incomplete. Our SDK works directly with PBM's internal `pbm/` packages.
5. **Run `task check` after every change.** Not `go build` or `go test` directly — use the Taskfile.
6. **Never push to remote.** Pushing is always done by the human developer.
7. **Keep docs updated.** After completing work, update `docs/agents/progress.md`. After learning something non-obvious, add it to `docs/agents/pitfalls.md`.
8. **Ask before modifying agent docs.** Always consult the user before changing CLAUDE.md, AGENTS.md, or any docs/ file that is auto-loaded.
9. **No underscores in test function names.** Use `TestFunctionName` for package-level functions and `TestTypeMethod` (concatenated) for type methods. Never use `TestType_Method` — that pattern is not a Go stdlib convention despite common misconceptions.
10. **Prefer table-driven tests.** Use `[]struct{ ... }` with `t.Run` subtests wherever multiple inputs/expected outputs are being tested against the same logic.
11. **Commit before new work.** Never start a new task if there is uncommitted work in the tree. Commit (or ask the user) first.

## Workflow

Follow this process for every task:

### 1. Understand
- Read `docs/agents/agent-rules.md` (this file) and the relevant conventions doc (`docs/agents/sdk-conventions.md` or `docs/agents/tui-conventions.md`).
- Read `docs/agents/progress.md` to understand current project state.
- Read the specific code files involved. Never propose changes to code you haven't read.
- Check `docs/agents/pitfalls.md` for known issues in the area you're working on.

### 2. Plan
- Present a clear plan: what files will change, what the approach is, what the risks are.
- For SDK changes: identify which `_convert.go`, `_impl.go`, and test files are affected.
- For TUI changes: identify which sub-model, overlay, or data flow is affected. Consult `docs/agents/tui-design.md`.
- Wait for explicit approval before writing code.

### 3. Implement
- Make changes in small, logical increments.
- Follow the coding conventions strictly (see `docs/agents/sdk-conventions.md` or `docs/agents/tui-conventions.md`).
- Write tests alongside implementation, not as an afterthought.

### 4. Verify
- Run `task check` (build + vet + lint + test for all modules).
- If a test fails, fix it. Do not skip or disable tests.
- For TUI changes, describe what the user should visually verify (there are no TUI integration tests).

### 5. Commit & Document
- Commit with scope prefix: `[sdk]`, `[tui]`, or `[mcp]`. General changes (docs, Taskfile) have no prefix.
- Commit message: imperative mood summary (~50 chars), optional body after blank line.
- Update `docs/agents/progress.md`: check off completed items, add new items if scope expanded.
- If you hit a non-obvious bug, add it to `docs/agents/pitfalls.md`.

## Build & Tasks

Use `task <name>` (Taskfile) for all build, test, and lint operations.

```
task check      # build + vet + lint + test (ALL modules) — run after every change
task build      # build all modules
task test       # run all tests
task lint       # golangci-lint
task fmt        # auto-fix formatting
task --list     # show all available tasks
```

Module-specific: `sdk:build`, `sdk:test`, `sdk:check`, `tui:build`, `tui:test`, `tui:check`.

## Git Practices

- One logical change per commit.
- Prefix: `[sdk]`, `[tui]`, `[mcp]`, or none for general changes.
- Imperative mood summary (~50 chars), optional body after blank line.
- Never push to remote.
- Never amend published commits.

## Doc Maintenance

| When | Update |
|------|--------|
| Completed a task | `docs/agents/progress.md` — move item from backlog to completed |
| Discovered a non-obvious bug | `docs/agents/pitfalls.md` — add entry with what/why/fix |
| Added a new SDK service or command | `docs/agents/sdk-conventions.md` — update tables |
| Added a new TUI tab or pattern | `docs/agents/tui-conventions.md` — document pattern |
| Changed TUI layout or keybindings | `docs/agents/tui-design.md` — update spec |
| Changed build or project structure | This file — update Build & Tasks or structure |

## Extending the Project

### Adding a new SDK service

1. Create `<service>.go` — public interface + domain types (no PBM imports).
2. Create `<service>_impl.go` — unexported struct implementing the interface.
3. Create `<service>_convert.go` — PBM -> SDK conversions.
4. Create `<service>_convert_test.go` — conversion tests.
5. Add interface compliance guard: `var _ ServiceName = (*serviceNameImpl)(nil)`.
6. Wire into `client.go` `newMongoClient()`.
7. Update `docs/agents/sdk-conventions.md` Services table.

### Adding a new sealed command variant

1. Add struct in the relevant `<service>.go` with valid-only fields.
2. Implement the sealed interface marker method + `Validate() error`.
3. Add converter in `command_convert.go`.
4. Add type switch case in service `Start`/`Delete`/`Resync` method.
5. Add `Validate()` tests in `command_test.go`.
6. Add conversion test in `command_convert_test.go`.
7. The compiler will catch missing type switch cases via `default: panic("unreachable")`.

### Adding a new CLI command

1. Add command struct with `cmd:""` tag in `main.go` CLI hierarchy.
2. Add `Run(cfg *config.AppConfig) error` method.
3. Wire into parent struct (e.g., add field to `ContextCmd`).
4. Kong auto-discovers it — no registration needed.
5. Update `docs/agents/tui-design.md` CLI section with the new command.

### Adding a new TUI tab

1. Create `<tab>.go` with a sub-model struct (not `tea.Model`).
2. Implement `update(msg tea.KeyMsg, keys globalKeyMap) tea.Cmd`, `view(w, h int) string`, `resize(w, h int)`, `setData(...)`.
3. Add data message type in `data.go` and fetch command.
4. Wire into `app.go`: tab constant, Init, Update routing, View routing.
5. Add keybindings in `keys.go`.
6. Update `docs/agents/tui-design.md` with the new tab spec.

## Related Docs

| Document | Purpose | When to read |
|----------|---------|-------------|
| `docs/agents/sdk-conventions.md` | SDK architecture, patterns, PBM context | Working on SDK code |
| `docs/agents/tui-conventions.md` | TUI architecture, BubbleTea patterns | Working on TUI code |
| `docs/agents/pitfalls.md` | Known bugs and lessons learned | Before working on any area |
| `docs/agents/progress.md` | Work tracking, backlog, completed | Start of any task |
| `docs/agents/tui-design.md` | TUI layout spec, message flow, keybindings | Changing TUI behavior |
| `sdk/README.md` | SDK API docs and examples | Understanding SDK public API |
