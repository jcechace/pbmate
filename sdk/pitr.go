package sdk

import (
	"context"
	"sort"
)

// PITRService provides read access to PITR status and oplog timelines.
//
// Example — check PITR status and available restore windows:
//
//	status, err := client.PITR.Status(ctx)
//	if status.Enabled && status.Running {
//	    fmt.Println("PITR is actively slicing oplog")
//	}
//
//	timelines, _ := client.PITR.Timelines(ctx)
//	for _, tl := range timelines {
//	    fmt.Printf("PITR window: %s → %s\n",
//	        tl.Start.Time().UTC(), tl.End.Time().UTC())
//	}
type PITRService interface {
	// Status returns the current PITR status, including whether PITR is
	// enabled in the configuration and whether oplog slicing is actively
	// running.
	Status(ctx context.Context) (*PITRStatus, error)

	// Timelines returns the available PITR oplog timelines. Each timeline
	// represents a contiguous range of oplog coverage across all replica
	// sets. Gaps in coverage (e.g., from a stopped agent) result in
	// separate timeline entries. Returns an empty slice if no oplog data
	// is available.
	//
	// Note: timelines represent raw oplog coverage. A PITR restore also
	// requires a base backup snapshot within the timeline's range.
	Timelines(ctx context.Context) ([]Timeline, error)

	// Bases returns backups that are valid base snapshots for a PITR restore
	// to the given target timestamp. It fetches all backups and timelines,
	// then filters using [FilterPITRBases]. Returns an empty slice if no
	// eligible base backup exists.
	//
	// Example:
	//
	//	target := sdk.Timestamp{T: uint32(time.Now().Add(-time.Hour).Unix())}
	//	bases, err := client.PITR.Bases(ctx, target)
	//	if len(bases) == 0 {
	//	    fmt.Println("no valid base backup for PITR")
	//	}
	Bases(ctx context.Context, target Timestamp) ([]Backup, error)

	// Delete requests deletion of PITR oplog chunks. The deletion is
	// processed asynchronously by PBM agents — the command returns
	// immediately. Returns a [*ConcurrentOperationError] if another
	// operation is running.
	//
	// The cmd parameter is a sealed [DeletePITRCommand] with two variants:
	//   - [DeletePITRBefore] deletes chunks older than a cutoff time.
	//   - [DeletePITRAll] deletes all chunks (equivalent to "older than now").
	//
	// Example — delete chunks older than 7 days:
	//
	//	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	//	_, err := client.PITR.Delete(ctx, sdk.DeletePITRBefore{
	//	    OlderThan: cutoff,
	//	})
	//
	// Example — delete all chunks:
	//
	//	_, err := client.PITR.Delete(ctx, sdk.DeletePITRAll{})
	Delete(ctx context.Context, cmd DeletePITRCommand) (CommandResult, error)
}

// PITRStatus represents the current state of PITR (Point-in-Time Recovery).
type PITRStatus struct {
	Enabled bool     // true if PITR is enabled in the PBM configuration
	Running bool     // true if oplog slicing is actively running (agents are capturing oplog)
	Nodes   []string // nodes actively performing oplog slicing; empty when not running
	Error   string   // aggregated per-replset errors from PITR meta, if any
}

// Timeline represents a contiguous range of oplog coverage for PITR.
// A PITR restore can target any point within [Start, End]. Multiple
// timelines indicate gaps in oplog coverage.
type Timeline struct {
	Start Timestamp // inclusive start of oplog coverage
	End   Timestamp // inclusive end of oplog coverage
}

// FilterPITRBases returns the subset of backups that are valid base snapshots
// for a PITR restore to the given target timestamp. A backup qualifies when
// all of the following hold:
//
//   - Status is [StatusDone]
//   - LastWriteTS is non-zero and strictly before the target
//   - Not selective (no namespace filter)
//   - Not external
//   - On the main configuration (PITR chunks are stored on main only)
//   - LastWriteTS falls within a timeline whose coverage reaches the target
//
// Results are sorted by LastWriteTS descending (most recent first).
//
// This is a pure function exported for convenience — the signature may evolve
// in future SDK versions.
func FilterPITRBases(target Timestamp, backups []Backup, timelines []Timeline) []Backup {
	var result []Backup
	for i := range backups {
		bk := &backups[i]
		if !isPITRBaseCandidate(bk, target, timelines) {
			continue
		}
		result = append(result, *bk)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].LastWriteTS.T > result[j].LastWriteTS.T
	})

	return result
}

// isPITRBaseCandidate checks whether a single backup qualifies as a PITR
// base for the given target.
func isPITRBaseCandidate(bk *Backup, target Timestamp, timelines []Timeline) bool {
	if !bk.Status.Equal(StatusDone) {
		return false
	}
	if bk.LastWriteTS.IsZero() {
		return false
	}
	if bk.LastWriteTS.T >= target.T {
		return false
	}
	if bk.IsSelective() {
		return false
	}
	if bk.Type.Equal(BackupTypeExternal) {
		return false
	}
	if !bk.ConfigName.Equal(MainConfig) {
		return false
	}
	// The backup's LastWriteTS must fall within a timeline that also
	// covers the target — otherwise there's a gap in oplog coverage
	// between the backup and the restore point.
	return timelineCovers(bk.LastWriteTS, target, timelines)
}

// timelineCovers reports whether any timeline contains both the backup
// point and the target point (i.e., Start <= backupTS and target <= End).
func timelineCovers(backupTS, target Timestamp, timelines []Timeline) bool {
	for i := range timelines {
		tl := &timelines[i]
		if tl.Start.T <= backupTS.T && target.T <= tl.End.T {
			return true
		}
	}
	return false
}
