# Usage Guide

A task-oriented guide to PBMate's terminal UI. Press `?` inside the TUI at any
time to see all available keybindings for the current view.

## Connecting

```bash
# Direct connection
pbmate --uri mongodb://localhost:27017

# Save a named context and reuse it
pbmate context add dev --uri mongodb://localhost:27017
pbmate context use dev
pbmate
```

If the connection fails, PBMate retries automatically with exponential backoff
(2s, 4s, 8s, ... up to 30s). Press `Ctrl+C` to abort. See
[Troubleshooting](troubleshooting.md#connection-issues) for details.

## Navigating the Interface

PBMate has three tabs, each with a split-panel layout:

| Key | Tab | Purpose |
|-----|-----|---------|
| `1` | Overview | Cluster health, agent status, PITR status, live logs |
| `2` | Backups | Backup/restore list, start/cancel/delete operations |
| `3` | Config | PBM configuration, storage profiles, YAML viewer |

Navigation basics:

| Key | Action |
|-----|--------|
| `up`/`down` or `k`/`j` | Move selection in the focused panel |
| `]` / `[` | Cycle focus between panels |
| `space` / `enter` | Expand/collapse groups (replica sets, profiles, chains) |
| `tab` | Toggle sub-views (Backups/Restores, Preview/YAML) |
| `?` | Show full help overlay |
| `q` (2x) | Quit (double-press within 2s) |
| `Ctrl+C` | Quit immediately |

## Monitoring

### Check Cluster Health

The **Overview** tab (`1`) shows four panels:

- **Cluster** (top-left) — Agent tree grouped by replica set. Status indicators:
  `●` green = healthy, `●` red = error, `○` dim = stale. Replica set headers
  are collapsible with `space`.
- **Detail** (top-right) — Full info for the selected agent (node, role, version, status).
- **Status** (bottom-left) — PITR enabled/running, active operation, latest backup with age, storage type.
- **Logs** (bottom-right) — Live PBM log viewer, color-coded by severity.

### Stream Logs

| Key | Action |
|-----|--------|
| `f` | Toggle real-time log streaming (follow mode) |
| `w` | Toggle line wrapping in log panel |
| `l` | Open log filter (severity, replica set, event type) |

When a filter is active, the panel title shows the active criteria
(e.g. "Logs (W, rs0, backup)"). Filters persist across refresh cycles.

## Backups

### Start a Backup

| Key | Action |
|-----|--------|
| `s` | Quick backup — logical backup to the main storage, one-key confirm |
| `S` | Custom backup — choose type, profile, compression, namespaces |

The custom backup form (`S`) offers three backup types:

- **Logical** — Mongodump-based. Supports namespace filtering and parallel collections.
- **Physical** — WiredTiger file copy. Faster for large datasets, but restores shut down mongod.
- **Incremental** — Physical delta since last backup. When no chain exists for the
  selected profile, the form forces a new base backup automatically.

### Cancel a Running Backup

Press `X` on the Backups tab while a backup is in progress.

### Restore from a Snapshot

| Key | Action |
|-----|--------|
| `r` | Restore the selected backup (options form with scope, tuning) |
| `R` | Restore wizard — choose snapshot or PITR, pick a profile, select a backup |

The `R` wizard is a two-step flow: first select a target (type, backup), then
configure options (namespaces, parallel collections, workers).

### Point-in-Time Restore

Select a PITR timeline in the backup list and press `r`, or use the `R` wizard
and choose PITR mode. The form offers time presets:

- Latest, -5 min, -15 min, -30 min, -1 hour, -6 hour, Custom

You'll also select a base backup from the list of valid bases for that timeline.

### Physical/Incremental Restore Warning

Restoring from a physical or incremental backup (including PITR restores with a
physical base) **shuts down mongod on every node** in the cluster. PBMate shows
a final warning confirmation before dispatch. After confirming, the TUI exits
and prints instructions for monitoring progress with `pbm status`.

### Delete Backups

| Key | Action |
|-----|--------|
| `d` | Delete the selected backup (with confirmation) |
| `D` | Bulk delete — backups or PITR chunks older than a threshold |

The bulk delete form (`D`) supports:
- Target: Backups or PITR chunks
- Time threshold: presets (Now, 1d, 3d, 1w, 2w, 1m) or a custom date
- Filters: backup type (logical/physical/incremental), storage profile

Pressing `d` on a PITR timeline opens the bulk delete form with PITR preselected.

For incremental backups, delete is chain-aware: deleting a base backup deletes
the entire chain.

## PITR

### Toggle PITR

Press `p` from any tab to enable or disable point-in-time recovery. A
confirmation overlay appears before the change is applied.

## Configuration

### View PBM Configuration

The **Config** tab (`3`) shows the main PBM configuration and storage profiles
in the left panel. The right panel shows details and syntax-highlighted YAML.

Press `tab` to toggle between the detail preview and raw YAML view.

### Upload a New Config

| Key | Action |
|-----|--------|
| `C` | Set config wizard — choose target (main or profile), pick a YAML file |
| `c` | Set config for the selected item (pre-filled from cursor) |

The `C` wizard is a three-step flow: target form, file picker, and an optional
override confirmation if the target already has a config.

### Edit in External Editor

Press `e` on the Config tab to open the selected configuration in your external
editor. This works like `kubectl edit`:

1. PBMate writes the current YAML to a temp file.
2. Your editor opens the file.
3. On save and exit, PBMate validates and applies the changes.

If the YAML is invalid, the temp file is preserved and its path is shown for
recovery. See [Configuration — Editor](configuration.md#editor) for editor
setup.

### Manage Storage Profiles

- Create a profile via `C` (set config wizard, choose a profile name target).
- Delete a profile with `d` on the Config tab.
- Each profile appears under the "Profiles" section in the config list.

### Resync Agents

| Key | Action |
|-----|--------|
| `R` | Resync — choose scope (Main, Profile, or All Profiles) |
| `r` | Resync the selected config/profile |

Resync tells PBM agents to re-read their configuration from the database.

## Readonly Mode

Launch PBMate in readonly mode for safe production monitoring:

```bash
pbmate --readonly
```

All mutation keys are silently disabled. A yellow `READONLY` badge appears in
the status bar, and the help overlay only shows navigation bindings.

See [Configuration — Readonly Mode](configuration.md#readonly-mode) for
persistent setup.

## Keybinding Reference

### Global

| Key | Action |
|-----|--------|
| `q` (2x) | Quit (double-press within 2 seconds) |
| `Ctrl+C` | Quit immediately |
| `1` / `2` / `3` | Switch to Overview / Backups / Config tab |
| `?` | Toggle help overlay |
| `Esc` | Close overlay / go back |
| `up`/`down` or `k`/`j` | Navigate within panels |
| `]` / `[` | Focus next / previous panel |
| `s` / `S` | Start backup (quick / custom) |
| `X` | Cancel running backup |
| `d` | Delete selected item |
| `p` | Toggle PITR enable/disable |

### Overview Tab

| Key | Action |
|-----|--------|
| `space` / `enter` | Expand/collapse replica set group |
| `f` | Toggle log follow (real-time streaming) |
| `w` | Toggle log word wrap |
| `l` | Open log filter form |

### Backups Tab

| Key | Action |
|-----|--------|
| `tab` | Toggle between Backups and Restores list |
| `space` / `enter` | Expand/collapse profile group |
| `r` | Restore selected backup/timeline |
| `R` | Open restore wizard |
| `D` | Bulk delete (backups or PITR) |

### Config Tab

| Key | Action |
|-----|--------|
| `tab` | Toggle between detail preview and YAML |
| `C` / `c` | Set config (wizard / for selected item) |
| `R` / `r` | Resync (wizard / for selected item) |
| `e` | Edit in external editor |
