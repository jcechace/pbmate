# PBMate Progress

## Phase 1: Project Setup
- [x] .gitignore
- [x] AGENTS.md
- [x] Update go.mod files to Go 1.26
- [x] Bootstrap MCP module (mcp/go.mod)
- [x] PROGRESS.md (this file)

## Phase 2: SDK Scaffolding
- [x] Shared types (types.go)
- [x] Client struct (client.go) -- functional options pattern
- [x] BackupService (interface + impl stub)
- [x] RestoreService (interface + impl stub)
- [x] ConfigService (interface + impl stub)
- [x] ClusterService (interface + impl stub)
- [x] PITRService (interface + impl stub)
- [x] LogService (interface + impl stub)
- [x] PBM dependency via replace directive (local source)
- [x] Refactor enum types to DDD-style value objects (unexported field, exported instances, Parse functions)
- [x] Add ConfigName type to encapsulate config/profile identity (normalizes PBM's "" -> "main")
- [x] Redesign ConfigService -- unified Config type, GetYAML/GetProfileYAML for raw access, separate typed vs raw methods
- [x] Redesign LogService -- simplify to Get+Follow, structured attrs via map[string]any, well-known key constants
- [x] Expand Restore types -- BcpChain for incremental, RestoreNode for physical restore monitoring
- [x] Add restore-specific Status values (StatusDown, StatusCleanupCluster)
- [x] CommandService -- sealed Command interface, BackupCommand/RestoreCommand/CancelBackupCommand, result types
- [x] Write operations on services -- BackupService.Start/Cancel, RestoreService.Start (delegate to CommandService)
- [x] Wire CommandService into Client and service impls

## Phase 3: SDK Implementation
- [x] Foundation -- errors (ErrNotFound, ConcurrentOperationError), shared conversion helpers, CommandType enum, Operation struct revision
- [x] CommandService -- lock checking via lock.GetLocks + staleness, command dispatch via CmdStreamCollection insert
- [x] BackupService -- custom MongoDB query for List with filters, Get/GetByOpID via PBM functions, Start/Cancel via CommandService
- [x] RestoreService -- List via restore.RestoreList, Get/GetByOpID via PBM functions, Start via CommandService
- [x] PITRService -- Status aggregation (config + oplog slicing + active slicers + meta errors), Timelines via oplog.PITRTimelines
- [x] ClusterService -- Members via topo.ClusterMembers, Agents via topo.ListAgentStatuses, RunningOperations via lock.GetLocks with stale filtering, ClusterTime via topo.GetClusterTime
- [x] ConfigService -- Get/GetYAML via config.GetConfig, ListProfiles/GetProfile/GetProfileYAML via config profile functions
- [x] LogService -- Get via log.LogGet with Info default severity, Follow via log.Follow tailable cursor with goroutine adapter

## Phase 4: MCP Server
- [ ] (TBD)

## Phase 5: TUI

### Phase 5a: Initial scaffold (complete)
- [x] TUI design document (TUI.md)
- [x] App skeleton with tab navigation and window size handling
- [x] Theming support with Catppuccin color palettes
- [x] Tick-based polling with adaptive intervals (10s idle, 2s active)
- [x] Overview tab with agent tree and recent backups
- [x] Right panel detail view for selected agent/backup
- [x] Backups tab with list and detail panels
- [x] Backup actions: start, cancel, delete
- [x] Shared rendering helpers (render.go)

### Phase 5b: TUI redesign (complete)
- [x] TUI design research (lazydocker, lazygit, k9s, gh-dash, btop, ctop, dry, dolphie)
- [x] Revised TUI.md with new 4-quadrant Overview layout
- [x] Drop Logs tab -- change from 5 to 4 tabs
- [x] Merge two bottom bars into single bar (status HUD left, hints right)
- [x] Redesign Overview: remove Recent Backups, 4-quadrant layout
- [x] Collapsible RS groups with inline status indicators
- [x] Status panel (PITR, op, latest backup with relative age, storage)
- [x] Fetch config/storage and latest backup data for status panel
- [x] Log panel in Overview bottom-right (5s refresh + 50 entries)
- [x] Follow mode toggle (`f`) for log panel via LogService.Follow
- [x] Stable cursor -- track selection by item identity, not index
- [x] Context-sensitive action hints in bottom bar
- [x] Fix log panel jump when toggling follow mode
- [x] Migrate all panels to viewport components
- [x] Fix panel overflow by correcting lipgloss width calculations
- [x] Add scrollable log viewport with auto-pin and wrap toggle (`w`)
- [x] Redesign panel focus: 4-quadrant `[]` cycling, per-panel Up/Down dispatch
- [x] Move start/cancel backup actions (`s`/`c`) to global scope

### Phase 5b+: Architecture refactoring (complete)
- [x] Extract layout helpers (horizontalSplit, innerHeight) into layout.go
- [x] Make backupsModel self-contained with view() and resize()
- [x] Make overviewModel self-contained with view() and resize()
- [x] Add SectionHeader and Bold to Styles, move relativeTime to render.go
- [x] Extract logPanel as a reusable component (log_panel.go)
- [x] Move log follow state from root Model into overviewModel
- [x] Move shared panel type to layout.go
- [x] Eliminate data duplication between root Model and sub-models
- [x] Extract clusterPanel from overviewModel (cluster_panel.go)
- [x] Add panel titles to border rendering (╭─ Title ─────╮)

### Phase 5c: Interactions (complete)
- [x] `huh` form overlay for start backup (quick confirm + full wizard with type, compression, profile)
- [x] `huh` confirm overlay for destructive actions (delete, cancel)
- [x] `?` full help overlay
- [x] Incremental backup chain grouping (base + indented children in backup tree)
- [x] Chain-aware delete (auto-resolves to base, shows chain count)
- [x] Restore list with `tab` toggle in Backups tab
- [x] SDK documentation enrichment (usage examples and field comments on all domain files)

### Phase 5c+: Code quality refactoring (complete)
- [x] Fix PITR duration truncation (`Truncate(time.Second)` not `Truncate(1)`)
- [x] Fix stale timeline cursor pointer comparison (compare by value, not pointer)
- [x] Fix error clearing asymmetry (all data messages clear flashErr on success)
- [x] Extract chain logic into backup_chain.go (chainOrderedItems, resolveIncrChain, groupBackupsByProfile, sortedProfileNames, profileDisplayName)
- [x] Add comprehensive tests for chain logic (backup_chain_test.go)
- [x] Deduplicate cursor rendering (renderCursorList helper in render.go)
- [x] Unify replaceTitleBorder / replaceStyledTitleBorder (wrapper delegation)
- [x] Derive help overlay content from key.Binding definitions (eliminates drift)
- [x] Consistent `*Styles` passing in all function signatures
- [x] Move layout constants to layout.go, maxLogEntries to overview.go

### SDK domain enrichment (complete)
- [x] Remove isTerminalStatus in favor of Status.IsTerminal()
- [x] Add domain methods to Backup (IsIncremental, IsIncrementalBase, IsSelective, InProgress, Duration) with tests
- [x] Add domain methods to Restore (InProgress, Duration) with tests
- [x] Document ConfigName normalization guarantee on Backup.ConfigName field
- [x] Add severity filtering to LogService Get/Follow (GetLogsOptions, FollowOptions structs)
- [x] Add BackupChain type with GroupIncrementalChains and FindChainBase + tests
- [x] Refactor TUI chain logic to use SDK BackupChain utilities
- [x] Replace string switches in toOptions() with ParseBackupType/ParseCompressionType
- [x] Remove debug fmt.Println from cluster_convert_test.go
- [x] Add package-level documentation (sdk/doc.go)
- [x] Remove defensive zero-ConfigName checks in TUI (relies on SDK normalization guarantee)
- [x] Fix goroutine leak in LogService.Follow adapter on context cancellation

### Code quality polish (complete)
- [x] Use SDK domain methods at all remaining TUI call sites (IsIncrementalBase, IsIncremental, IsSelective)
- [x] Extract waitForTerminal generic helper to unify Backup/Restore Wait pattern
- [x] Standardize error message prefixes to verb-noun format across all SDK services
- [x] Show profile name in delete confirmation dialog

### Phase 5d: Config tab (complete)
- [x] Config tab with main config + profile list + profile detail
- [x] Profile YAML syntax highlighting (Chroma, theme-matched)
- [x] File picker overlay for applying config / profile YAML
- [x] Profile name form for creating new profiles
- [x] `huh` form overlays extracted to `formOverlay` interface (overlay.go)

### Code quality improvements (complete)
- [x] SDK: Log warnings for unknown PBM enum values in conversions
- [x] SDK: Skip warnings for empty/unset PBM enum values (zero = valid)
- [x] SDK: Generic `convertSlice` helper
- [x] SDK: Unify Limit type to int across all service options
- [x] SDK: Remove duplicate convertLogTimestamp
- [x] SDK: Wrap Client.Close error with context
- [x] SDK: Add TODO(pbm-fix) markers to PBM workarounds
- [x] SDK: MarshalText/UnmarshalText round-trip tests for all value objects
- [x] TUI: Extract formOverlay interface to unify 4 overlay handler patterns
- [x] TUI: Map Chroma syntax highlighting style to user's theme
- [x] TUI: Unify bare "main" literals to defaultConfigName constant
- [x] TUI: Rename backupFormInnerWidth to formOverlayInnerWidth
- [x] TUI: Document rationales for magic number constants
- [x] TUI: Thread root context through cmd factories and overlays
- [x] TUI: Remove unimplemented Restore keybinding
- [x] TUI: Unit tests for pure render helpers (statusIndicator, agentIndicator, etc.)

### Theming fixes (complete)
- [x] Build per-flavor huh themes instead of using adaptive `ThemeCatppuccin()`

### SDK hardening (complete)
- [x] Fix CanDelete: reject non-base increments with `ErrNotChainBase`
- [x] Add `Validate() error` to Command interface; all commands self-validate
- [x] Move service-level validations into command `Validate()` methods
- [x] Add `UsersAndRoles` field to `StartLogicalBackup` for selective backups
- [x] Shared validation helpers: `validateUsersAndRoles`, `validateNamespaceRemap`
- [x] Comprehensive `Validate()` tests for all command types

### Phase 5e: Additional TUI features (planned)
- [ ] Detail panel sub-tabs (`[`/`]`) for Backups (Info, Replicas, Logs)
- [ ] `/` filter in list views
- [ ] `--readonly` flag
- [ ] Connection reconnect on failure (currently dead-end after connect error)

---

## SDK Completeness

Gap analysis vs PBM CLI completed. Features were refined individually before
implementation, with design questions (type choices, validation, API shape)
resolved per-feature.

### Sealed command architecture

Major architectural refactor: user-facing command types use sealed interfaces
to make invalid states unrepresentable. Each operation with variants gets a
sealed interface (`StartBackupCommand`, `StartRestoreCommand`, etc.) with
concrete types for each variant. Name fields are unexported and auto-generated
by service methods.

- [x] Sealed `StartBackupCommand` — `StartLogicalBackup`, `StartIncrementalBackup`
- [x] Sealed `StartRestoreCommand` — `StartSnapshotRestore`, `StartPITRRestore`
- [x] Sealed `DeleteBackupCommand` — `DeleteBackupByName`, `DeleteBackupsBefore`
- [x] Sealed `DeletePITRCommand` — `DeletePITRBefore`, `DeletePITRAll`
- [x] Sealed `ResyncCommand` — `ResyncMain`, `ResyncProfile`, `ResyncAllProfiles`

### Completed

- [x] Log filtering — `LogFilter` struct with event, replica set, node, OPID;
      embedded in `GetLogsOptions` (adds `TimeMin`/`TimeMax`) and `FollowOptions`
- [x] Bulk backup deletion — sealed `DeleteBackupCommand` with `DeleteBackupByName`
      and `DeleteBackupsBefore` (older-than + type + profile filter)
- [x] Delete PITR — sealed `DeletePITRCommand` with `DeletePITRBefore` and
      `DeletePITRAll` on `PITRService.Delete()`
- [x] Restore options — sealed `StartRestoreCommand` with `StartSnapshotRestore`
      and `StartPITRRestore`, including namespace remapping, users-and-roles,
      and RS remapping fields
- [x] Backup CompressionLevel — `CompressionLevel *int` on `StartLogicalBackup`
      and `StartIncrementalBackup`
- [x] Resync — sealed `ResyncCommand` with `ResyncMain`, `ResyncProfile`, and
      `ResyncAllProfiles` on `ConfigService.Resync()`
- [x] Server info — `ClusterService.ServerInfo()` returning MongoDB version
      and PBM library version

### Deferred

| Feature | Reason |
|---------|--------|
| Config SetVar | Deferred pending PITR enable/disable design — needs a cohesive config mutation surface |
| Cleanup command | Needs analysis: does PBM's cleanup codepath differ materially from delete-backup + delete-pitr? |
| Oplog replay | Advanced disaster recovery. `CmdTypeReplay` constant ready. Implement when needed. |
| Physical/external backup | Out-of-band file operations. Display types exist. Start/Finish deferred. |
| Performance knobs | `NumParallelColls`, `NumInsertionWorkers` — server-side tuning with sensible PBM defaults. |
| Backup collections list | `--with-collections` requires storage I/O (reads archive files). Non-trivial. |
| Diagnostic reports | CLI-oriented, composable from existing service methods. |
