package sdk

import (
	"github.com/percona/percona-backup-mongodb/pbm/defs"
	"github.com/percona/percona-backup-mongodb/pbm/restore"
)

// convertRestore converts a PBM RestoreMeta to an SDK Restore.
func convertRestore(meta *restore.RestoreMeta) Restore {
	r := Restore{
		Name:             meta.Name,
		OPID:             meta.OPID,
		Backup:           meta.Backup,
		BcpChain:         meta.BcpChain,
		Type:             convertBackupType(meta.Type),
		Status:           convertStatus(meta.Status),
		StartTS:          convertUnixToTime(meta.StartTS),
		PITRTarget:       convertPITRTarget(meta.PITR),
		Namespaces:       meta.Namespaces,
		LastTransitionTS: convertUnixToTime(meta.LastTransitionTS),
		Error:            meta.Error,
		Replsets:         convertRestoreReplsets(meta.Replsets),
	}

	// Derive FinishTS from LastTransitionTS on terminal statuses.
	if isTerminalStatus(meta.Status) {
		r.FinishTS = r.LastTransitionTS
	}

	return r
}

// convertPITRTarget converts a PBM PITR unix timestamp to an SDK Timestamp.
// Returns the zero Timestamp if not a PITR restore.
func convertPITRTarget(pitr int64) Timestamp {
	if pitr == 0 {
		return Timestamp{}
	}
	return Timestamp{T: uint32(pitr), I: 0}
}

// isTerminalStatus reports whether a PBM status represents a finished operation.
func isTerminalStatus(s defs.Status) bool {
	switch s {
	case defs.StatusDone, defs.StatusError, defs.StatusCancelled, defs.StatusPartlyDone:
		return true
	default:
		return false
	}
}

// convertRestoreReplsets converts a slice of PBM RestoreReplset to SDK RestoreReplsets.
func convertRestoreReplsets(replsets []restore.RestoreReplset) []RestoreReplset {
	if len(replsets) == 0 {
		return nil
	}

	result := make([]RestoreReplset, len(replsets))
	for i, rs := range replsets {
		result[i] = convertRestoreReplset(&rs)
	}
	return result
}

// convertRestoreReplset converts a PBM RestoreReplset to an SDK RestoreReplset.
func convertRestoreReplset(rs *restore.RestoreReplset) RestoreReplset {
	return RestoreReplset{
		Name:             rs.Name,
		Status:           convertStatus(rs.Status),
		LastTransitionTS: convertUnixToTime(rs.LastTransitionTS),
		Error:            rs.Error,
		Nodes:            convertRestoreNodes(rs.Nodes),
	}
}

// convertRestoreNodes converts a slice of PBM RestoreNode to SDK RestoreNodes.
func convertRestoreNodes(nodes []restore.RestoreNode) []RestoreNode {
	if len(nodes) == 0 {
		return nil
	}

	result := make([]RestoreNode, len(nodes))
	for i, n := range nodes {
		result[i] = RestoreNode{
			Name:             n.Name,
			Status:           convertStatus(n.Status),
			LastTransitionTS: convertUnixToTime(n.LastTransitionTS),
			Error:            n.Error,
		}
	}
	return result
}
