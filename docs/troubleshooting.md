# Troubleshooting

## PBM Version Compatibility

PBMate's SDK is built against a specific PBM version. It works with PBM
clusters running compatible versions.

If your PBM cluster runs a newer version with new enum values or status codes,
the SDK logs warnings (`slog.Warn`) for unknown values rather than crashing.
The TUI will still work — unknown statuses appear as their raw string values.

For the best experience, keep PBMate and your PBM agents on similar versions.

## Connection Issues

### Initial Connection

PBMate connects to the MongoDB cluster used by PBM for coordination. The URI
should point to a mongos (sharded) or replica set member where PBM is
configured.

```bash
# Standard connection
pbmate --uri mongodb://localhost:27017

# With authentication
pbmate --uri "mongodb://user:password@host:27017/?authSource=admin"
```

### Connection Failures and Retries

If the initial connection fails, PBMate retries automatically:

- Backoff: 2s, 4s, 8s, 16s, 30s (capped)
- Each attempt has a 10-second timeout
- The bottom bar shows the retry status and attempt count
- Press `Ctrl+C` at any time to abort

### Mid-Session Disconnects

If the connection drops after PBMate is running, the MongoDB driver handles
reconnection automatically. PBMate does not need to restart. You may see
transient errors in the flash bar during the reconnection window.

### "No URI Available" Error

If PBMate starts with no `--uri` flag, no active context, and no
`current-context` in the config file, it prints a help message. Fix it by
adding a context:

```bash
pbmate context add mycluster --uri mongodb://host:27017
pbmate context use mycluster
```

## Physical Restore Behavior

Physical and incremental restores (including PITR restores based on a
physical/incremental backup) **shut down mongod on every node** in the cluster.
This is fundamental to how PBM's physical restore works — it replaces WiredTiger
data files directly.

What happens:

1. PBMate shows a warning confirmation overlay.
2. On confirm, the restore command is dispatched to PBM.
3. PBMate exits cleanly and prints: `Monitor progress with: pbm status`.
4. PBM agents stop mongod, copy files, and restart mongod on each node.
5. After the restore completes, you can reconnect with `pbmate`.

If the dispatch itself fails (e.g. concurrent operation), PBMate stays open
with the error in the flash bar.

## Concurrent Operation Errors

PBM uses a distributed lock system — only one operation (backup, restore,
resync) can run at a time across the cluster. If you try to start an operation
while another is running, you'll see:

```
start backup: another operation is running: backup 2026-01-15T10:30:00Z
```

Wait for the current operation to finish, or cancel it with `X` (for backups).

## PITR Restore: "No Valid Base Backup"

A PITR restore requires a base backup that meets all of these criteria:

- Status is **done** (completed successfully)
- Last write timestamp is **before** the target restore point
- Not a **selective** backup (no namespace filtering)
- Not from a **named storage profile** (must be from the main config)
- The backup's timeline **covers** the target restore point

If no backup qualifies, the restore form shows "No valid base backup". To fix
this, ensure you have a full (non-selective) backup on the main config that
precedes your target restore time.

## Incremental Backup Chain Errors

Incremental backups form chains: a base backup followed by deltas. Rules:

- You cannot start a non-base incremental backup without an existing chain for
  that storage profile. The TUI detects this and forces a new base automatically.
- Deleting a base backup deletes the entire chain.
- If the TUI reports `ErrNotChainBase`, it means a delete was attempted on a
  chain member that is not the base — this shouldn't happen through the TUI
  (which is chain-aware) but could occur via the SDK.

## Terminal Rendering

PBMate works best in terminals that support:

- **256 colors or true color** — Required for Catppuccin themes. The `default`
  theme uses adaptive colors that work with basic 16-color terminals.
- **Unicode** — Used for status indicators (`●`, `○`), borders, and tree
  connectors.
- **Reasonable size** — The four-panel Overview layout needs at least ~80
  columns and ~24 rows to render without clipping.

If colors look wrong, try the `default` theme which adapts to your terminal's
color scheme, or explicitly set a Catppuccin flavor that matches your terminal
background (e.g. `mocha` for dark, `latte` for light).

## Building from Source

Requires **Go 1.26+** and [Task](https://taskfile.dev/) (task runner).

```bash
task build    # build all modules
task check    # build + vet + lint + test
```

Integration tests use [testcontainers](https://golang.testcontainers.org/) to
launch a MongoDB instance in Docker automatically — no pre-existing cluster
needed. They require a running Docker daemon. Run them with:

```bash
task sdk:integration
```
