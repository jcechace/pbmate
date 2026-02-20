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

### Phase 5d: Additional tabs and features (planned)
- [ ] Config tab
- [ ] Detail panel sub-tabs (`[`/`]`) for Backups (Info, Replicas, Logs)
- [ ] `/` filter in list views
- [ ] `--readonly` flag
- [ ] Connection reconnect on failure (currently dead-end after connect error)
- [ ] Fix goroutine leak in log follow mode (nextLogCmd blocks forever if follow stopped between dispatch and message arrival)
