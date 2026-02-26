# Progress

## Current Status

SDK wraps the core PBM operations (backup, restore, config, cluster, PITR, logs) with gaps remaining (see Deferred Features). TUI has 3 tabs (Overview, Backups, Config) with form overlays, incremental chain support, and context-sensitive restores. See Backlog for planned features. MCP module is a placeholder.

## In Progress

(none)

## Backlog

Prioritized next items:
- [x] Connection reconnect on failure (auto-retry with exponential backoff)
- [x] Refine error messages (follow context canceled, double prefixes, config ErrNotFound, connect verbosity)
- [x] CI/CD: GitHub Actions (test/lint/vulncheck), Dependabot, GoReleaser, `--version` flag
- [ ] `/` filter in list views
- [ ] MCP server implementation (Phase 4 — scope TBD)
- [ ] Homebrew tap for binary distribution

Deferred (add when needed):
- [ ] `/` filter in list views (backups, maybe logs search)
- [ ] Codecov integration for coverage tracking

## Completed Milestones

### Phase 1: Project Setup
Go 1.26 modules, .gitignore, AGENTS.md, PROGRESS.md, MCP module bootstrap.

### Phase 2: SDK Scaffolding
Client struct with functional options. All 6 service interfaces (Backup, Restore, Config, Cluster, PITR, Log) with stubs. PBM dependency via replace directive. Enum types refactored to DDD-style value objects. ConfigName normalization. CommandService with sealed command interface. Write operations wired through services.

### Phase 3: SDK Implementation
Foundation (errors, shared conversions, CommandType enum). All 6 services fully implemented: BackupService with custom MongoDB queries, RestoreService via PBM functions, PITRService with status aggregation, ClusterService with topology and agent status, ConfigService with profiles, LogService with tailable cursor follow. Command dispatch via lock checking + MongoDB insert.

### Phase 4: MCP Server
TBD — placeholder module exists.

### Phase 5a: TUI Initial Scaffold
TUI.md design doc. App skeleton with tab navigation. Catppuccin theming. Adaptive polling (10s/2s). Overview tab with agent tree. Backups tab with list/detail. Backup actions (start, cancel, delete). Shared rendering helpers.

### Phase 5b: TUI Redesign
Research of lazydocker, k9s, gh-dash, etc. 4-quadrant Overview layout. Dropped Logs tab (4→3 tabs). Merged bottom bars into single status HUD + hints bar. Collapsible RS groups with status indicators. Status panel (PITR, op, latest backup, storage). Log panel with follow mode. Stable cursor by identity. Context-sensitive action hints. All panels migrated to viewport components. Fixed panel overflow. 4-quadrant focus cycling.

### Phase 5b+: Architecture Refactoring
Layout helpers extracted. Sub-models made self-contained (backupsModel, overviewModel). logPanel extracted as reusable component. clusterPanel extracted. Data duplication eliminated between root and sub-models. Panel titles in borders.

### Phase 5c: Interactions
huh form overlays (quick/full backup, confirm dialogs). Help overlay (`?`). Incremental backup chain grouping with base/children display. Chain-aware delete. Restore list with tab toggle. SDK documentation enrichment.

### Phase 5c+: Code Quality
Bug fixes (PITR truncation, stale cursor, error clearing). Chain logic extracted with tests. Cursor rendering deduplicated. Help overlay derived from keybinding definitions. Consistent `*Styles` passing.

### SDK Domain Enrichment
Domain methods on Backup/Restore (IsIncremental, Duration, etc.). Severity filtering for logs. BackupChain type with SDK utilities. Package-level documentation. Goroutine leak fix in LogService.Follow.

### Code Quality Polish
SDK domain methods used at all TUI call sites. waitForTerminal generic helper. Standardized error message prefixes. Profile name in delete dialog.

### Phase 5d: Config Tab
Config tab with main config + profiles + YAML syntax highlighting (Chroma). File picker overlay for config apply. Profile name form. formOverlay interface extracted.

### SDK Hardening
CanDelete with ErrNotChainBase. Validate() on all command types. Shared validation helpers. UsersAndRoles field. Comprehensive Validate() tests.

### TUI Audit Fixes
Expanded config detail. Oplog range/sizes in backup detail. Parallel collections in backup wizard. CanDelete pre-check. Context-sensitive restore forms (snapshot + PITR). Flash error persistence fix. Backup name display. Auto-switch to Restores list. Config apply nil handling.

### SDK Code Quality
slog.Warn for unknown enums. Generic convertSlice. Unified Limit type. Removed duplicate conversion. Client.Close error wrapping. TODO(pbm-fix) markers. MarshalText/UnmarshalText round-trip tests.

### TUI Code Quality
formOverlay interface. Chroma style mapped to theme. defaultConfigName constant. Form overlay width rename. Magic number documentation. Root context threading. Render helper unit tests.

### Theming Fixes
Per-flavor huh themes built from catppuccin-go instead of adaptive ThemeCatppuccin().

### Form Redesign
Flat single-screen forms (no wizard pages). Inline selectors for 2-3 options. Adaptive overlay width (50% terminal, 40-60 clamped). PITR presets (Latest, -5m, -15m, -30m, -1h, -6h, Custom). Backup context headers in restore forms. Dynamic form rebuild on value changes (LayoutStack, no WithHideFunc). Resync form with target selector (Main/Profile), conditional options (include restores, clear metadata). Delete profile with confirm overlay.

### Set Config Wizard + Keybinding Rework
3-step set-config wizard (target form → file picker → optional override confirm). Cancel backup remapped to `X`. Config tab keybindings: `C`/`c` set config, `R`/`r` resync, `x` delete profile (replaces old `e`/`p`). Two-column help overlay with tab-specific sections. Combined help entries (`s / S`, `C / c`, `R / r`).

### Restore Wizard
Two-step restore wizard: Step 1 (target selection — type, profile filter, backup/PITR target), Step 2 (options — scope, tuning, confirm). `R` opens generic wizard (Step 1 → Step 2), `r` skips Step 1 and opens from cursor (same as previous behavior). Consistent `R`/`r` pattern matching `C`/`c` and `s`/`S`. Profile inline selector in Snapshot mode filters backup list by storage profile.

### Code Quality Refactoring
Overlay helpers extracted (`dismissOverlay`, `updateFormModel`, `initFormWithAdvance` in `backup_form.go`) — all 7 overlay types use them. Four action message types unified into single `actionResultMsg`. Shared layout helper `panelBorderColor` in `layout.go`. `firstErrCollector` type for concurrent fetch error coalescing. Form-to-command test coverage (4 new test files: `backup_form_test.go`, `restore_form_test.go`, `resync_form_test.go`, `config_form_test.go`).

### TUI Visual Polish
Config tab list redesigned: two-column layout (name + type), path dropped, bold-on-select for Main (star removed), muted "── Profiles ──" section label. Backup list separator "── Backups ──" between PITR timelines and backup profiles. Vertical padding added to all panels (`Padding(1,1)`). Global `d` delete key (unified across Backups and Config tabs).

### CLI & Configuration
Kong-based CLI with `pbmate tui` as default command. XDG-compliant config file (`$XDG_CONFIG_HOME/pbmate/config.yaml`). Named connection contexts with URI + optional theme/readonly overrides. Flag precedence: CLI flag > context setting > global config > default. Context management subcommands: list, current, use, add, remove. `internal/config` package with Load/Save/Resolve methods and full test coverage. `tui.New()` accepts `Options` struct for stable extensibility. Active context name displayed in TUI header bar (muted style, hidden for direct `--uri`).

### Readonly Mode
`--readonly` TUI enforcement. All 11 mutation keys guarded (`s`, `S`, `X`, `d`, `R`, `r` on Backups, `C`, `c`, `d`, `R`, `r` on Config). Bold yellow `READONLY` badge in bottom bar status zone. Help overlay filters out mutation entries in readonly mode. Resolved from CLI flag > context override > global config > false.

### Connection Reconnect
Auto-retry with exponential backoff (2s, 4s, 8s, 16s, 30s cap) on initial connection failure. Retry chain via `reconnectMsg` message. Bottom bar shows attempt count during retries. User can quit at any time. Mid-session disconnects handled by MongoDB driver.

### CI/CD & Release Pipeline
GitHub Actions CI: `task check` (build+vet+lint+test) and `govulncheck` (CVE scanning) run on PRs and main pushes. Dependabot configured for Go modules (root + SDK) and GitHub Actions versions. GoReleaser config for cross-compiled binaries (linux/amd64, linux/arm64, darwin/arm64) with version ldflags. Release workflow triggers on `v*` tags. Added `--version` flag via Kong's `VersionFlag` with `var version = "dev"` overridden at build time.

### Error Message Refinement
Minimal connection error (`"Connection failed (retry in Ns)"`). Suppress `context.Canceled` in follow handlers (normal on double-`f` toggle). Show SDK error directly for action results — removes redundant TUI prefix (`"start: start backup:"` → `"start backup:"`). Suppress `ErrNotFound` from config fetch goroutines (no-main-config is valid state, not an error). SDK `WithConnectTimeout` option bounds each connection attempt (TUI uses 10s) instead of the 30s MongoDB driver default.

### Config CLI Command
`pbmate config` subcommand with `show`, `set`, `unset`, and `path` subcommands. Reflection-based `SetByPath`/`GetByPath`/`UnsetByPath` in `internal/config/field.go` walks struct fields by yaml tag, coerces string values to field types (string, bool, *bool, int, int64, time.Duration). `config set` and `config unset` support `--context` flag to target per-context settings. `config unset` resets fields to zero value (nil for pointers, "" for strings, false for bools). `config show` supports `--context` to display a single context. `config path` prints resolved config file path. `FormatYAML` helper added to config package.

### TUI Code Quality (Review Items)
Extracted `newStandardForm`/`newBorderlessForm` helpers — 8 identical form construction blocks replaced with one-liners. Unified byte formatting: removed `humanize.IBytes` in restore forms, use `humanBytes` consistently everywhere. Split domain-specific detail renderers (`statusIndicator`, `agentIndicator`, `renderBackupDetail`, `renderRestoreDetail`) from `render.go` into `detail_render.go`. Added tests for `rebuildItems` cursor stability (4 scenarios), `selectedItem`, `setRestoreData` cursor clamping, and `appendLogEntries` buffer trimming (4 scenarios).

### Physical Backup Support
SDK `StartPhysicalBackup` command type added (ConfigName, Compression, CompressionLevel only — no Namespaces, NumParallelColls, or IncrBase since PBM ignores them for physical backups). Removed dead `NumParallelColls` field from `StartIncrementalBackup` (PBM's `doPhysical` never reads it). Added `IsPhysical()` and `IsLogical()` domain methods on `Backup`. TUI backup form now offers three types: Logical, Physical, Incremental. "Parallel Collections" input moved to logical-only (was incorrectly shown for all types). Form dynamically rebuilds on type change — Physical shows only Compression + Profile + Confirm.

### Physical/Incremental Restore Safety
Physical and incremental restores shut down mongod on all nodes (including PITR restores with a physical/incremental base). TUI now intercepts these restores with an explicit final warning confirmation overlay before dispatch. On confirm, the restore command is dispatched and the TUI exits cleanly, printing a farewell message to stdout ("Monitor progress with: pbm status"). On dispatch error, the TUI stays open with the error in the flash bar. The `R` wizard includes physical/incremental backups in its selector — the warning overlay is the safety gate.

### PITR Base Backup Selector
SDK `FilterPITRBases` pure function with full validation (StatusDone, LastWriteTS before target, not selective, not external, main config only, timeline coverage). `PITRService.Bases()` server method. TUI PITR restore forms (both Step 1 wizard and Step 2 direct) now show a "Base backup" selector instead of auto-selecting via the old `findBaseBackup()`. Pre-selects most recent valid base. Shows "No valid base backup" note when none qualify. Physical/incremental PITR bases still trigger the physical restore warning overlay.

### SDK Hardening (PITR & Timestamps)
`Timestamp.Before()`/`After()` comparison methods (T first, ordinal tiebreaker). PITR filtering uses `Before()` throughout — fixes ordinal-insensitive comparisons. `FilterPITRBases` switched to `sort.SliceStable`. `PITRService.Bases()` parallelized with `errgroup`. `BackupService.Start` doc updated for `StartPhysicalBackup`. Doc comments on empty `Validate()` methods. Table-driven validate tests for physical and incremental backup commands.

### TUI Restore Form Quality
Extracted `resolvePITRTarget` free function (deduplicated from two `effectivePITRTarget` methods). Extracted `pitrBaseGroup` helper (deduplicated base backup selector logic from `newRestoreTargetForm` and `newPITRRestoreForm`). Fixed `parseNamespaces` empty string edge case (`strings.Split("", ",")` returns `[""]`). Removed dead `backupName` field from `restoreRequest`. Normalized `restoreFormResult` receivers to pointer. Replaced `restoreTargetOverlay.findBackup` method with standalone `findBackupByName`. Added tests for `resolvePITRTarget`, `parseNamespaces`, `backupContextDescription`, `pitrPresetOptions`, `physicalRestoreWarning`, and `findBackupByName`.

### Code Review Fixes
Fixed `latestTimeline()` to use `Timestamp.After()` instead of raw `.T` comparison (same class of ordinal bug as SDK fix). Replaced hardcoded `"c"` string literal in backup form overlay with `customizeKey` binding constant. Added `FilterPITRBases` boundary test for backup at `timeline.End`. Added PBM version compatibility note to SDK README.

## Deferred Features

| Feature | Reason | Priority |
|---------|--------|----------|
| Config domain validation | Validate theme names, URI format, etc. in `config set` | Low |
| Cleanup command | Composable from Backups.Delete + PITR.Delete. Add only if requested. | Low |
| Oplog replay | Very low priority. `CmdTypeReplay` constant ready. | Low |
| External backup start | Out-of-band file operations. Display types exist. | Low |
| Backup collections list | `--with-collections` requires storage I/O. Non-trivial. | Low |
| Diagnostic reports | CLI-oriented, composable from existing services. | Low |
| Wait for delete/resync | No status query exists in PBM to drive polling. | Blocked |
| Per-RS PITR timelines | Diagnostic, not operational. | Low |
| Oplog chunk access | Storage insight, not blocking. | Low |
| Restore progress detail | Nice for PITR progress display. | Low |
| Timeline.Size | Storage cost insight. | Low |
| Status filter on list | Client-side filtering works given small data volumes. | Low |
| Convenience queries | `GetLastBackup` etc. — trivial wrappers. | Low |

## SDK Architecture Notes

### Sealed Command Architecture

User-facing command types use sealed interfaces to make invalid states unrepresentable. Each operation with variants gets a sealed interface with concrete types per variant. Name fields are unexported and auto-generated by service methods.

Completed sealed hierarchies:
- `StartBackupCommand` — `StartLogicalBackup`, `StartPhysicalBackup`, `StartIncrementalBackup`
- `StartRestoreCommand` — `StartSnapshotRestore`, `StartPITRRestore`
- `DeleteBackupCommand` — `DeleteBackupByName`, `DeleteBackupsBefore`
- `DeletePITRCommand` — `DeletePITRBefore`, `DeletePITRAll`
- `ResyncCommand` — `ResyncMain`, `ResyncProfile`, `ResyncAllProfiles`

### Command Architecture Simplification

Removed dead `Command` interface and `CommandService` from public API. Services now call specific converters directly via internal `*commandServiceImpl` helpers (`validateAndCheckLock` + `checkLock` + `dispatch`).
