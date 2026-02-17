package sdk

import "context"

// PITRService provides read access to PITR status and oplog timelines.
type PITRService interface {
	// Status returns the current PITR status.
	Status(ctx context.Context) (*PITRStatus, error)

	// Timelines returns the available PITR oplog timelines.
	Timelines(ctx context.Context) ([]Timeline, error)
}

// PITRStatus represents the current state of PITR (Point-in-Time Recovery).
type PITRStatus struct {
	Enabled bool
	Running bool
	Nodes   []string // nodes actively performing oplog slicing
	Error   string
}

// Timeline represents a contiguous range of oplog coverage for PITR.
type Timeline struct {
	Start Timestamp
	End   Timestamp
}
