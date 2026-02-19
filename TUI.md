# PBMate TUI Design

## Vision

A monitoring-first panel-based TUI for managing PBM backups. Keyboard-driven,
information-dense, visually rich with bordered panels, color-coded status
indicators, and a master-detail layout. Inspired by lazydocker's simplicity,
k9s's power, and gh-dash's BubbleTea patterns.

Built with BubbleTea (Elm architecture), Bubbles components, and Lipgloss
styling.

## Layout

```
┌──────────────────────────────────────────────────────────────────┐
│  PBMate   [1:Overview]  2:Backups  3:Restores  4:Config         │
├──────────────────────────┬───────────────────────────────────────┤
│                          │                                       │
│    Left panel            │         Right panel (detail)          │
│    (navigable list)      │         (selected item info)          │
│                          │                                       │
├──────────────────────────┴───────────────────────────────────────┤
│ PITR:on  Op:none  15:04 │ ↑↓:nav  s:backup  ?:help  q:quit      │
└──────────────────────────────────────────────────────────────────┘
```

Three zones: **header** (tab bar), **content** (tab-specific layout),
**bottom bar** (status HUD left, context-sensitive hints right).

## Tabs

### 1. Overview (landing page / monitoring dashboard)

Four-quadrant layout:

```
┌─── Cluster ─────────────┬─── Detail ────────────────────────────┐
│                          │                                       │
│  ▾ rs1 (3)               │  Agent: rs101:27017                   │
│    ● rs101:27017  P 2.8  │  Replica Set: rs1                     │
│    ● rs102:27017  S 2.8  │  Role: Primary                        │
│    ○ rs103:27017  S 2.8  │  Version: 2.8.0                       │
│  ▸ rs2 ●●● (3)          │  Status: OK                            │
│  ▸ rs3 ●●○ (3)          │                                       │
│  ▸ cfg ●●● (3)          │                                       │
│                          │                 ~60%                   │
├─── Status ──────────────┼─── Logs ──────────────────────────────┤
│                          │                                       │
│  PITR     on (running)   │  15:04:05 I backup start logical      │
│  Op       none           │  15:04:06 I rs1 snapshot started      │
│  Latest   01-15 10:30 ● │  15:04:12 I rs1 snapshot done         │
│            (3h ago)      │  15:04:12 I backup done               │
│  Storage  s3://bucket    │                 [refresh: 5s]         │
│                          │                 ~40%                   │
├──────────────────────────┴───────────────────────────────────────┤
│ PITR:on  Op:none  15:04 │ ↑↓:nav  ␣:expand  s:backup  f:follow  │
└──────────────────────────────────────────────────────────────────┘
```

**Top-left (Cluster)**: Scrollable agent tree grouped by replica set.

- RS headers are collapsible: `▾` expanded, `▸` collapsed.
- Collapsed RS shows inline status dots (one per agent) + count:
  `▸ rs2 ●●● (3)` -- all healthy; `▸ rs3 ●●○ (3)` -- one stale.
- Space or Enter toggles expand/collapse on RS headers.
- Scales to 100+ shard clusters via collapse + scrolling.
- Status dots: `●` green = OK, `●` red = error, `○` dim = stale.

**Top-right (Detail, ~60%)**: Detail for the selected agent. Shows node, RS,
role, version, status, errors. When no agent selected, shows a cluster summary.

**Bottom-left (Status)**: Static operational status panel:
- PITR enabled/running status
- Active operation (with type, or "none")
- Latest backup name + status + relative age ("3h ago")
- Main storage type + path/bucket

**Bottom-right (Logs, ~40%)**: Live PBM log viewer.
- Auto-refreshes every 5 seconds by default.
- `f` toggles follow mode (continuous streaming via LogService.Follow).
- Color-coded by severity: D=dim, I=normal, W=yellow, E=red.
- Shows time, severity, and message in compact format.

### 2. Backups

Master-detail layout with sub-tabs in the detail panel.

```
┌─── Backups ─────────────┬─── [Info] Replicas  Logs ─────────────┐
│                          │                                       │
│▶ 2024-01-15 10:30        │  Name: 2024-01-15T10:30:00Z           │
│  logical  ● done    73KB │  Type: logical                         │
│                          │  Status: done                          │
│  2024-01-14 22:00        │  Compression: zstd                     │
│  logical  ● done    68KB │  Size: 73KB (120KB raw)                │
│                          │  Profile: main                         │
│  2024-01-14 10:00        │  Started: 2024-01-15 10:30:00          │
│  physical ● done  1.2GB  │  Completed: 2024-01-15 10:30:06        │
│                          │  Duration: 6s                          │
│  2024-01-13 10:00        │                                       │
│  incremental ● done 45KB │  Errors: none                          │
│                          │                                       │
│  (scrollable, filterable)│                                       │
├──────────────────────────┴───────────────────────────────────────┤
│ PITR:on  Op:none  15:04 │ s:backup  d:delete  c:cancel  ?:help   │
└──────────────────────────────────────────────────────────────────┘
```

Left panel: full scrollable backup list. Compact names (drop seconds).

Right panel: Full backup metadata, timestamps, compression, errors.

Future sub-tabs (Phase 5c): Info, Replicas, Logs -- will use dedicated keys
for cycling within the detail panel.

Actions: `s` start backup, `d` delete, `c` cancel running.

Future: PITR timeline visualization, incremental backup chain view, `/` filter.

### 3. Restores

Same master-detail pattern.

Left panel: list of restores (backup source, type, status).
Right panel: restore detail with sub-tabs (Info, Nodes, Logs).

Actions: `r` start restore from selected backup (via Backups tab).

### 4. Config

Left panel: main config + storage profiles list.
Right panel: detail (storage settings, PITR config, compression, raw YAML).

Read-only for MVP. Later: `e` edit via YAML, `p` add profile.

## Keybindings

### Global
- `q` / `ctrl+c` -- quit
- `1`-`4` -- jump to tab
- `tab` / `shift+tab` -- cycle tabs
- `?` -- toggle full help overlay
- `esc` -- back / close overlay / clear filter

### Navigation (within panels)
- `up`/`down` or `k`/`j` -- move selection / scroll in focused panel
- `]` / `[` -- cycle focus to next/previous panel

### Overview-specific
- `space` / `enter` -- expand/collapse RS group (when cluster panel focused)
- `f` -- toggle log follow mode
- `w` -- toggle log word-wrap

### Backups-specific
- `d` -- delete backup
- (`s` start backup and `c` cancel are global -- work from any tab)

### Restores-specific
- (view only for now)

### Future
- `/` -- filter/search in list views
- `Ctrl+z` -- toggle error-only filter

## Bottom Bar

Single merged bar replacing the previous two-bar (status + help) design.

```
│ PITR:on  Op:backup(●)  15:04 │ s:backup  d:delete  c:cancel  ?:help  q:quit │
```

**Left zone**: Persistent operational HUD visible on all tabs.
- PITR status (on/off)
- Running operation type (with spinner `●` when active, "none" when idle)
- Cluster time (HH:MM format)

**Right zone**: Context-sensitive keybinding hints.
- Changes per tab and per selection state.
- Keys in bold/primary color, descriptions in muted color.
- Hints that exceed available width are dropped (rightmost first).
- 6-8 most important hints; full reference via `?` overlay (planned).

## Polling & Data Flow

- **Tick-based polling** with adaptive intervals:
  - Idle: 10s
  - Active operation detected: 2s
- Overview always fetches cluster + status data (needed for bottom bar HUD).
- Tab-specific data fetched only for the active tab.
- **Log panel**: polls every 5s in normal mode; streams via `Logs.Follow()`
  goroutine in follow mode.
- **Stable cursor**: selection tracked by item identity (agent node name,
  backup name), not by list index. Prevents cursor jumping on data refresh.

## Styling

- **Status colors**: green = done/ok, red = error, yellow = running/in-progress,
  gray = cancelled/stale.
- **Status indicators**: `●` (filled green) = healthy/done, `●` (filled red) =
  error, `○` (empty/dim) = stale/cancelled. Shape + color for accessibility.
- Bordered panels with lipgloss `RoundedBorder`.
- Adaptive colors (`lipgloss.AdaptiveColor`) for light/dark terminals.
- Compact, information-dense -- no wasted space.
- Catppuccin theme support (Mocha/Latte/Frappe/Macchiato) + adaptive default.

## Confirmation Dialogs (planned -- Phase 5c)

- **Destructive actions** (delete, cancel): inline y/n confirmation in the
  bottom bar. Press `d` -> bar shows "Delete backup X? [y/n]" -> `y` confirms,
  any other key cancels. (Currently actions execute immediately.)
- **Parameterized actions** (start backup): `huh` form with type, compression,
  and profile selection. (Currently starts a logical backup with defaults.)
- **Future**: `--readonly` flag disables all mutation actions.

## Project Structure

```
pbmate/
├── main.go                    # Entry point: flags, SDK client, tea.Program
├── internal/
│   └── tui/
│       ├── app.go             # Root model: Init, Update, View, tab routing
│       ├── keys.go            # Key bindings (global + per-view keymaps)
│       ├── styles.go          # Lipgloss styles, colors, borders
│       ├── theme.go           # Theme definitions (Catppuccin + adaptive)
│       ├── poll.go            # Tick intervals and tick command
│       ├── data.go            # Data fetching commands and message types
│       ├── render.go          # Shared rendering helpers
│       ├── overview.go        # Overview tab (cluster tree + detail + status + logs)
│       ├── backups.go         # Backups tab (list + detail with sub-tabs)
│       ├── restores.go        # Restores tab (planned -- Phase 5d)
│       └── config.go          # Config tab (planned -- Phase 5d)
```

## Dependencies

- `charmbracelet/bubbletea` -- framework
- `charmbracelet/bubbles` -- help, key, spinner, viewport, progress
- `charmbracelet/lipgloss` -- styling and layout
- `charmbracelet/huh` -- forms (planned -- Phase 5c, not yet in go.mod)
- `jcechace/pbmate/sdk/v2` -- PBMate SDK
