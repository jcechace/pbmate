# TUI Design

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
│  PBMate   [1:Overview]  2:Backups  3:Config                      │
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
  `▸ rs2 ●●● (3)` — all healthy; `▸ rs3 ●●○ (3)` — one stale.
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

Left panel: scrollable backup tree with `tab` toggle between Backups and
Restores. Each backup shows its name (RFC 3339 timestamp). Backups are grouped
by storage profile with collapsible headers. Incremental backups are grouped into chains
under their base (base shows `⌂` icon, children indented). PITR timelines
appear at the top of the list.

Right panel: full backup/restore metadata, timestamps, compression, errors.

Actions: `s` start backup (quick confirm), `S` custom backup (full wizard with
type, compression, profile), `r` context-sensitive restore (on a backup =
snapshot restore, on a PITR timeline = PITR restore with auto-selected base
backup), `d` delete (overlay confirmation, chain-aware for incrementals),
`c` cancel running backup. After successful restore dispatch, the tab
auto-switches to the Restores list.

### 3. Config

Left panel: main config + storage profiles list.
Right panel: detail (storage settings, PITR config, compression, raw YAML with
syntax highlighting).

Actions: `e` apply YAML configuration (file picker overlay — works even when no
main config exists yet), `p` create new storage profile (name form + file
picker).

### Future (not yet implemented)

- Detail panel sub-tabs (`[`/`]`) for Backups (Info, Replicas, Logs)
- `/` filter in list views
- `--readonly` flag to disable all mutation actions
- Connection reconnect on failure

## Keybindings

### Global
- `q` / `ctrl+c` — quit
- `1`-`3` — jump to tab
- `?` — toggle full help overlay
- `esc` — back / close overlay / clear filter
- `s` — start backup (quick confirm)
- `S` — custom backup (full wizard)
- `c` — cancel running backup

### Navigation (within panels)
- `up`/`down` or `k`/`j` — move selection / scroll in focused panel
- `]` / `[` — cycle focus to next/previous panel

### Overview-specific
- `space` / `enter` — expand/collapse RS group (when cluster panel focused)
- `f` — toggle log follow mode
- `w` — toggle log word-wrap

### Backups-specific
- `tab` — toggle between Backups and Restores list
- `r` — restore (on backup = snapshot, on PITR timeline = point-in-time)
- `d` — delete backup (overlay confirmation)
- `space` / `enter` — expand/collapse profile group

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
- 6-8 most important hints; full reference via `?` overlay.

## Polling & Data Flow

- **Tick-based polling** with adaptive intervals:
  - Idle: 10s
  - Active operation detected: 2s
- Overview always fetches cluster + status data (needed for bottom bar HUD).
- Tab-specific data fetched only for the active tab.
- **Log panel**: auto-refreshes every poll cycle in normal mode; streams via
  `Logs.Follow()` goroutine in follow mode. Follow-mode log entries are
  preserved across poll cycles by `overview.setData()`.

## Message Flow Architecture

### Startup Sequence

```
Init() -> tea.Batch(tea.WindowSize(), connectCmd(mongoURI))
```

Two commands fire in parallel: `tea.WindowSize()` triggers a `WindowSizeMsg` so
we know terminal dimensions, and `connectCmd` runs `sdk.NewClient()` in a
goroutine, returning `connectMsg{client, err}`. The TUI renders immediately
with "Connecting..." while the SDK connects in the background.

### Polling Chain

See `docs/agents/tui-conventions.md` Polling Pattern section for the chained
single-shot timer design and adaptive intervals (2s active, 10s idle).

### Message Types

| Message             | Source               | Purpose                           |
|---------------------|----------------------|-----------------------------------|
| `tea.WindowSizeMsg` | BubbleTea runtime    | Terminal resized                  |
| `tea.KeyMsg`        | BubbleTea runtime    | User keypress                     |
| `connectMsg`        | `connectCmd`         | SDK client ready or error         |
| `tickMsg`           | `tickCmd`            | Timer fired, time to fetch        |
| `overviewDataMsg`   | `fetchOverviewCmd`   | Overview data arrived             |
| `backupsDataMsg`    | `fetchBackupsCmd`    | Backup list arrived               |
| `restoresDataMsg`   | `fetchRestoresCmd`   | Restore list arrived              |
| `configDataMsg`     | `fetchConfigCmd`     | Config data arrived               |
| `profileYAMLMsg`    | `fetchProfileYAMLCmd`| Profile YAML content arrived      |
| `backupActionMsg`   | action commands      | Backup/delete/cancel completed    |
| `restoreActionMsg`  | `startRestoreCmd`    | Restore action completed          |
| `configActionMsg`   | config commands      | Config apply/profile completed    |
| `logFollowMsg`      | `nextLogCmd`         | New log entries from follow       |
| `logFollowDoneMsg`  | follow goroutine     | Follow channel closed             |
| `backupFormReadyMsg`| `fetchProfilesCmd`   | Profiles loaded, form can open    |
| `deleteCheckRequest`| backups sub-model    | CanDelete pre-check needed        |
| `canDeleteMsg`      | `canDeleteCmd`       | Pre-check result, show confirm    |
| `restoreRequest`    | backups sub-model    | Restore form overlay needed       |

## Styling

- **Status colors**: green = done/ok, red = error, yellow = running/in-progress,
  gray = cancelled/stale.
- **Status indicators**: `●` (filled green) = healthy/done, `●` (filled red) =
  error, `○` (empty/dim) = stale/cancelled. Shape + color for accessibility.
- Bordered panels with lipgloss `RoundedBorder` and **titled top borders**:
  `╭─ Cluster ─────╮`. Title color matches the border color (primary when
  focused, subtle when unfocused).
- Compact, information-dense — no wasted space.
- Catppuccin theme support (Mocha/Latte/Frappe/Macchiato) + adaptive default.
- The default theme uses `lipgloss.AdaptiveColor` for light/dark terminals.
  Named flavors use hardcoded `lipgloss.Color` for exact color matching.
- `huh` form themes are built per-flavor from catppuccin-go, not from huh's
  built-in `ThemeCatppuccin()` (which is adaptive and ignores the chosen flavor).

## Form Overlays

Actions that need user input render centered `huh` form overlays on top of the
current tab content. All key input is routed to the form while it's open; data
polling continues in the background.

- **Destructive actions** (delete, cancel): confirm overlay with
  affirmative/negative buttons. Chain-aware delete shows the base backup name
  and total chain count for incremental backups.
- **Quick backup** (`s`): single-step confirm overlay. Press `c` to switch to
  the full wizard.
- **Custom backup** (`S`): multi-step wizard with type, compression, and profile
  selection. Profiles are fetched asynchronously before the form opens.
- **Restore** (`r`): context-sensitive restore forms:
  - On a completed backup: snapshot restore form (namespaces, conditional
    users-and-roles, performance tuning).
  - On a PITR timeline: PITR restore form (target time pre-filled from
    timeline end, namespaces, conditional users-and-roles, performance tuning).
    Base backup is auto-selected from cached data (latest completed backup
    before target time).
  - Users-and-roles question is only shown for selective (namespace-filtered)
    restores, since full restores always include them.
- `esc` or `q` dismisses any open overlay.
