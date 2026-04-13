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
│  PBMate  production  [1:Overview]  2:Backups  3:Config            │
├──────────────────────────┬───────────────────────────────────────┤
│                          │                                       │
│    Left panel            │         Right panel (detail)          │
│    (navigable list)      │         (selected item info)          │
│                          │                                       │
├──────────────────────────┴───────────────────────────────────────┤
│ PITR:on  Op:none  15:04 │ ↑↓:nav  s:backup  ?:help  q:quit(2x)   │
└──────────────────────────────────────────────────────────────────┘
```

Three zones: **header** (tab bar with optional context name), **content**
(tab-specific layout), **bottom bar** (status HUD left, context-sensitive hints
right). The context name appears in muted style after the title when a named
context is used; hidden for direct `--uri` connections.

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
appear at the top of the list, separated from backups by a muted "── Backups ──"
section label.

Right panel: full backup/restore metadata, timestamps, compression, errors.

Actions: `s` start backup (quick confirm), `S` custom backup (full wizard with
type, compression, profile), `R` restore wizard (2-step: target → options),
`r` restore from selected item (snapshot or PITR based on cursor).
After successful restore dispatch, the tab auto-switches to the Restores list.

### 3. Config

Left panel: two-column list (name + storage type). Main config entry uses
bold-on-select styling (no icon). A muted "── Profiles ──" section label
separates Main from storage profiles.
Right panel: detail (storage settings, PITR config, compression, raw YAML with
syntax highlighting).

Actions: `C` set config (3-step wizard: target form → file picker → optional
override confirm), `c` set config for selected item (pre-filled from cursor),
`R` resync storage, `r` resync selected.

### Future (not yet implemented)

- Detail panel sub-tabs (`[`/`]`) for Backups (Info, Replicas, Logs)
- `/` filter in list views
- Connection reconnect on failure

## Readonly Mode

When `--readonly` is active (via CLI flag, context override, or global config),
all mutation actions are disabled. The TUI operates as a monitoring-only viewer.

### Behavior

- **Disabled keys**: `s`, `S` (start backup), `X` (cancel backup), `d` (delete),
  `R`, `r` (restore) on the Backups tab, `C`, `c` (set config), `R`, `r` (resync),
  `d` (delete profile) on the Config tab. These keys are silently ignored.
- **Help overlay**: Mutation entries are omitted from the help overlay. Only
  navigation, view-only, and general bindings are shown.
- **Bottom bar badge**: A bold yellow `READONLY` badge is displayed as the first
  item in the bottom bar status zone (left side), before PITR/Op/time indicators.
  This makes the mode immediately visible on all tabs.

```
│ READONLY  PITR:on  Op:none  15:04 │ ↑↓:nav  tab:toggle  ?:help  q:quit(2x) │
```

### Resolution

Readonly is resolved with full precedence at startup:
CLI `--readonly`/`--no-readonly` flag > context-level `readonly` override >
global `readonly` config field > `false` (default).

## Keybindings

### Global
- `q` (double-press) — quit (first press shows hint, second quits within 2s)
- `ctrl+c` — quit immediately (bypasses double-press guard)
- `1`-`3` — jump to tab
- `?` — toggle full help overlay
- `esc` — back / close overlay / clear filter
- `s` / `S` — start backup (quick confirm / full wizard)
- `X` — cancel running backup
- `d` — delete (backup on Backups tab, profile on Config tab; on PITR timeline opens bulk delete with PITR preselected)
- `p` — toggle PITR (enable/disable with confirm overlay)

### Navigation (within panels)
- `up`/`down` or `k`/`j` — move selection / scroll in focused panel
- `]` / `[` — cycle focus to next/previous panel

### Overview-specific
- `space` / `enter` — expand/collapse RS group (when cluster panel focused)
- `f` — toggle log follow mode
- `w` — toggle log word-wrap
- `l` — open log filter form (severity, replica set, event type)

### Backups-specific
- `tab` — toggle between Backups and Restores list
- `R` / `r` — restore (generic wizard / from selected item)
- `D` — bulk delete (form overlay: Backups or PITR, preset/custom date, type/profile filters)
- `space` / `enter` — expand/collapse profile group

### Config-specific
- `C` / `c` — set config (generic / for selected item)
- `R` / `r` — resync storage (generic / for selected item)
- `e` — edit selected config/profile in `$EDITOR`
- `tab` — toggle between preview and YAML view

## Bottom Bar

Single merged bar replacing the previous two-bar (status + help) design.

```
│ PITR:on  Op:backup(●)  15:04 │ s:backup  d:delete  X:cancel  ?:help  q:quit(2x) │
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
Init() -> tea.Batch(tea.RequestWindowSize, tea.RequestBackgroundColor, connectCmd(mongoURI))
```

Three commands fire in parallel: `tea.RequestWindowSize` triggers a
`WindowSizeMsg` so we know terminal dimensions, `tea.RequestBackgroundColor`
requests the terminal background so the adaptive default theme can resolve, and
`connectCmd` runs `sdk.NewClient()` in a goroutine, returning
`connectMsg{client, err}`. The TUI renders immediately with "Connecting..."
while the SDK connects in the background.

### Connection Retry

If the initial connection fails, the TUI retries automatically with exponential
backoff (2s, 4s, 8s, 16s, 30s cap). The retry chain uses `reconnectMsg`:

```
connectCmd(uri)
  └─> connectMsg{err}  → flashErr shows error + retry delay
        └─> reconnectCmd(delay)
              └─> reconnectMsg  → flashErr cleared
                    └─> connectCmd(uri)
                          └─> connectMsg{ok}  → normal polling starts
```

Each connection attempt uses a 10s timeout (`sdk.WithConnectTimeout`) so
the user gets feedback quickly rather than waiting for the driver's 30s default.

Bottom bar status during retries:
- First attempt: `Connecting...` (up to 10s)
- After failure: `Connection failed (retry in 2s)` (red)
- During retry: `Connecting... (attempt 3)` (yellow)

The user can quit (`q` double-press or `Ctrl+C`) at any time. Once connected, retry state is
cleared. Mid-session disconnects are handled by the MongoDB driver's built-in
automatic reconnection — PBMate does not re-create the SDK client.

### Polling Chain

See `docs/agents/tui-conventions.md` Polling Pattern section for the chained
single-shot timer design and adaptive intervals (2s active, 10s idle).

### Message Types

| Message             | Source               | Purpose                           |
|---------------------|----------------------|-----------------------------------|
| `tea.WindowSizeMsg` | BubbleTea runtime    | Terminal resized                  |
| `tea.KeyPressMsg`   | BubbleTea runtime    | User keypress                     |
| `connectMsg`        | `connectCmd`         | SDK client ready or error         |
| `reconnectMsg`      | `reconnectCmd`       | Retry delay elapsed, reconnect    |
| `tickMsg`           | `tickCmd`            | Timer fired, time to fetch        |
| `overviewDataMsg`   | `fetchOverviewCmd`   | Overview data arrived             |
| `backupsDataMsg`    | `fetchBackupsCmd`    | Backup list arrived               |
| `restoresDataMsg`   | `fetchRestoresCmd`   | Restore list arrived              |
| `configDataMsg`     | `fetchConfigCmd`     | Config data arrived               |
| `profileYAMLMsg`    | `fetchProfileYAMLCmd`| Profile YAML content arrived      |
| `actionResultMsg`   | action commands      | Any action completed (backup, restore, resync, config) |
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
- The default theme resolves to explicit light/dark palettes via
  `LookupTheme(name, isDark)` after `tea.BackgroundColorMsg` arrives. Named
  flavors use fixed `lipgloss.Color` values for exact color matching.
- `huh` form themes are built per-flavor from catppuccin-go, not from huh's
  built-in `ThemeCatppuccin()` (which is adaptive and ignores the chosen flavor).

## Form Overlays

Actions that need user input render centered `huh` form overlays on top of the
current tab content. All key input is routed to the form while it's open; data
polling continues in the background.

- **Destructive actions** (delete, cancel): confirm overlay with
  affirmative/negative buttons. Chain-aware delete shows the base backup name
  and total chain count for incremental backups.
- `esc` or `q` dismisses any open overlay.

### Form Redesign

Design principles:
- **Single-screen over multi-step.** All fields visible at once. No wizard pages.
- **Smart defaults eliminate fields.** Hide expert options behind a collapsible
  "Advanced" section toggled with `space`.
- **Inline selectors for 2-3 options.** Use `Select.Inline(true)` instead of
  full scrollable lists.
- **Validate inline.** Show errors on the field, not at submit time.
- **Context from selection.** Show what the user selected (backup metadata,
  timeline range) in the form header.
- **Adaptive width.** Form overlay width scales with terminal width instead of
  a fixed 40-char constant.

#### Quick Backup (`s`)

Single-screen confirm. Shows target profile name dynamically. If an active
incremental chain exists, mentions "Continues existing chain."

#### Custom Backup (`S`)

Flat single-screen form:
- **Type**: Inline select (`Logical` / `Incremental`)
- **Profile**: Select with filtering for many profiles
- **Compression**: Select with "Server default" pre-selected
- **Namespaces**: Input, only shown for logical (`WithHideFunc`)
- **Incremental options** (when type=incremental): Inline confirm "Start new
  chain?" with description. Shows which chain will be continued if "No."
- **Advanced section**: Collapsed by default (`space` to expand).
  Contains Parallel Collections.
- **Submit**: Clear "Start Backup" button via `Note.NextLabel`.

#### Restore Wizard (`R` — generic)

Two-step wizard. Step 1 selects the restore target:
- **Type**: Inline selector (Snapshot / PITR). PITR only available when timelines exist.
- **Profile** (Snapshot mode): Inline selector filtering the backup list by storage profile.
  Options built from distinct profiles among completed backups. Defaults to Main.
- **Backup** (Snapshot mode): Dropdown of completed backups matching the selected profile.
- **Restore to** (PITR mode): Preset selector with computed offsets + Custom input.
- **Confirm**: "Next" proceeds to Step 2 (options form).

Step 2 is the same options form as `r` (scope, tuning, confirm).

#### Snapshot Restore (`r` on backup)

Single-screen with backup context header (name, type, status, size).
For incremental backups, shows chain position.
- **Namespaces**: Input
- **Users & Roles**: Inline confirm, only shown when selective
- **Advanced**: Collapsed. Contains Parallel Collections, Insertion Workers.
- **Submit**: "Restore" button.

#### PITR Restore (`r` on timeline)

Single-screen with timeline range displayed.
- **Restore to**: Select with computed presets:
  `Latest (15:30:00)`, `-5 min`, `-30 min`, `-1 hour`, `Custom...`.
  "Custom" reveals a text input for manual timestamp.
- **Namespaces**: Input
- **Users & Roles**: Inline confirm, only shown when selective
- **Advanced**: Collapsed.
- **Submit**: "Restore" button.

#### New Operations

- **Resync** (`R`): Inline scope selector (Main / Profile / All). Profile
  select shown conditionally.
- **Delete** (`d`): Global key — deletes backup on Backups tab, profile on Config tab. Confirm overlay.

## CLI

PBMate uses [kong](https://github.com/alecthomas/kong) for CLI parsing.
The CLI struct is defined in `main.go` using struct tags.

### Commands

```
pbmate                                  # default: starts TUI with current context
pbmate tui                              # explicit: same as above
pbmate tui --uri <uri>                  # explicit URI, bypasses context
pbmate tui --context <name>             # one-time context override
pbmate tui --theme <name>               # one-time theme override
pbmate tui --readonly                   # one-time readonly override
pbmate --config <path>                  # global: custom config file path

pbmate context list                     # list contexts (* = current)
pbmate context current                  # print current context name + URI
pbmate context use <name>               # switch active context (writes config)
pbmate context add <name> --uri=<uri>   # add context (optional: --theme, --readonly)
pbmate context remove <name>            # remove context

pbmate config show                      # print full config as YAML
pbmate config show --context=<name>     # print single context as YAML
pbmate config set <key> <value>         # set a global config value
pbmate config set <key> <val> --context=<name>  # set per-context override
pbmate config unset <key>               # reset global value to default
pbmate config unset <key> --context=<name>      # remove per-context override (inherit)
pbmate config path                      # print resolved config file path
```

`pbmate` with no subcommand runs `pbmate tui` via kong's `default:"withargs"`.

### Flag Precedence

```
CLI flag  >  context setting  >  global config  >  built-in default
```

For theme: `--theme mocha` > `contexts.staging.theme` > top-level `theme` > `"default"`.
For readonly: `--readonly` > `contexts.staging.readonly` > top-level `readonly` > `false`.

If no URI is available (no `--uri`, no context, no `current-context`), print a
helpful error directing the user to `pbmate context add`.

## Configuration

### File Location

```
$XDG_CONFIG_HOME/pbmate/config.yaml    # if XDG_CONFIG_HOME is set
~/.config/pbmate/config.yaml           # fallback (XDG default)
```

Overridable with `--config <path>` or `PBMATE_CONFIG` env var.

### File Format

Single YAML file containing both global settings and connection contexts:

```yaml
theme: mocha
readonly: false

current-context: production

contexts:
  production:
    uri: mongodb://prod-host:27017
  staging:
    uri: mongodb://staging-host:27017
    theme: latte
    readonly: true
  local:
    uri: mongodb://localhost:27017
```

### Config Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `theme` | string | `"default"` | Global color theme |
| `readonly` | bool | `false` | Global readonly mode |
| `current-context` | string | `""` | Active context name |
| `contexts` | map | `{}` | Named connection contexts |

### Context Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `uri` | string | Yes | MongoDB connection URI |
| `theme` | string | No | Theme override for this context |
| `readonly` | *bool | No | Readonly override (nil = inherit global) |

### Package Layout

```
internal/config/
├── config.go           # AppConfig struct, Context struct, Load(), Save(), FormatYAML(), XDG path
├── config_test.go      # Round-trip, merge, validation tests
├── field.go            # SetByPath(), GetByPath(), UnsetByPath() — reflection helpers for config set/unset
└── field_test.go       # Table-driven tests for field operations
```

`internal/config` is shared between CLI commands and TUI startup. It has no
dependency on the SDK or TUI packages.
