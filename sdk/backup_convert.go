package sdk

import (
	"github.com/percona/percona-backup-mongodb/pbm/backup"
)

// convertBackup converts a PBM BackupMeta to an SDK Backup.
func convertBackup(meta *backup.BackupMeta) Backup {
	return Backup{
		Name:             meta.Name,
		OPID:             meta.OPID,
		Type:             convertBackupType(meta.Type),
		Status:           convertStatus(meta.Status),
		Compression:      convertCompressionType(meta.Compression),
		ConfigName:       convertConfigName(meta.Store.Name),
		StartTS:          convertUnixToTime(meta.StartTS),
		FirstWriteTS:     convertTimestamp(meta.FirstWriteTS),
		LastWriteTS:      convertTimestamp(meta.LastWriteTS),
		LastTransitionTS: convertUnixToTime(meta.LastTransitionTS),
		Size:             meta.Size,
		SizeUncompressed: meta.SizeUncompressed,
		Namespaces:       meta.Namespaces,
		SrcBackup:        meta.SrcBackup,
		MongoVersion:     meta.MongoVersion,
		FCV:              meta.FCV,
		PBMVersion:       meta.PBMVersion,
		Error:            meta.Err,
		Replsets:         convertBackupReplsets(meta.Replsets),
	}
}

// convertBackupReplsets converts a slice of PBM BackupReplset to SDK BackupReplsets.
func convertBackupReplsets(replsets []backup.BackupReplset) []BackupReplset {
	return convertSlice(replsets, func(rs backup.BackupReplset) BackupReplset {
		return convertBackupReplset(&rs)
	})
}

// convertBackupReplset converts a PBM BackupReplset to an SDK BackupReplset.
func convertBackupReplset(rs *backup.BackupReplset) BackupReplset {
	return BackupReplset{
		Name:             rs.Name,
		Status:           convertStatus(rs.Status),
		Node:             rs.Node,
		LastWriteTS:      convertTimestamp(rs.LastWriteTS),
		LastTransitionTS: convertUnixToTime(rs.LastTransitionTS),
		Size:             rs.Size,
		SizeUncompressed: rs.SizeUncompressed,
		IsConfigSvr:      derefBool(rs.IsConfigSvr),
		Error:            rs.Error,
	}
}

// derefBool safely dereferences a *bool, returning false for nil.
func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
