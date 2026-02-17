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

## Phase 3: SDK Implementation
- [ ] BackupService -- wire to PBM internals
- [ ] RestoreService -- wire to PBM internals
- [ ] ConfigService -- wire to PBM internals
- [ ] ClusterService -- wire to PBM internals
- [ ] PITRService -- wire to PBM internals
- [ ] LogService -- wire to PBM internals

## Phase 4: MCP Server
- [ ] (TBD)

## Phase 5: TUI
- [ ] (TBD)
