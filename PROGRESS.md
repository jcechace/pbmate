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
- [ ] (TBD)
