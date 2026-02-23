package sdk

import (
	"fmt"

	"github.com/percona/percona-backup-mongodb/pbm/compress"
	"github.com/percona/percona-backup-mongodb/pbm/config"
	"github.com/percona/percona-backup-mongodb/pbm/ctrl"
	"github.com/percona/percona-backup-mongodb/pbm/defs"
)

func convertStartLogicalBackupToPBM(cmd StartLogicalBackup) ctrl.Cmd {
	return ctrl.Cmd{
		Cmd: ctrl.CmdBackup,
		Backup: &ctrl.BackupCmd{
			Type:             defs.LogicalBackup,
			Name:             cmd.name,
			Namespaces:       cmd.Namespaces,
			UsersAndRoles:    cmd.UsersAndRoles,
			Compression:      compress.CompressionType(cmd.Compression.String()),
			CompressionLevel: cmd.CompressionLevel,
			NumParallelColls: intToInt32Ptr(cmd.NumParallelColls),
			Profile:          configNameToPBM(cmd.ConfigName),
		},
	}
}

func convertStartIncrementalBackupToPBM(cmd StartIncrementalBackup) ctrl.Cmd {
	return ctrl.Cmd{
		Cmd: ctrl.CmdBackup,
		Backup: &ctrl.BackupCmd{
			Type:             defs.IncrementalBackup,
			IncrBase:         cmd.Base,
			Name:             cmd.name,
			Compression:      compress.CompressionType(cmd.Compression.String()),
			CompressionLevel: cmd.CompressionLevel,
			NumParallelColls: intToInt32Ptr(cmd.NumParallelColls),
			Profile:          configNameToPBM(cmd.ConfigName),
		},
	}
}

func convertStartSnapshotRestoreToPBM(cmd StartSnapshotRestore) ctrl.Cmd {
	return ctrl.Cmd{
		Cmd: ctrl.CmdRestore,
		Restore: &ctrl.RestoreCmd{
			Name:                cmd.name,
			BackupName:          cmd.BackupName,
			Namespaces:          cmd.Namespaces,
			NamespaceFrom:       cmd.NamespaceFrom,
			NamespaceTo:         cmd.NamespaceTo,
			UsersAndRoles:       cmd.UsersAndRoles,
			RSMap:               cmd.RSMap,
			NumParallelColls:    intToInt32Ptr(cmd.NumParallelColls),
			NumInsertionWorkers: intToInt32Ptr(cmd.NumInsertionWorkers),
			AllowPartlyDone:     cmd.AllowPartlyDone,
			Fallback:            cmd.Fallback,
		},
	}
}

func convertStartPITRRestoreToPBM(cmd StartPITRRestore) ctrl.Cmd {
	return ctrl.Cmd{
		Cmd: ctrl.CmdRestore,
		Restore: &ctrl.RestoreCmd{
			Name:                cmd.name,
			BackupName:          cmd.BackupName,
			Namespaces:          cmd.Namespaces,
			NamespaceFrom:       cmd.NamespaceFrom,
			NamespaceTo:         cmd.NamespaceTo,
			UsersAndRoles:       cmd.UsersAndRoles,
			RSMap:               cmd.RSMap,
			OplogTS:             convertTimestampToPBM(cmd.Target),
			NumParallelColls:    intToInt32Ptr(cmd.NumParallelColls),
			NumInsertionWorkers: intToInt32Ptr(cmd.NumInsertionWorkers),
			AllowPartlyDone:     cmd.AllowPartlyDone,
			Fallback:            cmd.Fallback,
		},
	}
}

func convertDeleteByNameToPBM(cmd DeleteBackupByName) ctrl.Cmd {
	return ctrl.Cmd{
		Cmd: ctrl.CmdDeleteBackup,
		Delete: &ctrl.DeleteBackupCmd{
			Backup: cmd.Name,
		},
	}
}

func convertDeleteBackupsBeforeToPBM(cmd DeleteBackupsBefore) ctrl.Cmd {
	return ctrl.Cmd{
		Cmd: ctrl.CmdDeleteBackup,
		Delete: &ctrl.DeleteBackupCmd{
			OlderThan: cmd.OlderThan.Unix(),
			Type:      defs.BackupType(cmd.Type.String()),
			Profile:   configNameToPBM(cmd.ConfigName),
		},
	}
}

func convertAddProfileCommandToPBM(cmd AddProfileCommand) (ctrl.Cmd, error) {
	storage, ok := cmd.storage.(config.StorageConf)
	if !ok {
		return ctrl.Cmd{}, fmt.Errorf("add profile %q: storage config not set", cmd.Name)
	}
	return ctrl.Cmd{
		Cmd: ctrl.CmdAddConfigProfile,
		Profile: &ctrl.ProfileCmd{
			Name:      cmd.Name,
			IsProfile: true,
			Storage:   storage,
		},
	}, nil
}

func convertRemoveProfileCommandToPBM(cmd RemoveProfileCommand) ctrl.Cmd {
	return ctrl.Cmd{
		Cmd: ctrl.CmdRemoveConfigProfile,
		Profile: &ctrl.ProfileCmd{
			Name: cmd.Name,
		},
	}
}

func convertDeletePITRBeforeToPBM(cmd DeletePITRBefore) ctrl.Cmd {
	return ctrl.Cmd{
		Cmd: ctrl.CmdDeletePITR,
		DeletePITR: &ctrl.DeletePITRCmd{
			OlderThan: cmd.OlderThan.Unix(),
		},
	}
}

func convertResyncMainToPBM(cmd ResyncMain) ctrl.Cmd {
	c := ctrl.Cmd{Cmd: ctrl.CmdResync}
	if cmd.IncludeRestores {
		c.Resync = &ctrl.ResyncCmd{IncludeRestores: true}
	}
	return c
}

func convertResyncProfileToPBM(cmd ResyncProfile) ctrl.Cmd {
	return ctrl.Cmd{
		Cmd: ctrl.CmdResync,
		Resync: &ctrl.ResyncCmd{
			Name:  cmd.Name,
			Clear: cmd.Clear,
		},
	}
}

func convertResyncAllProfilesToPBM(cmd ResyncAllProfiles) ctrl.Cmd {
	return ctrl.Cmd{
		Cmd: ctrl.CmdResync,
		Resync: &ctrl.ResyncCmd{
			All:   true,
			Clear: cmd.Clear,
		},
	}
}

// configNameToPBM converts an SDK ConfigName to PBM's profile name string.
// MainConfig and zero value both map to "" (PBM's representation of the main config).
func configNameToPBM(cn ConfigName) string {
	if cn.Equal(MainConfig) || cn.IsZero() {
		return ""
	}
	return cn.String()
}

// intToInt32Ptr converts *int to *int32 for PBM command fields.
// Returns nil for nil input.
func intToInt32Ptr(v *int) *int32 {
	if v == nil {
		return nil
	}
	n := int32(*v)
	return &n
}
