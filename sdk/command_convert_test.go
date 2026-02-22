package sdk

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/percona-backup-mongodb/pbm/compress"
	"github.com/percona/percona-backup-mongodb/pbm/config"
	"github.com/percona/percona-backup-mongodb/pbm/ctrl"
	"github.com/percona/percona-backup-mongodb/pbm/defs"
	"github.com/percona/percona-backup-mongodb/pbm/storage"
	"github.com/percona/percona-backup-mongodb/pbm/storage/fs"
)

func TestConvertBackupCommandToPBM(t *testing.T) {
	cmd := BackupCommand{
		Name:        "2024-01-15T10:30:00Z",
		Type:        BackupTypeLogical,
		ConfigName:  MainConfig,
		Compression: CompressionTypeZSTD,
		Namespaces:  []string{"db1.coll1"},
		IncrBase:    true,
	}

	result, err := convertCommandToPBM(cmd)
	require.NoError(t, err)

	assert.Equal(t, ctrl.CmdBackup, result.Cmd)
	require.NotNil(t, result.Backup)
	assert.Equal(t, defs.LogicalBackup, result.Backup.Type)
	assert.Equal(t, "2024-01-15T10:30:00Z", result.Backup.Name)
	assert.Equal(t, compress.CompressionTypeZstandard, result.Backup.Compression)
	assert.Equal(t, []string{"db1.coll1"}, result.Backup.Namespaces)
	assert.True(t, result.Backup.IncrBase)
	assert.Equal(t, "", result.Backup.Profile, "MainConfig should map to empty string")
}

func TestConvertBackupCommandWithProfile(t *testing.T) {
	cn, err := NewConfigName("my-s3")
	require.NoError(t, err)

	cmd := BackupCommand{
		Name:       "2024-01-15T10:30:00Z",
		Type:       BackupTypePhysical,
		ConfigName: cn,
	}

	result, err := convertCommandToPBM(cmd)
	require.NoError(t, err)

	assert.Equal(t, "my-s3", result.Backup.Profile)
}

func TestConvertRestoreCommandToPBM(t *testing.T) {
	cmd := RestoreCommand{
		Name:       "restore-2024",
		BackupName: "2024-01-15T10:30:00Z",
		PITRTarget: Timestamp{T: 1700000000, I: 1},
		Namespaces: []string{"db1.coll1"},
	}

	result, err := convertCommandToPBM(cmd)
	require.NoError(t, err)

	assert.Equal(t, ctrl.CmdRestore, result.Cmd)
	require.NotNil(t, result.Restore)
	assert.Equal(t, "restore-2024", result.Restore.Name)
	assert.Equal(t, "2024-01-15T10:30:00Z", result.Restore.BackupName)
	assert.Equal(t, uint32(1700000000), result.Restore.OplogTS.T)
	assert.Equal(t, uint32(1), result.Restore.OplogTS.I)
	assert.Equal(t, []string{"db1.coll1"}, result.Restore.Namespaces)
}

func TestConvertDeleteBackupByNameToPBM(t *testing.T) {
	cmd := DeleteBackupByName{
		Name: "2024-01-15T10:30:00Z",
	}

	result, err := convertCommandToPBM(cmd)
	require.NoError(t, err)

	assert.Equal(t, ctrl.CmdDeleteBackup, result.Cmd)
	require.NotNil(t, result.Delete)
	assert.Equal(t, "2024-01-15T10:30:00Z", result.Delete.Backup)
	assert.Zero(t, result.Delete.OlderThan)
	assert.Empty(t, result.Delete.Type)
	assert.Empty(t, result.Delete.Profile)
}

func TestConvertDeleteBackupOlderThanToPBM(t *testing.T) {
	t.Run("all fields set", func(t *testing.T) {
		cn, err := NewConfigName("my-s3")
		require.NoError(t, err)

		olderThan := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		cmd := DeleteBackupOlderThan{
			OlderThan:  olderThan,
			Type:       BackupTypeLogical,
			ConfigName: cn,
		}

		result, err := convertCommandToPBM(cmd)
		require.NoError(t, err)

		assert.Equal(t, ctrl.CmdDeleteBackup, result.Cmd)
		require.NotNil(t, result.Delete)
		assert.Empty(t, result.Delete.Backup)
		assert.Equal(t, olderThan.Unix(), result.Delete.OlderThan)
		assert.Equal(t, defs.LogicalBackup, result.Delete.Type)
		assert.Equal(t, "my-s3", result.Delete.Profile)
	})

	t.Run("minimal", func(t *testing.T) {
		olderThan := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		cmd := DeleteBackupOlderThan{
			OlderThan: olderThan,
		}

		result, err := convertCommandToPBM(cmd)
		require.NoError(t, err)

		assert.Equal(t, ctrl.CmdDeleteBackup, result.Cmd)
		require.NotNil(t, result.Delete)
		assert.Equal(t, olderThan.Unix(), result.Delete.OlderThan)
		assert.Empty(t, result.Delete.Type)
		assert.Empty(t, result.Delete.Profile)
	})

	t.Run("main config maps to empty profile", func(t *testing.T) {
		cmd := DeleteBackupOlderThan{
			OlderThan:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			ConfigName: MainConfig,
		}

		result, err := convertCommandToPBM(cmd)
		require.NoError(t, err)

		assert.Empty(t, result.Delete.Profile)
	})
}

func TestConvertCancelBackupCommandToPBM(t *testing.T) {
	cmd := CancelBackupCommand{}

	result, err := convertCommandToPBM(cmd)
	require.NoError(t, err)

	assert.Equal(t, ctrl.CmdCancelBackup, result.Cmd)
	assert.Nil(t, result.Backup)
	assert.Nil(t, result.Restore)
}

func TestConvertAddProfileCommandToPBM(t *testing.T) {
	stg := config.StorageConf{
		Type: storage.Filesystem,
		Filesystem: &fs.Config{
			Path: "/opt/backups",
		},
	}
	cmd := AddProfileCommand{
		Name:    "my-fs",
		storage: stg,
	}

	result, err := convertCommandToPBM(cmd)
	require.NoError(t, err)

	assert.Equal(t, ctrl.CmdAddConfigProfile, result.Cmd)
	require.NotNil(t, result.Profile)
	assert.Equal(t, "my-fs", result.Profile.Name)
	assert.True(t, result.Profile.IsProfile)
	assert.Equal(t, stg, result.Profile.Storage)
}

func TestConvertAddProfileCommandWithoutStorage(t *testing.T) {
	cmd := AddProfileCommand{Name: "bad"}

	_, err := convertCommandToPBM(cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage config not set")
}

func TestConvertRemoveProfileCommandToPBM(t *testing.T) {
	cmd := RemoveProfileCommand{Name: "my-fs"}

	result, err := convertCommandToPBM(cmd)
	require.NoError(t, err)

	assert.Equal(t, ctrl.CmdRemoveConfigProfile, result.Cmd)
	require.NotNil(t, result.Profile)
	assert.Equal(t, "my-fs", result.Profile.Name)
}

func TestConfigNameToPBM(t *testing.T) {
	assert.Equal(t, "", configNameToPBM(MainConfig))
	assert.Equal(t, "", configNameToPBM(ConfigName{}))

	cn, err := NewConfigName("my-profile")
	require.NoError(t, err)
	assert.Equal(t, "my-profile", configNameToPBM(cn))
}
