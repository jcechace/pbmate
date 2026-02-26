# PBMate SDK

A Go client library for [Percona Backup for MongoDB (PBM)](https://github.com/percona/percona-backup-mongodb).

The SDK wraps PBM's internal packages behind stable, domain-typed interfaces. PBM internals can change freely — the SDK's conversion layer absorbs the changes, and consumer code doesn't break.

> **PBM version:** The SDK is built against PBM v2.9.x internals. Check `sdk/go.mod` for the exact pinned version. Running against a different PBM version may produce `slog.Warn` messages for unknown enum values but will not crash.

```go
import sdk "github.com/jcechace/pbmate/sdk/v2"
```

## Quick Start

```go
// Connect to a PBM-configured MongoDB cluster.
client, err := sdk.NewClient(ctx, sdk.WithMongoURI("mongodb://localhost:27017"))
if err != nil {
    log.Fatal(err)
}
defer client.Close(ctx)

// List the 5 most recent backups.
backups, err := client.Backups.List(ctx, sdk.ListBackupsOptions{Limit: 5})
for _, bk := range backups {
    fmt.Printf("%-30s  %-12s  %s\n", bk.Name, bk.Type, bk.Status)
}

// Start a logical backup and wait for it.
result, err := client.Backups.Start(ctx, sdk.StartLogicalBackup{})
if err != nil {
    log.Fatal(err)
}
bk, err := client.Backups.Wait(ctx, result.Name, sdk.BackupWaitOptions{
    OnProgress: func(b *sdk.Backup) {
        fmt.Printf("  %s elapsed...\n", b.Elapsed().Truncate(time.Second))
    },
})
fmt.Printf("backup %s finished: %s (%s)\n", bk.Name, bk.Status, bk.Duration())
```

> `Elapsed()` returns live elapsed time for in-progress operations and the
> final duration for completed ones. `Duration()` returns zero until the
> operation reaches a terminal status.

## Services

The client exposes domain-specific services as interface-typed fields:

| Service | Field | Purpose |
|---|---|---|
| [BackupService](backup.go) | `client.Backups` | List, get, start, wait, delete, cancel backups; pre-delete safety check |
| [RestoreService](restore.go) | `client.Restores` | List, get, start, wait for restores |
| [ConfigService](config.go) | `client.Config` | Read/write configuration and storage profiles |
| [ClusterService](cluster.go) | `client.Cluster` | Cluster topology, agent status, running operations, lock checks |
| [PITRService](pitr.go) | `client.PITR` | PITR status, oplog timelines, chunk deletion |
| [LogService](log.go) | `client.Logs` | Query and stream PBM logs |

Every service is an interface — mock any of them in your tests.

## Examples

### Start an Incremental Backup

```go
// Base backup (starts a new chain).
result, err := client.Backups.Start(ctx, sdk.StartIncrementalBackup{
    Base: true,
})

// Subsequent increment (extends the chain).
result, err := client.Backups.Start(ctx, sdk.StartIncrementalBackup{})
```

### Start a Selective Backup to a Named Profile

```go
profile, _ := sdk.NewConfigName("archive")
result, err := client.Backups.Start(ctx, sdk.StartLogicalBackup{
    ConfigName:    profile,
    Namespaces:    []string{"mydb.*", "analytics.*"},
    UsersAndRoles: true, // include users/roles (requires whole-database namespaces)
    Compression:   sdk.CompressionTypeZSTD,
})
```

### Restore from a Backup

```go
// Snapshot restore.
result, err := client.Restores.Start(ctx, sdk.StartSnapshotRestore{
    BackupName: "2026-02-19T20:28:16Z",
})

// Point-in-time restore.
result, err := client.Restores.Start(ctx, sdk.StartPITRRestore{
    BackupName: "2026-02-19T20:28:16Z",
    Target:     sdk.Timestamp{T: 1740000000},
})

// Wait for completion.
restore, err := client.Restores.Wait(ctx, result.Name, sdk.RestoreWaitOptions{})
```

### Override Performance Tuning

```go
// Backup with custom parallelism.
numColls := 8
result, err := client.Backups.Start(ctx, sdk.StartLogicalBackup{
    NumParallelColls: &numColls,
})

// Restore with custom parallelism and insertion workers.
colls, workers := 4, 2
result, err := client.Restores.Start(ctx, sdk.StartSnapshotRestore{
    BackupName:          "2026-02-19T20:28:16Z",
    NumParallelColls:    &colls,
    NumInsertionWorkers: &workers,
})
```

> All performance fields are `*int` or `*bool` — nil means "use the
> server-configured default". Set them only when you need to override.

### Check Cluster Health

```go
agents, err := client.Cluster.Agents(ctx)
for _, a := range agents {
    status := "ok"
    if a.Stale {
        status = "STALE"
    } else if !a.OK {
        status = fmt.Sprintf("ERROR: %v", a.Errors)
    }
    fmt.Printf("%-20s  %-10s  %s  %s\n", a.Node, a.ReplicaSet, a.Role, status)
}
```

### Monitor PITR Status

```go
status, _ := client.PITR.Status(ctx)
if status.Enabled && status.Running {
    fmt.Println("PITR is actively slicing oplog")
}

timelines, _ := client.PITR.Timelines(ctx)
for _, tl := range timelines {
    fmt.Printf("restore window: %s - %s\n",
        tl.Start.Time().UTC().Format(time.RFC3339),
        tl.End.Time().UTC().Format(time.RFC3339))
}
```

### Stream Logs

```go
ctx, cancel := context.WithCancel(ctx)
defer cancel()

entries, errs := client.Logs.Follow(ctx, sdk.FollowOptions{
    LogFilter: sdk.LogFilter{Severity: sdk.LogSeverityWarning},
})
for entry := range entries {
    fmt.Printf("[%s] %s: %s\n", entry.Severity, entry.Timestamp.UTC().Format(time.RFC3339), entry.Message)
}
if err := <-errs; err != nil {
    log.Printf("follow ended: %v", err)
}
```

### Handle Concurrent Operations

```go
result, err := client.Backups.Start(ctx, sdk.StartLogicalBackup{})
if err != nil {
    var concurrent *sdk.ConcurrentOperationError
    if errors.As(err, &concurrent) {
        fmt.Printf("blocked by %s (opid: %s)\n", concurrent.Type, concurrent.OPID)
        return
    }
    log.Fatal(err)
}
```

### Delete Backups

```go
// Delete a single backup by name.
_, err := client.Backups.Delete(ctx, sdk.DeleteBackupByName{
    Name: "2026-02-19T20:28:16Z",
})

// Bulk delete backups older than 30 days.
cutoff := time.Now().Add(-30 * 24 * time.Hour)
_, err := client.Backups.Delete(ctx, sdk.DeleteBackupsBefore{
    OlderThan: cutoff,
    Type:      sdk.BackupTypeLogical,
})
```

### Pre-Delete Safety Check

```go
// Check whether a backup can be safely deleted before dispatching.
if err := client.Backups.CanDelete(ctx, bk.Name); err != nil {
    switch {
    case errors.Is(err, sdk.ErrDeleteProtectedByPITR):
        fmt.Println("backup is the last PITR base snapshot, cannot delete")
    case errors.Is(err, sdk.ErrNotChainBase):
        fmt.Println("incremental backup must be deleted from its chain base")
    case errors.Is(err, sdk.ErrBackupInProgress):
        fmt.Println("backup is still running, wait for completion")
    }
    return
}
_, err := client.Backups.Delete(ctx, sdk.DeleteBackupByName{Name: bk.Name})
```

### Manage Storage Profiles

```go
// List profiles.
profiles, _ := client.Config.ListProfiles(ctx)
for _, p := range profiles {
    fmt.Printf("%s: %s %s\n", p.Name, p.Storage.Type, p.Storage.Path)
}

// Apply a profile from YAML.
f, _ := os.Open("archive-profile.yaml")
defer f.Close()
_, err := client.Config.SetProfile(ctx, "archive", f)
```

## Design

### Conversion Boundary

The SDK owns all public types. PBM internal types (`backup.BackupMeta`, `ctrl.Cmd`, etc.) are converted to SDK types in `*_convert.go` files before reaching the public API. When PBM internals change, the conversion layer is updated — consumer code stays stable.

```
Consumer  <-->  SDK types  <-->  *_convert.go  <-->  PBM internals
                (stable)         (absorbs changes)    (can change freely)
```

### Sealed Command Interfaces

Operations that have distinct variants use sealed interfaces with unexported marker methods. This prevents invalid command construction at compile time:

```go
// StartBackupCommand is sealed — only these two types implement it:
//   - StartLogicalBackup    (has Namespaces field)
//   - StartIncrementalBackup (has Base field)
//
// You can't mix fields from different strategies or pass an arbitrary struct.
result, err := client.Backups.Start(ctx, sdk.StartLogicalBackup{
    Namespaces: []string{"mydb.mycol"},
})
```

Other sealed hierarchies: `StartRestoreCommand` (snapshot vs PITR), `DeleteBackupCommand` (by name vs before timestamp), `DeletePITRCommand`, `ResyncCommand`.

### Command Validation

Every command type implements `Validate() error`. Service methods call `Validate()` before checking locks or dispatching, so invalid commands fail fast with a clear error — no round-trip to MongoDB needed. Commands with no constraints return `nil`.

```go
cmd := sdk.StartLogicalBackup{
    UsersAndRoles: true,
    // Namespaces is empty — UsersAndRoles requires a selective operation.
}
if err := cmd.Validate(); err != nil {
    fmt.Println(err) // "start backup: users-and-roles is only valid for selective operations (namespaces must be set)"
}
```

### Value Objects

Enum-like types (`Status`, `BackupType`, `CompressionType`, etc.) use unexported value fields with constructor functions. Invalid values can't be created by external code:

```go
// These work:
bt := sdk.BackupTypeLogical                           // predefined constant
bt, err := sdk.ParseBackupType("logical")             // parse from string
bt, err := sdk.ParseBackupType("garbage")             // returns error

// This doesn't compile — value field is unexported:
// bt := sdk.BackupType{value: "garbage"}
```

### Interface-Based Services

Every service is an interface. This enables mocking in consumer tests without wrapping or code generation:

```go
type myApp struct {
    backups sdk.BackupService  // inject real or mock
}
```

## Requirements

- Go 1.26+
- A running MongoDB cluster with PBM agents configured
- Network access to the MongoDB cluster
