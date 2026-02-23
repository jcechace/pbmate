// Package sdk provides a Go client for Percona Backup for MongoDB (PBM).
//
// The SDK wraps PBM's internal packages behind clean domain types and
// service interfaces, isolating consumers from PBM internals. All MongoDB
// driver and BSON types are converted to SDK-owned equivalents before
// reaching the public API.
//
// # Client
//
// Create a client with functional options:
//
//	client, err := sdk.NewClient(ctx, sdk.WithMongoURI("mongodb://user:pass@host:27017"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close(ctx)
//
// # Services
//
// The client exposes domain services as interface-typed fields:
//
//   - [BackupService]  — list, get, start, wait, delete, and cancel backups
//   - [RestoreService] — list, get, start, and wait for restores
//   - [ConfigService]  — read/write configuration, manage storage profiles, resync
//   - [ClusterService] — cluster topology, agents, running operations, and lock checks
//   - [PITRService]    — PITR status, oplog timelines, and chunk deletion
//   - [LogService]     — query and follow PBM logs
//
// # Sealed Commands
//
// Operations with distinct variants use sealed interfaces that prevent
// invalid command construction at compile time. For example,
// [BackupService.Start] accepts a [StartBackupCommand] which is either a
// [StartLogicalBackup] or [StartIncrementalBackup] — each variant exposes
// only the fields valid for that strategy.
//
// # Value Objects
//
// Enum-like types ([Status], [BackupType], [CompressionType], [LogSeverity],
// etc.) use a DDD-style value object pattern with an unexported value field.
// Compare with Equal, parse from strings with the corresponding Parse*
// function, and check for zero with IsZero.
package sdk
