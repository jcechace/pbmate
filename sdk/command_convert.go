package sdk

import (
	"fmt"

	"github.com/percona/percona-backup-mongodb/pbm/compress"
	"github.com/percona/percona-backup-mongodb/pbm/config"
	"github.com/percona/percona-backup-mongodb/pbm/ctrl"
	"github.com/percona/percona-backup-mongodb/pbm/defs"
)

// convertCommandToPBM converts an SDK Command to PBM's ctrl.Cmd.
func convertCommandToPBM(cmd Command) (ctrl.Cmd, error) {
	switch c := cmd.(type) {
	case BackupCommand:
		return convertBackupCommandToPBM(c), nil
	case RestoreCommand:
		return convertRestoreCommandToPBM(c), nil
	case DeleteBackupByName:
		return convertDeleteByNameToPBM(c), nil
	case DeleteBackupsBefore:
		return convertDeleteBackupsBeforeToPBM(c), nil
	case AddProfileCommand:
		return convertAddProfileCommandToPBM(c)
	case RemoveProfileCommand:
		return convertRemoveProfileCommandToPBM(c), nil
	case CancelBackupCommand:
		return ctrl.Cmd{Cmd: ctrl.CmdCancelBackup}, nil
	default:
		return ctrl.Cmd{}, fmt.Errorf("unsupported command type: %T", cmd)
	}
}

func convertBackupCommandToPBM(cmd BackupCommand) ctrl.Cmd {
	return ctrl.Cmd{
		Cmd: ctrl.CmdBackup,
		Backup: &ctrl.BackupCmd{
			Type:        defs.BackupType(cmd.Type.String()),
			IncrBase:    cmd.IncrBase,
			Name:        cmd.Name,
			Namespaces:  cmd.Namespaces,
			Compression: compress.CompressionType(cmd.Compression.String()),
			Profile:     configNameToPBM(cmd.ConfigName),
		},
	}
}

func convertRestoreCommandToPBM(cmd RestoreCommand) ctrl.Cmd {
	return ctrl.Cmd{
		Cmd: ctrl.CmdRestore,
		Restore: &ctrl.RestoreCmd{
			Name:       cmd.Name,
			BackupName: cmd.BackupName,
			Namespaces: cmd.Namespaces,
			OplogTS:    convertTimestampToPBM(cmd.PITRTarget),
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

// configNameToPBM converts an SDK ConfigName to PBM's profile name string.
// MainConfig and zero value both map to "" (PBM's representation of the main config).
func configNameToPBM(cn ConfigName) string {
	if cn.Equal(MainConfig) || cn.IsZero() {
		return ""
	}
	return cn.String()
}
