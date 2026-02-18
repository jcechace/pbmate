package sdk

import (
	"github.com/percona/percona-backup-mongodb/pbm/oplog"
)

// convertTimeline converts a PBM oplog.Timeline to an SDK Timeline.
// PBM timelines use uint32 unix seconds; SDK uses Timestamp{T, I} where I is 0.
func convertTimeline(tl oplog.Timeline) Timeline {
	return Timeline{
		Start: Timestamp{T: tl.Start},
		End:   Timestamp{T: tl.End},
	}
}

// convertTimelines converts a slice of PBM oplog.Timeline to SDK Timelines.
// A single zero-valued timeline is treated as empty — PBM's internal
// gettimelines() produces this artifact when the oplog chunk list is empty.
func convertTimelines(tlns []oplog.Timeline) []Timeline {
	if len(tlns) == 0 {
		return nil
	}

	// PBM bug workaround: gettimelines() unconditionally appends the
	// accumulator after its loop, producing [{0,0,0}] for empty input.
	if len(tlns) == 1 && tlns[0].Start == 0 && tlns[0].End == 0 {
		return nil
	}

	result := make([]Timeline, len(tlns))
	for i := range tlns {
		result[i] = convertTimeline(tlns[i])
	}
	return result
}

// collectPITRErrors aggregates error messages from PITR replset statuses.
// Returns an empty string if no replsets report errors.
func collectPITRErrors(replsets []oplog.PITRReplset) string {
	var errMsg string
	for _, rs := range replsets {
		if rs.Status == oplog.StatusError && rs.Error != "" {
			if errMsg != "" {
				errMsg += "; "
			}
			errMsg += rs.Name + ": " + rs.Error
		}
	}
	return errMsg
}
