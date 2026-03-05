# Progress

## Current Status

SDK wraps the core PBM operations (backup, restore, config, cluster, PITR, logs) with gaps remaining (see Deferred Features). TUI has 3 tabs (Overview, Backups, Config) with form overlays, incremental chain support, and context-sensitive restores. See Backlog for planned features. MCP module is a placeholder.

## In Progress

(none)

## Recently Completed

### Pre-Release Code Quality Cleanup (v0.2.0 prep)
Six-commit cleanup pass across all modules:
- `[datefield]` Fix nanosecond truncation in `New()`, fix `WithKeyMap` doc, `go mod tidy`
- `[datefield]` Document UTC normalization and second precision in `doc.go`
- `[datefield]` Add missing test coverage (last-seg digit advance, ModeDate/ModeDateTime bounds, setSegValue preservation, ModeDateTime view rendering)
- `[sdk]` Add `RestoreResult` compliance guards, extract `pitrEnabledKey` const
- Update agent docs: datefield module, missing service methods, overlay files, Config keybindings, release tag row
- `[sdk]` Add missing method docs to README (GetByOpID, ClusterTime, RemoveProfile)

### datefield Module + TUI Integration
Reusable `huh.Field` datetime picker (`datefield/` module). Three modes: `ModeDate`, `ModeDateTime`, `ModeDateTimeSec`. Implements the full `huh.Field` interface (18 methods). Full test coverage including `RunAccessible` fix (`bufio.Scanner` instead of `fmt.Fscan` for whitespace-safe input). Integrated into TUI:
- `bulk_delete_form.go` — `customDate string` → `time.Time`, deleted `parseCustomDate`, `huh.NewInput` → `datefield.New(...).Mode(datefield.ModeDateTimeSec)`
- `restore_form.go` — `pitrTarget string` → `time.Time` in both `restoreFormResult` and `restoreTargetResult`; `resolvePITRTarget` now returns `sdk.Timestamp` directly (parses preset string or uses custom `time.Time`); `pitrBaseGroup`/`pitrBaseOptions` accept `sdk.Timestamp` instead of string; `toPITRCommand` is now infallible (returns single value); deleted `parsePITRTarget`, `pitrTargetFormatAlt`; both `huh.NewInput` custom-target fields replaced with `datefield.New(...).Mode(datefield.ModeDateTimeSec)`. Tests updated to match all new signatures.

## Backlog

Prioritized next items:
- [x] Connection reconnect on failure (auto-retry with exponential backoff)
- [x] Refine error messages (follow context canceled, double prefixes, config ErrNotFound, connect verbosity)
- [x] CI/CD: GitHub Actions (test/lint/vulncheck), Dependabot, GoReleaser, `--version` flag
- [x] Result Type Redesign: `BackupResult.Wait()`, `RestoreResult` interface, `ErrRestoreUnwaitable` (see `docs/agents/sdk-storage-design.md`)
- [ ] MCP server implementation (Phase 4 — scope TBD)
- [x] Homebrew tap for binary distribution

Deferred (add when needed):
- [x] Codecov integration for coverage tracking

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

### Double-Press Quit
`q` now requires a double-press within 2s to quit — prevents accidental exit. First press shows "Press q again to quit" in the status bar (uses `StatusWarning` style, higher priority than `flashErr`). Timer auto-clears the pending state via `quitTimeoutMsg`. `ctrl+c` bypasses the guard for immediate exit. Overlays still dismiss on single `q` (they receive the `Quit` binding, not `ForceQuit`). Help text updated to show `q quit (2x)`.

### PITR Enable/Disable
SDK `PITRService.Enable()` and `Disable()` methods using PBM's exported `config.SetConfigVar` (which handles the epoch bump internally). Returns plain `error` — not `CommandResult` — because PITR toggle is a direct config update, not a PBM command. TUI global `p` key toggles PITR state with a confirmation overlay. Help overlay shows `p` in Global section.

### Bulk Delete (Backups & PITR)
SDK: Replaced `DeletePITRAll` with `DeletePITROlderThan` (takes `time.Duration`; 0 = delete all, matching PBM's deprecation of `--all`). Added `DeleteBackupsOlderThan` with the same duration-based pattern. Fixed `DeleteBackupsBefore.ConfigName` doc — zero value means main config, not all profiles (PBM does not support cross-profile deletion in a single command).

TUI: Backups-tab `D` key opens a bulk delete form overlay; `d` on a PITR timeline opens it with PITR preselected. Target selector (Backups/PITR), preset durations (Now, 1d, 3d, 1w, 2w, 1m, Custom), custom date input (YYYY-MM-DD or YYYY-MM-DD HH:MM), backup type filter (All/Logical/Physical/Incremental), and profile selector (Main + named profiles). Dynamic form rebuild on target/preset changes. Tests cover date parsing, preset resolution, command conversion, and confirm titles.

### Log Filtering
Overview tab `l` key opens a log filter form overlay with severity (Debug/Info/Warning/Error/Fatal), replica set (populated from cluster agents), and event type (backup/restore/cancelBackup/resync/pitr/delete) selectors. Filters persist across poll/follow cycles — changing filters during follow restarts the stream. Panel title shows active filters (e.g. "Logs (W, rs0, backup)"). Log entries enriched with `[rs/port]` source prefix from structured attributes. Apply/Reset buttons in the form.

### Result Type Redesign
`BackupResult` gains `Wait()` method (concrete struct, exported fields preserved). `RestoreResult` becomes an interface with `Name()`, `OPID()`, `Waitable()`, `Wait()`. Two implementations: `waitableRestoreResult` (polls MongoDB, for logical restores) and `unwaitableRestoreResult` (returns `ErrRestoreUnwaitable`, for physical/incremental-based restores). `Wait()` removed from both `BackupService` and `RestoreService` interfaces — waiting lives on the result types. `Start()` looks up backup type to determine result implementation. See `docs/agents/sdk-storage-design.md` for full design rationale and exploratory future directions (storage-based physical restore Wait, ListFiles, Verify).

### Release Preparation
Apache-2.0 license added (compatible with all dependencies — PBM is Apache-2.0, all others MIT/Apache-2.0). CI golangci-lint bumped from v2.6 to v2.9 (Go 1.26 compatibility). Local builds (`task tui:build`, `task tui:install`) now inject version from `git describe` via ldflags. GoReleaser configured with `brews:` section targeting `jcechace/homebrew-tap`. Release workflow passes `HOMEBREW_TAP_TOKEN` secret. Install flow: `brew tap jcechace/tap && brew install pbmate`.

### Incremental Backup Chain Guard
TUI backup form now prevents starting a non-base incremental backup when no incremental chain exists for the selected storage profile. When no chain is found, the "Start new chain?" toggle is hidden and `Base` is forced to `true`, with a note explaining this will start a new chain. Chain detection is per-profile and reacts to profile changes (form rebuilds on profile switch). The backup list is passed through from already-fetched data (no extra SDK call). Tests cover `hasIncrementalChain` (8 scenarios) and form behavior with/without chains.

### External Editor Support
Config tab `e` key opens the selected config or profile in `$EDITOR` (kubectl-style edit flow). Editor resolution chain: `PBMATE_EDITOR` env → `config.editor` → `VISUAL` env → `EDITOR` env → `vi` fallback. Compound editor commands supported (e.g. `"code -w"`). Readonly mode guards the `e` key. Temp file lifecycle: preserved on apply failure (path shown in error for recovery), deleted on success or no changes. SDK `cleanParseError` unwraps `yaml.TypeError` to produce user-friendly validation messages (strips PBM internal type names). Tests cover `cleanParseError` (4 scenarios).

### Credential Unmasking in YAML Output
SDK `GetYAML`/`GetProfileYAML` now support `WithUnmasked()` functional option to produce YAML with real credential values instead of the default masked `"***"`. PBM's `storage.MaskedString` type has `MarshalYAML` that unconditionally masks credentials — a design flaw where a presentation concern (CLI output safety) is baked into the serialization layer. The Percona Operator hit the same issue. Workaround uses BSON roundtrip (`config_unmask.go`): `bson.Marshal` preserves real values (no `MarshalBSON` on `MaskedString`), then converts ordered `bson.D` to `yaml.MapSlice` for output. Filters PBM metadata keys (`epoch`, `name`, `profile`). Default (no option) returns masked YAML safe for display; TUI editor flow passes `WithUnmasked()` for roundtripping. Tests cover unmasking, metadata filtering, field order, and BSON-to-YAML conversion.

### Integration Test Harness
testcontainers-go with Percona Server for MongoDB 8.0 single-node replica set. Package `sdk/integtest/` with `//go:build integration` tag. `TestMain` manages container lifecycle (start, connect SDK + raw Mongo, cleanup, terminate). Fixture builders (`newBackupMeta`, `newRestoreMeta`, `newAgentStat`, `newLockData`, `newMainConfig`, `newPITRChunk`) with functional options for all PBM types. Seeding helpers insert PBM documents directly. Verification helpers read back commands. `task sdk:integration` target. Also fixed `Config.Get`/`GetYAML` to handle `mongo.ErrNoDocuments` (PBM's `GetConfig` doesn't translate to `ErrMissedConfig` unlike `GetProfile`).

### Backup Integration Tests
14 integration tests for `BackupService` covering `List`, `Get`, and `GetByOpID`. Tests verify: sort order (newest first), empty list, limit, type filtering, profile filtering, full field mapping (type, status, compression, timestamps, sizes, namespaces, versions), domain methods (`IsSelective`, `IsIncremental`, `IsIncrementalBase`, `IsPhysical`, `IsLogical`, `InProgress`), incremental chain (base + child with `SrcBackup`), `ConfigName` normalization (main vs named profile), error status with message, and all 5 status round-trips (done, error, running, starting, cancelled).

### Config Integration Tests
13 integration tests for `ConfigService` covering `Get`, `GetYAML`, `SetYAML`, `ListProfiles`, `GetProfile`, `GetProfileYAML`. Tests verify: basic config read (storage type/path/region), not-found handling for all read methods, full section mapping (PITR with enabled/oplogOnly/compression/priority, Backup with compression/parallelism/timeouts, Restore with all tuning fields), PBM default initialization behavior (nil sub-configs become empty structs with default S2 compression), YAML roundtrip via `SetYAML` (on existing and empty DB), profile listing (multiple profiles, empty), and S3 profile field mapping. Config fixture options added for PITR, Backup, Restore sections.

### Cluster Integration Tests
12 integration tests for `ClusterService` covering `Members`, `Agents`, `RunningOperations`, `CheckLock`, `ClusterTime`, `ServerInfo`. Tests verify: real RS topology from testcontainers (single-node RS named "rs"), agent field mapping (node, RS, version, role, OK, stale, errors), stale agent detection (heartbeat in GC-safe stale window — PBM GCs at 35s, SDK marks stale at 30s), agent with error string, empty agent list, running operations from non-stale locks, stale lock filtering, CheckLock clear/blocked (`ConcurrentOperationError`), non-zero cluster time, and server info version strings (Mongo 8.0, PBM version).

### Restore Integration Tests
13 integration tests for `RestoreService` covering `List`, `Get`, and `GetByOpID`. Tests verify: sort order (newest first), empty list, limit, full field mapping (name, OPID, backup, type, status, timestamps, PITR target, namespaces, replsets with nodes), FinishTS derivation from LastTransitionTS on terminal status, domain methods (`InProgress`, `Duration`, `Elapsed`), PITR target (present and absent), incremental backup chain (`BcpChain`), error status with message, and all 5 status round-trips. Restore fixture options added (`withRestoreOPID`, `withRestoreStartTS`, `withRestoreLastTransitionTS`, `withRestoreNamespaces`, `withRestoreReplsets`, `withRestoreBcpChain`, `withRestoreError`).

### PITR Integration Tests
10 integration tests for `PITRService` covering `Timelines`, `Bases`, `Status`, `Enable`, and `Disable`. Tests verify: single contiguous timeline from merged chunks, empty timelines, gap producing two separate timelines, PITR base filtering (valid base, empty bases, selective backups excluded, profile backups excluded), PITR status disabled (config without PITR section), PITR status enabled but not running (no active locks), and Enable/Disable round-trip (seed config → enable → verify → disable → verify).

### Log Integration Tests
6 integration tests for `LogService.Get` covering basic retrieval, field mapping, filtering, and limits. Tests verify: full field mapping (timestamp, severity, message, attrs with RS/node/event/objName), empty log collection, limit enforcement, severity filtering (Warning includes Fatal+Error+Warning via PBM's `$lte` filter), replica set filtering, and event type filtering. Log fixture builder (`newLogEntry`) and options (`withLogSeverity`, `withLogRS`, `withLogNode`, `withLogEvent`, `withLogTS`, `withLogObjName`, `withLogOPID`) added. `seedLog` helper for `pbmLog` collection. Follow tests deferred (requires capped collection for tailable cursor).

### SDK Coverage Target
`sdk:cover` runs all SDK tests (unit + integration) in a single `go test` pass with `-tags integration -coverpkg=./...` and prints a per-function coverage report. Uses `-coverpkg=./...` so integration tests in `sdk/integtest/` count toward `sdk/v2` coverage. Coverage output files (`*.out`) already ignored by `.gitignore`.

### Coverage Gap Tests
10 new tests filling meaningful coverage gaps, raising SDK coverage from 87.9% to 89.6%. Unit tests: `TestTranslateCanDeleteError` (4-case table test covering nil, ErrBackupInProgress, ErrBaseForPITR, generic error), `TestAddProfileCommandValidate`, `TestNewClientNoBackend`. Integration tests: `TestConfigGetYAMLMasked`/`TestConfigGetYAMLUnmasked`/`TestConfigGetProfileYAMLUnmasked` (credential masking roundtrip with `WithUnmasked()` option), `TestBackupStartAndWait`/`TestBackupStartAndWaitError` (seed terminal metadata before Wait, verify immediate return with correct status and OperationError), `TestRestoreStartAndWait`/`TestRestoreStartUnwaitableWait` (waitable vs unwaitable result paths), `TestClientClose` (second client lifecycle). Remaining 0% functions are sealed interface markers (empty method bodies) and `LogService.Follow` (requires capped collection).
## Deferred Features

| Feature | Reason | Priority |
|---------|--------|----------|
| Config domain validation | Validate theme names, URI format, etc. in `config set` | Low |
| Cleanup command | Superseded by bulk delete (`D` key). | Done |
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
| Physical restore Wait (storage) | Requires storage handle creation in `Start()`. Design ready in `sdk-storage-design.md`. | Medium |
| BackupService.ListFiles | List backup contents from storage. | Medium |
| BackupService.Verify | Check backup file integrity on storage. | Medium |
| Storage-only client | SDK client from config file, no MongoDB needed. | Low |
| ConfigService.SetVar/GetVar | Per-key config changes via PBM's `config.SetConfigVar`. | Low |
| ConfigService.CheckStorage | Verify storage accessibility. | Low |

## SDK Architecture Notes

### Sealed Command Architecture

User-facing command types use sealed interfaces to make invalid states unrepresentable. Each operation with variants gets a sealed interface with concrete types per variant. Name fields are unexported and auto-generated by service methods.

Completed sealed hierarchies:
- `StartBackupCommand` — `StartLogicalBackup`, `StartPhysicalBackup`, `StartIncrementalBackup`
- `StartRestoreCommand` — `StartSnapshotRestore`, `StartPITRRestore`
- `DeleteBackupCommand` — `DeleteBackupByName`, `DeleteBackupsBefore`, `DeleteBackupsOlderThan`
- `DeletePITRCommand` — `DeletePITRBefore`, `DeletePITROlderThan`
- `ResyncCommand` — `ResyncMain`, `ResyncProfile`, `ResyncAllProfiles`

### Command Architecture Simplification

Removed dead `Command` interface and `CommandService` from public API. Services now call specific converters directly via internal `*commandServiceImpl` helpers (`validateAndCheckLock` + `checkLock` + `dispatch`).
