package sdk

import "context"

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
