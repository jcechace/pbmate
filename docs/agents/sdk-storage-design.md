# SDK Storage Design — Exploratory

> This document captures design decisions and future directions for storage-aware
> SDK operations. It records the rationale from design discussions so future
> sessions can pick up context without re-deriving everything.

## Context

PBM uses backup storage (S3, GCS, Azure, filesystem) for backup data and for
coordination metadata during physical restores. The SDK currently accesses
only MongoDB for all operations. This limits the SDK in two areas:

1. **Physical restore monitoring** — mongod shuts down during physical restores,
   making MongoDB-based polling impossible.
2. **Backup inspection** — listing backup contents and verifying integrity
   requires reading from storage.

All the heavy lifting is in reusable PBM library packages (`pbm/restore`,
`pbm/storage`, `pbm/util`, `pbm/backup`), not in the CLI. The SDK can call
these functions directly.

## Decisions Made

### Result Type Redesign (Immediate — implement now)

**`BackupResult`** stays a concrete struct. Gets a `Wait()` method. Exported
fields preserved (`Name`, embedded `CommandResult`). Unexported `svc` field
added for `Wait()` delegation.

**`RestoreResult`** becomes an interface:

```go
type RestoreResult interface {
    Name() string
    OPID() string
    Waitable() bool
    Wait(ctx context.Context, opts RestoreWaitOptions) (*Restore, error)
}
```

Two unexported implementations:
- `waitableRestoreResult` — polls MongoDB. Used for logical restores.
- `unwaitableRestoreResult` — returns `ErrRestoreUnwaitable`. Used for any
  restore based on a physical or incremental backup.

`Wait()` removed from both `BackupService` and `RestoreService` interfaces.
The only way to wait is via the result from `Start()`. Consumers who need to
monitor by name use `Get()` in their own polling loop.

**`Start()` dispatch logic:**

| Command | Base Backup Type | Result |
|---------|-----------------|--------|
| `StartSnapshotRestore` | Logical | `waitableRestoreResult` |
| `StartSnapshotRestore` | Physical/Incremental | `unwaitableRestoreResult` |
| `StartPITRRestore` | Logical | `waitableRestoreResult` |
| `StartPITRRestore` | Physical/Incremental | `unwaitableRestoreResult` |

`Start()` looks up the backup metadata to determine the base backup type.
This is done while MongoDB is still up (before the restore command shuts
anything down).

**Why BackupResult is not an interface:** All backup types keep mongod running.
There is no polymorphism need. External backups (future) also keep mongod
running — the difference is when to stop waiting (CopyReady vs Done), which
is an option, not a different implementation.

**Why not "logical" / "physical" naming:** The `waitableRestoreResult` name
describes behavior, not mechanism. If PBM separates the metadata DB from the
managed DB in the future, the "waitable" implementation could handle physical
restores too (MongoDB stays up). Naming by behavior avoids a rename.

### Physical Restore Lifecycle

Physical restores (snapshot restore of physical/incremental backup) have this
lifecycle:

1. Command dispatch (MongoDB up)
2. Agents coordinate via storage sync files (MongoDB up briefly)
3. **Agents shut down mongod on every node**
4. Wipe data directory, copy files from storage
5. Start temporary standalone mongod, clear replication metadata
6. Restart with oplog recovery
7. Agent cleanup, write final metadata to storage
8. **Operator must manually restart mongod and agents**

PITR with physical base adds more steps between 6 and 8:
- `restore-finish` must be called
- Agents restart
- Oplog replay from storage
- Human-in-the-loop gap makes this fundamentally unwaitable

During phases 3-8, all progress metadata lives on backup storage at
`.pbm.restore/<name>/` (sync files, heartbeats, status per node/RS/cluster).

### PBM Library Functions Available

All exported and in reusable packages (not CLI):

| Function | Package | Purpose |
|----------|---------|---------|
| `restore.GetPhysRestoreMeta(name, stg, log)` | `pbm/restore` | Read physical restore metadata from storage |
| `restore.ParsePhysRestoreStatus(name, stg, log)` | `pbm/restore` | Parse per-node sync file status |
| `restore.IsCleanupHbAlive(name, stg, tskew)` | `pbm/restore` | Check if post-restore cleanup is running |
| `util.GetStorage(ctx, conn, node, log)` | `pbm/util` | Create storage client from live MongoDB config |
| `util.GetProfiledStorage(ctx, conn, profile, node, log)` | `pbm/util` | Same but for a named storage profile |
| `backup.ReadArchiveNamespaces(stg, metafile)` | `pbm/backup` | List namespaces in a logical backup archive |
| `backup.ReadFilelistForReplset(stg, name, rs)` | `pbm/backup` | List files in a physical/incremental backup |
| `backup.CheckBackupFiles(ctx, stg, name)` | `pbm/backup` | Verify all backup files exist on storage |
| `backup.CheckBackupDataFiles(ctx, stg, meta)` | `pbm/backup` | Verify backup integrity (type-aware) |
| `storage.HasReadAccess(ctx, stg)` | `pbm/storage` | Check storage read access |

The SDK already imports `pbm/restore`, `pbm/backup`, `pbm/storage`,
`pbm/config`. Only `pbm/util` would be new.

## Exploratory — Not Yet Implemented

### Physical Restore Storage-Based Wait

Replace `unwaitableRestoreResult` with a new `physicalRestoreResult` for
snapshot restores of physical/incremental backups. PITR with physical base
remains unwaitable.

**Design:** `Start()` eagerly creates a storage handle and records time skew
(`wallTime - clusterTime`) while MongoDB is still up. The `physicalRestoreResult`
carries these internally and uses them in `Wait()`:

- Poll via `restore.GetPhysRestoreMeta(name, stg, log)`
- Extended stale frame (180s vs 30s)
- Wall clock + time skew for heartbeat freshness
- `restore.IsCleanupHbAlive()` check before declaring done
- Convert `*restore.RestoreMeta` to `sdk.Restore` via existing `convertRestore`

**Open question:** PBM has active work to separate the metadata DB from the
managed DB. If completed, physical restores would no longer shut down the
metadata connection, and `waitableRestoreResult` (MongoDB polling) would work
for physical restores too. The interface-based design handles both futures —
the implementation behind `RestoreResult` can change without API impact.

### Backup File Inspection

```go
// On BackupService:
ListFiles(ctx context.Context, name string) ([]BackupFile, error)
Verify(ctx context.Context, name string) error
```

`ListFiles` returns backup contents from storage:
- Logical: namespaces (db.collection) via `ReadArchiveNamespaces`
- Physical/incremental: WiredTiger files via `ReadFilelistForReplset`

`Verify` checks file existence and non-emptiness via `CheckBackupFiles`.

Both create storage clients internally via `util.GetProfiledStorage`.
Profile name comes from backup metadata (`Store.Name`).

### Storage Health

```go
// On ConfigService or a new StorageService:
CheckStorage(ctx context.Context, name ConfigName) error
```

Verify storage accessibility and initialization for main or profile storage.

### Storage-Only Client Construction

Currently the SDK always requires a MongoDB connection. Future work may add:

```go
sdk.NewClient(ctx, sdk.WithConfigFile("/path/to/pbm-config.yaml"))
```

This would enable:
- `Verify` / `ListFiles` without MongoDB
- Describe-restore when cluster is down
- Cross-cluster backup inspection

This is a significant design change to the Client constructor and is deferred.

### Per-Key Config Changes

```go
// On ConfigService:
SetVar(ctx context.Context, key, value string) error
GetVar(ctx context.Context, key string) (string, error)
```

Wraps PBM's exported `config.SetConfigVar` / `config.GetConfigVar`.
Deferred — file-based workflow is sufficient for now.

## Known Limitations

1. **`ListFiles`/`Verify` require a live MongoDB connection** — storage clients
   are constructed from PBM config stored in MongoDB.
2. **Result types are not serializable** — they carry unexported service
   references. Cannot be persisted across process restarts.
3. **`unwaitableRestoreResult` for all physical-based restores** — until
   storage-based polling is implemented, physical snapshot restores are also
   unwaitable even though they theoretically could be.
