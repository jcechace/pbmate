package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/percona-backup-mongodb/pbm/compress"
	"github.com/percona/percona-backup-mongodb/pbm/ctrl"
	"github.com/percona/percona-backup-mongodb/pbm/defs"
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

func TestConvertDeleteBackupCommandToPBM(t *testing.T) {
	cmd := DeleteBackupCommand{
		Name: "2024-01-15T10:30:00Z",
	}

	result, err := convertCommandToPBM(cmd)
	require.NoError(t, err)

	assert.Equal(t, ctrl.CmdDeleteBackup, result.Cmd)
	require.NotNil(t, result.Delete)
	assert.Equal(t, "2024-01-15T10:30:00Z", result.Delete.Backup)
}

func TestConvertCancelBackupCommandToPBM(t *testing.T) {
	cmd := CancelBackupCommand{}

	result, err := convertCommandToPBM(cmd)
	require.NoError(t, err)

	assert.Equal(t, ctrl.CmdCancelBackup, result.Cmd)
	assert.Nil(t, result.Backup)
	assert.Nil(t, result.Restore)
}

func TestConfigNameToPBM(t *testing.T) {
	assert.Equal(t, "", configNameToPBM(MainConfig))
	assert.Equal(t, "", configNameToPBM(ConfigName{}))

	cn, err := NewConfigName("my-profile")
	require.NoError(t, err)
	assert.Equal(t, "my-profile", configNameToPBM(cn))
}
