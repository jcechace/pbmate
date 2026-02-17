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
