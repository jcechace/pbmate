# TUI Conventions

## Architecture

PBMate's TUI uses BubbleTea, which enforces the Elm Architecture: a unidirectional data flow where `Update` is the only place state changes.

```
   ┌─────────────────────────────────────────┐
   │                                         │
   ▼                                         │
 View(Model) ──render──> Terminal            │
                                             │
 User input / timer / async result           │
   │                                         │
   ▼                                         │
 Update(Model, Msg) ──> (new Model, Cmd) ────┘
```

1. **View** — pure function: reads Model, returns a string. No side effects.
2. **Msg** arrives — a user keypress, window resize, timer tick, or async command result.
3. **Update** — takes current Model + Msg, returns new Model and optionally a Cmd.
4. **Cmd** — a `func() Msg`. BubbleTea runs it in a goroutine and feeds the resulting Msg back into Update. This is how side effects happen.

### The Cmd Contract

- BubbleTea runs Cmds in a separate goroutine.
- A Cmd must return exactly one Msg.
- A Cmd must not access or mutate the Model (no closures over `m`).
- A Cmd can close over immutable values (client pointer, channel, string).
- `tea.Batch(cmds...)` runs multiple commands concurrently.
- Returning nil from Update means "no command."

## Sub-Model Pattern

Each tab has its own sub-model. They are **plain structs with methods**, not `tea.Model` implementations:

- `update(msg tea.KeyMsg, keys globalKeyMap) tea.Cmd` — handles keypresses
- `view(w, h int) string` — renders the tab content
- `resize(w, h int)` — precomputes viewport dimensions
- `setData(...)` — receives fresh data from fetch commands

The root `Model.Update` calls sub-model methods directly. Sub-models never see non-key messages — the root handles all data routing.

## Value Receiver and Pointer Semantics

BubbleTea requires value receivers (`func (m Model) Update`). Mutations inside Update only affect the local copy; the modified `m` is returned.

Sub-models are embedded by value in Model. Pointers inside the model (like `m.client`, `m.overview.client`) are shared — mutations through them affect the real state.

### The `*Styles` Pattern

Model owns `styles Styles` by value. Sub-models hold `*Styles` pointing into it. Since sub-models are embedded in Model, the pointer remains valid across Update cycles.

## Data Ownership

Sub-models own their data. Root Model has no data fields; it reads `m.overview.data` directly for the status bar HUD and adaptive polling. This avoids fragile sync-back patterns between root and sub-models.

## Polling Pattern

PBMate doesn't use a persistent ticker goroutine. Instead it chains single-shot timers, with each data response scheduling the next tick:

```
connectMsg (client ready)
  └─> tickCmd(0)                           immediate tick
        └─> fetchOverviewCmd + fetchBackupsCmd
              └─> overviewDataMsg
                    └─> tickCmd(pollInterval)   schedule next tick
```

Adaptive intervals: 2s if operations are running, 10s if idle. The chain is self-healing — if a fetch errors, the next tick is still scheduled.

Each fetch command (`fetchOverviewCmd`, `fetchBackupsCmd`, `fetchConfigCmd`) uses `errgroup` to run multiple SDK calls concurrently (e.g., overview fetches agents, ops, PITR status, timelines, and latest backup in parallel). Results are collected and returned as a single message.

## Stable Cursor Pattern

Selection is tracked by item identity (backup name, agent node name), not by list index. When data refreshes and the list order potentially changes, the cursor resolves to the correct item by matching identity, preventing cursor jumps.

## Log Follow: Channel Bridge

The log follow mode bridges the SDK's channel-based API into BubbleTea's message model:

```go
nextLogCmd = func() tea.Msg {
    entries, ok := <-followCh
    if !ok { return logFollowDoneMsg{} }
    return logFollowMsg{entries}
}
```

Each `logFollowMsg` handler calls `nextLogCmd()` again, creating a chain that drains the channel one batch at a time. Follow sessions use a monotonic session ID so stale messages from a previous session are discarded. The `nextLogCmd` closure selects on both the entries channel and a cancellation context to avoid goroutine leaks.

## Form Overlays

Actions that need user input render centered `huh` form overlays. All key input routes to the form while it's open; data polling continues in the background.

All overlays implement the `formOverlay` interface (overlay.go):
- Confirm overlays for destructive actions (delete, cancel)
- Quick/full backup wizard
- Two-step restore wizard (target selection → options) and context-sensitive restore forms
- Resync form with target selector (Main / Profile / All)
- Bulk delete form with target (Backups/PITR), preset/custom date, type/profile filters
- Log filter form with severity, replica set, and event type selectors
- Set config wizard (target form → file picker → optional override confirm)
- Profile name form, file picker for config

Shared overlay helpers in `backup_form.go` reduce boilerplate across all overlay types:
- `dismissOverlay` — check for esc/quit key
- `updateFormModel` — forward message to huh.Form and write back pointer
- `initFormWithAdvance` — Init + optional NextField for dynamic rebuilds

`esc` or `q` dismisses any open overlay.

## Message Routing Priority

`Update` has a strict priority order:

1. **Data/system messages first** — WindowSizeMsg, connectMsg, tickMsg, all data messages, action messages, log follow messages, form ready messages. Handled regardless of overlay state, so polling continues while forms are open.
2. **Overlay routing** — if `activeOverlay != nil`, remaining messages forwarded to overlay via `formOverlay` interface.
3. **Key messages without overlay** — routed through `updateKeys`: help overlay -> global bindings -> active tab's sub-model.

## Coding Conventions

- Prefer `bubbles` components (viewport, table, list, etc.) over hand-rolled rendering. Less custom code is better.
- Form overlays use `huh` library with per-flavor Catppuccin themes (not `huh.ThemeCatppuccin()` which is adaptive and ignores the chosen flavor).
- Status colors: green = done/ok, red = error, yellow = running, gray = cancelled/stale.
- Status indicators: `●` (filled) = healthy/done or error, `○` (dim) = stale/cancelled. Shape + color for accessibility.
- Bordered panels with lipgloss `RoundedBorder` and titled top borders: `╭─ Title ─────╮`.
- Compact, information-dense — no wasted space.

## TUI File Structure

```
internal/tui/
├── app.go              # Root model: tab routing, bottom bar, global keys
├── overview.go         # Overview tab
├── cluster_panel.go    # Cluster tree + detail viewports
├── backups.go          # Backups tab (list + detail, tab toggles backups/restores)
├── backup_chain.go     # Chain grouping for display (separate from sdk/backup_chain.go which is domain logic)
├── backup_form.go      # Backup forms + shared overlay helpers (dismissOverlay, updateFormModel, initFormWithAdvance)
├── restore_form.go     # Restore forms (snapshot, PITR, target wizard with profile filter)
├── resync_form.go      # Resync form (Main / Profile / All)
├── bulk_delete_form.go # Bulk delete form (Backups/PITR, preset/custom date, type/profile)
├── log_filter_form.go  # Log filter form (severity, replica set, event type)
├── config.go           # Config tab
├── config_form.go      # Set config form (target → file picker → confirm)
├── overlay.go          # formOverlay interface + all overlay types
├── log_panel.go        # Reusable log viewer component
├── data.go             # Data fetching commands, message types, actionResultMsg, firstErrCollector
├── render.go           # Shared rendering primitives (panels, cursor list, help, formatting)
├── detail_render.go    # Domain-specific detail renderers (backup, restore, status/agent indicators)
├── layout.go           # Layout helpers, dimension math, panelBorderColor
├── keys.go             # Key bindings (global + per-tab)
├── styles.go           # Lipgloss styles
├── theme.go            # Theme definitions (Catppuccin + adaptive)
└── poll.go             # Tick intervals + adaptive polling
```
