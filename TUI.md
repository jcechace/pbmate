# PBMate TUI Design

## Vision

A lazydocker-inspired panel-based TUI for monitoring and managing PBM backups.
Keyboard-driven, information-dense, but visually rich with bordered panels,
color-coded status indicators, and a master-detail layout that makes good use of
screen space even with few resources.

Built with BubbleTea (Elm architecture), Bubbles components, and Lipgloss
styling.

## Layout

```
┌──────────────────────────────────────────────────────────────────┐
│  PBMate   [1:Overview]  2:Backups  3:Restores  4:Config  5:Logs │
├──────────────────────────┬───────────────────────────────────────┤
│                          │                                       │
│    Left panel            │         Right panel (detail)          │
│    (navigable list)      │         (selected item info)          │
│                          │                                       │
├──────────────────────────┴───────────────────────────────────────┤
│  PITR: on │ Op: backup (running) │ Cluster: 1740000000          │
├──────────────────────────────────────────────────────────────────┤
│  ?:help  q:quit  1-5:tabs  tab/shift-tab:cycle                  │
└──────────────────────────────────────────────────────────────────┘
```

Four zones: **header** (tab bar), **content** (left list + right detail panels),
**status bar**, **help bar**.

## Tabs

### 1. Overview (landing page)

Left panel -- two sections:
- **Cluster**: agents grouped by replica set (tree-like). Each agent shows
  node, role, version, status indicator.
- **Recent Backups**: last ~5 backups with compact summary (name, type, status).

Right panel -- detail for selected item:
- If agent selected: node, RS, role, version, status, errors.
- If backup selected: full backup info.
- Always shows: running operations, PITR status/timelines.

Scales from single-node RS to sharded clusters (3+ shards x 3 nodes + config
RS).

```
┌─ Cluster ────────────────┬─ Detail ──────────────────────────────┐
│ ▸ rs1                    │                                        │
│   ● rs101:27017  P  2.8  │  Agent: rs101:27017                   │
│   ● rs102:27017  S  2.8  │  Replica Set: rs1                     │
│   ○ rs103:27017  S  2.8  │  Role: Primary                        │
│ ▸ cfg                    │  Version: 2.8.0                        │
│   ● cfg01:27017  P  2.8  │  Status: OK                            │
│   ● cfg02:27017  S  2.8  │                                        │
│   ● cfg03:27017  S  2.8  │  ─── Running Operations ───           │
│                          │  backup logical 2024-01-15T10:30:00Z   │
├─ Recent Backups ─────────┤                                        │
│  2024-01-15T10:30 ● done │  ─── PITR ───                         │
│  2024-01-14T22:00 ● done │  Enabled: true  Running: true          │
│  2024-01-14T10:00 ● done │  Timeline: 1740000000 → 1740003600     │
└──────────────────────────┴────────────────────────────────────────┘
```

### 2. Backups

Left panel: navigable list of all backups (name, type, status indicator, size).
Right panel: full detail of selected backup (metadata, replset breakdown,
errors).

Actions: `s` start backup, `d` delete, `c` cancel running.

```
┌─ Backups ────────────────┬─ Detail ──────────────────────────────┐
│                          │                                        │
│  2024-01-15T10:30:00Z    │  Name: 2024-01-15T10:30:00Z           │
│  logical  ● done    73KB │  Type: logical                         │
│                          │  Status: done                          │
│  2024-01-14T22:00:00Z    │  Size: 73KB (120KB uncompressed)       │
│  logical  ● done    68KB │  Config: main                          │
│                          │  Compression: zstd                     │
│  2024-01-14T10:00:00Z    │  Started: 2024-01-15 10:30:00          │
│  physical ● done  1.2GB  │  Duration: 6s                          │
│                          │                                        │
│                          │  ─── Replica Sets ───                  │
│                          │  rs1: ● done  (rs101:27017)            │
│                          │  cfg: ● done  (cfg01:27017)            │
└──────────────────────────┴────────────────────────────────────────┘
```

### 3. Restores

Left panel: list of restores (name, status, source backup).
Right panel: detail of selected restore.

Simple view -- restores are ephemeral in PBM. Actions: `s` start restore.

### 4. Config

Left panel: main config + storage profiles list.
Right panel: detail of selected item (storage config, PITR settings,
compression).

Read-only for MVP. Later: edit via YAML.

### 5. Logs

Full-width viewport (no left/right split). Streaming log entries via
`Logs.Follow`. Severity filter. Auto-scroll with toggle.

## Keybindings

### Global
- `q` / `ctrl+c` -- quit
- `1`-`5` -- jump to tab
- `tab` / `shift+tab` -- cycle tabs
- `?` -- toggle help overlay
- `esc` -- back / close overlay

### Navigation (within panels)
- `up`/`down` or `k`/`j` -- move selection in list
- `left`/`right` or `h`/`l` -- switch focus between left/right panel

### Tab-specific
- Backups: `s` start, `d` delete, `c` cancel
- Restores: `s` start restore
- Logs: `w` toggle wrap, `f` toggle auto-scroll

## Polling & Data Flow

- **Tick-based polling** with adaptive intervals:
  - Idle: 10s
  - Active operation detected: 2s
- Only fetch data for the active tab.
- **Log streaming**: goroutine pipes `Logs.Follow()` channel into
  `Program.Send`.
- **Backup/restore progress**: `Wait` with `OnProgress` callback sends status
  messages into the event loop.

## Styling

- **Status colors**: green = done/ok, red = error, yellow = running/in-progress,
  gray = cancelled/stale.
- **Status indicators**: `●` (filled) for healthy/terminal, `○` (empty) for
  error/stale.
- Bordered panels with lipgloss `RoundedBorder`.
- Adaptive colors (`lipgloss.AdaptiveColor`) for light/dark terminals.
- Compact, information-dense -- no wasted space.

## Project Structure

```
pbmate/
├── main.go                    # Entry point: flags, SDK client, tea.Program
├── internal/
│   └── tui/
│       ├── app.go             # Root model: Init, Update, View, tab routing
│       ├── keys.go            # Key bindings (global + per-view keymaps)
│       ├── styles.go          # Lipgloss styles, colors, borders
│       ├── status.go          # Status bar sub-model
│       ├── overview.go        # Overview tab (agents + recent backups)
│       ├── backups.go         # Backups tab
│       ├── restores.go        # Restores tab
│       ├── config.go          # Config tab
│       └── logs.go            # Logs tab
```

## Dependencies

- `charmbracelet/bubbletea` -- framework
- `charmbracelet/bubbles` -- table, list, spinner, help, key, viewport, progress
- `charmbracelet/lipgloss` -- styling and layout
- `jcechace/pbmate/sdk/v2` -- PBMate SDK

## MVP Scope (Phase 5a)

1. App skeleton: tab bar, status bar scaffold, help bar, window size handling
2. Overview tab: agent tree grouped by RS, running ops, PITR status, recent
   backups
3. Backups tab: list with detail pane, start/cancel backup
4. Adaptive tick polling

### Deferred to later iterations
- Restores tab
- Config tab
- Logs tab with streaming
- Delete backup action
- Start restore from backups tab
- Progress indicators for active operations
- Help overlay
