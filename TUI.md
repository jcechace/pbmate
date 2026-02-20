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

Left panel: scrollable backup tree with `tab` toggle between Backups and
Restores. Compact timestamps (drop seconds). Backups are grouped by storage
profile with collapsible headers. Incremental backups are grouped into chains
under their base (base shows `⌂` icon, children indented). PITR timelines
appear at the top of the list.

Right panel: full backup/restore metadata, timestamps, compression, errors.

Actions: `s` start backup (quick confirm), `S` custom backup (full wizard with
type, compression, profile), `d` delete (overlay confirmation, chain-aware for
incrementals), `c` cancel running backup.

Future: detail panel sub-tabs (Info, Replicas, Logs), `/` filter.

### 3. Config

Left panel: main config + storage profiles list.
Right panel: detail (storage settings, PITR config, compression, raw YAML).

Read-only for MVP. Later: `e` edit via YAML, `p` add profile.

## Keybindings

### Global
- `q` / `ctrl+c` -- quit
- `1`-`3` -- jump to tab
- `?` -- toggle full help overlay
- `esc` -- back / close overlay / clear filter
- `s` -- start backup (quick confirm)
- `S` -- custom backup (full wizard)
- `c` -- cancel running backup

### Navigation (within panels)
- `up`/`down` or `k`/`j` -- move selection / scroll in focused panel
- `]` / `[` -- cycle focus to next/previous panel

### Overview-specific
- `space` / `enter` -- expand/collapse RS group (when cluster panel focused)
- `f` -- toggle log follow mode
- `w` -- toggle log word-wrap

### Backups-specific
- `tab` -- toggle between Backups and Restores list
- `d` -- delete backup (overlay confirmation)
- `space` / `enter` -- expand/collapse profile group

### Future
- `/` -- filter/search in list views
- `--readonly` flag to disable all mutation actions

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
- **Data ownership**: Sub-models own their data. Root Model has no data fields;
  it reads `m.overview.data` directly for the status bar HUD and adaptive
  polling. This avoids fragile sync-back patterns between root and sub-models.
- **Log panel**: auto-refreshes every poll cycle in normal mode; streams via
  `Logs.Follow()` goroutine in follow mode. Follow-mode log entries are
  preserved across poll cycles by `overview.setData()`.
- **Stable cursor**: selection tracked by item identity (agent node name,
  backup name), not by list index. Prevents cursor jumping on data refresh.

## Message Flow Architecture

PBMate's TUI follows the Elm Architecture enforced by BubbleTea: a
unidirectional data flow where `Update` is the only place state changes.

### The Core Loop

```
   ┌─────────────────────────────────────────┐
   │                                         │
   ▼                                         │
 View(Model) ──render──▶ Terminal            │
                                             │
 User input / timer / async result           │
   │                                         │
   ▼                                         │
 Update(Model, Msg) ──▶ (new Model, Cmd) ────┘
```

1. **View** -- pure function: reads Model, returns a string. No side effects.
2. **Msg** arrives -- a user keypress, window resize, timer tick, or the result
   of an async command.
3. **Update** -- takes current Model + Msg, returns a new Model and optionally
   a Cmd.
4. **Cmd** -- a `func() Msg`. BubbleTea runs it in a goroutine and feeds the
   resulting Msg back into Update. This is how side effects happen.

### Startup Sequence

```
Init() → tea.Batch(tea.WindowSize(), connectCmd(mongoURI))
```

Two commands fire in parallel: `tea.WindowSize()` triggers a `WindowSizeMsg` so
we know terminal dimensions, and `connectCmd` runs `sdk.NewClient()` in a
goroutine, returning `connectMsg{client, err}`. The TUI renders immediately
with "Connecting..." while the SDK connects in the background.

### Polling Chain

PBMate doesn't use a persistent ticker goroutine. Instead it chains single-shot
timers, with each data response scheduling the next tick:

```
connectMsg (client ready)
  └─▶ tickCmd(0)                           immediate tick
        └─▶ fetchOverviewCmd + fetchBackupsCmd
              └─▶ overviewDataMsg
                    └─▶ tickCmd(pollInterval)   schedule next tick
                          └─▶ (waits 2s or 10s)
                                └─▶ tickMsg
                                      └─▶ fetch...    (cycle repeats)
```

The `overviewDataMsg` handler decides the next interval: 2s if operations are
running (activeInterval), 10s if idle (idleInterval). The chain is self-healing:
if a fetch errors, we still schedule the next tick.

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
| `backupActionMsg`   | action commands      | Backup/delete/cancel completed    |
| `logFollowMsg`      | `nextLogCmd`         | New log entries from follow       |
| `logFollowDoneMsg`  | follow goroutine     | Follow channel closed             |
| `backupFormReadyMsg`| `fetchProfilesCmd`   | Profiles loaded, form can open    |
| `deleteConfirmMsg`  | `requestDeleteConfirm`| Confirm overlay needed           |

### Message Routing Priority

`Update` has a strict priority order:

1. **Data/system messages first** -- WindowSizeMsg, connectMsg, tickMsg, all
   data messages, action messages, log follow messages, form ready messages.
   These are handled regardless of overlay state, so polling continues while
   forms are open.

2. **Key messages** -- routed based on overlay state:
   - `backupForm != nil` → `updateBackupForm` (huh form gets all keys)
   - `confirmForm != nil` → `updateConfirmForm`
   - `showHelp == true` → only `?`/`esc` to dismiss
   - otherwise → `updateKeys` → global bindings, then forward to active tab

3. **Non-key messages with active form** -- forwarded to the form (huh
   internals like cursor blink timers).

### Sub-Model Pattern

Each tab has its own sub-model (overviewModel, backupsModel). They are plain
structs with methods, not tea.Model implementations:

- `update(msg tea.KeyMsg, keys globalKeyMap) tea.Cmd` -- handles keypresses
- `view(w, h int) string` -- renders the tab content
- `resize(w, h int)` -- precomputes viewport dimensions
- `setData(...)` -- receives fresh data from fetch commands

The root Model.Update calls sub-model methods directly. Sub-models never see
non-key messages -- the root handles all data routing.

### Value Receiver and Pointer Semantics

BubbleTea requires value receivers (`func (m Model) Update`). Mutations inside
Update only affect the local copy; the modified `m` is returned.

Sub-models are embedded by value in Model. Pointers inside the model (like
`m.client`, `m.overview.client`) are shared -- mutations through them affect
the real state.

The `*Styles` pattern: Model owns `styles Styles` by value, sub-models hold
`*Styles` pointing into it. Since sub-models are embedded in Model, the pointer
remains valid across Update cycles.

### The Cmd Contract

A Cmd is `func() Msg`. Key rules:

- BubbleTea runs it in a separate goroutine.
- It must return exactly one Msg.
- It must not access or mutate the Model (no closures over `m`).
- It can close over immutable values (client pointer, channel, string).
- `tea.Batch(cmds...)` runs multiple commands concurrently.
- Returning nil from Update means "no command."

### Log Follow: Channel Bridge

The log follow mode bridges the SDK's channel-based API into BubbleTea's
message model:

```go
nextLogCmd = func() tea.Msg {
    entries, ok := <-followCh
    if !ok { return logFollowDoneMsg{} }
    return logFollowMsg{entries}
}
```

Each logFollowMsg handler calls nextLogCmd() again, creating a chain that
drains the channel one batch at a time. This is the standard BubbleTea pattern
for bridging blocking I/O.

**Known issue**: a goroutine leak occurs when follow is stopped between
dispatching nextLogCmd and its message arriving -- the goroutine blocks on the
channel read forever because nobody closes the channel. This needs a
cancellable context or a done-channel to unblock it.

## Styling

- **Status colors**: green = done/ok, red = error, yellow = running/in-progress,
  gray = cancelled/stale.
- **Status indicators**: `●` (filled green) = healthy/done, `●` (filled red) =
  error, `○` (empty/dim) = stale/cancelled. Shape + color for accessibility.
- Bordered panels with lipgloss `RoundedBorder` and **titled top borders**:
  `╭─ Cluster ─────╮`. Title color matches the border color (primary when
  focused, subtle when unfocused).
- Adaptive colors (`lipgloss.AdaptiveColor`) for light/dark terminals.
- Compact, information-dense -- no wasted space.
- Catppuccin theme support (Mocha/Latte/Frappe/Macchiato) + adaptive default.

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
- `esc` or `q` dismisses any open overlay.

## Project Structure

```
pbmate/
├── main.go                       # Entry point: flags, SDK client, tea.Program
├── internal/
│   └── tui/
│       ├── app.go                # Root model: Init, Update, View, tab routing, bottom bar
│       ├── overview.go           # Overview tab: layout, focus, follow state, status panel
│       ├── cluster_panel.go      # Cluster tree + detail viewports (extracted from overview)
│       ├── backups.go            # Backups tab: list + detail + restore toggle
│       ├── backup_chain.go       # Pure chain logic: grouping, ordering, resolution
│       ├── backup_chain_test.go  # Tests for chain logic
│       ├── backup_form.go        # Quick/full backup forms + confirm overlay + renderFormOverlay
│       ├── log_panel.go          # Reusable log viewer: viewport, pin/wrap/follow
│       ├── data.go               # Data fetching commands, message types, action commands
│       ├── render.go             # Shared rendering: titled panels, cursor list, status dots
│       ├── layout.go             # Layout constants and helpers: splits, panel type
│       ├── keys.go               # Key bindings (global + per-tab keymaps)
│       ├── styles.go             # Lipgloss styles derived from theme
│       ├── theme.go              # Theme definitions (Catppuccin + adaptive)
│       └── poll.go               # Tick intervals and tick command
```

## Dependencies

- `charmbracelet/bubbletea` -- framework
- `charmbracelet/bubbles` -- key, viewport
- `charmbracelet/lipgloss` -- styling and layout
- `charmbracelet/huh` -- form overlays (backup wizard, confirm dialogs)
- `jcechace/pbmate/sdk/v2` -- PBMate SDK
