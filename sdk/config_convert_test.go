package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/percona-backup-mongodb/pbm/compress"
	"github.com/percona/percona-backup-mongodb/pbm/config"
	"github.com/percona/percona-backup-mongodb/pbm/storage"
	"github.com/percona/percona-backup-mongodb/pbm/storage/fs"
)

func TestConvertConfig(t *testing.T) {
	t.Run("full config", func(t *testing.T) {
		backupLevel := 6
		pitrLevel := 3
		startingTimeout := uint32(60)
		fallback := true
		partlyDone := false

		cfg := &config.Config{
			Name: "",
			Storage: config.StorageConf{
				Type:       storage.Filesystem,
				Filesystem: &fs.Config{Path: "/data/backups"},
			},
			PITR: &config.PITRConf{
				Enabled:          true,
				OplogOnly:        false,
				OplogSpanMin:     10.0,
				Priority:         map[string]float64{"rs0:27017": 1.0, "rs0:27018": 0.5},
				Compression:      compress.CompressionType("zstd"),
				CompressionLevel: &pitrLevel,
			},
			Backup: &config.BackupConf{
				Compression:            compress.CompressionType("zstd"),
				CompressionLevel:       &backupLevel,
				NumParallelCollections: 4,
				OplogSpanMin:           5.0,
				Priority:               map[string]float64{"rs0:27017": 1.0},
				Timeouts:               &config.BackupTimeouts{Starting: &startingTimeout},
			},
			Restore: &config.RestoreConf{
				BatchSize:              500,
				NumInsertionWorkers:    2,
				NumParallelCollections: 8,
				NumDownloadWorkers:     4,
				MaxDownloadBufferMb:    256,
				DownloadChunkMb:        32,
				MongodLocation:         "/usr/bin/mongod",
				MongodLocationMap:      map[string]string{"rs0": "/opt/mongod"},
				FallbackEnabled:        &fallback,
				AllowPartlyDone:        &partlyDone,
			},
		}

		result := convertConfig(cfg)

		assert.Equal(t, MainConfig, result.ConfigName)
		assert.Equal(t, StorageTypeFilesystem, result.Storage.Type)
		assert.Equal(t, "/data/backups", result.Storage.Path)
		assert.Empty(t, result.Storage.Region)

		// PITR
		assert.NotNil(t, result.PITR)
		assert.True(t, result.PITR.Enabled)
		assert.False(t, result.PITR.OplogOnly)
		assert.Equal(t, 10.0, result.PITR.OplogSpanMin)
		assert.Equal(t, map[string]float64{"rs0:27017": 1.0, "rs0:27018": 0.5}, result.PITR.Priority)
		assert.Equal(t, CompressionTypeZSTD, result.PITR.Compression)
		assert.Equal(t, &pitrLevel, result.PITR.CompressionLevel)

		// Backup
		assert.NotNil(t, result.Backup)
		assert.Equal(t, CompressionTypeZSTD, result.Backup.Compression)
		assert.Equal(t, &backupLevel, result.Backup.CompressionLevel)
		assert.Equal(t, 4, result.Backup.NumParallelCollections)
		assert.Equal(t, 5.0, result.Backup.OplogSpanMin)
		assert.Equal(t, map[string]float64{"rs0:27017": 1.0}, result.Backup.Priority)
		assert.NotNil(t, result.Backup.Timeouts)
		assert.Equal(t, &startingTimeout, result.Backup.Timeouts.StartingStatus)

		// Restore
		assert.NotNil(t, result.Restore)
		assert.Equal(t, 500, result.Restore.BatchSize)
		assert.Equal(t, 2, result.Restore.NumInsertionWorkers)
		assert.Equal(t, 8, result.Restore.NumParallelCollections)
		assert.Equal(t, 4, result.Restore.NumDownloadWorkers)
		assert.Equal(t, 256, result.Restore.MaxDownloadBufferMb)
		assert.Equal(t, 32, result.Restore.DownloadChunkMb)
		assert.Equal(t, "/usr/bin/mongod", result.Restore.MongodLocation)
		assert.Equal(t, map[string]string{"rs0": "/opt/mongod"}, result.Restore.MongodLocationMap)
		assert.Equal(t, &fallback, result.Restore.FallbackEnabled)
		assert.Equal(t, &partlyDone, result.Restore.AllowPartlyDone)
	})

	t.Run("nil sub-configs", func(t *testing.T) {
		cfg := &config.Config{
			Storage: config.StorageConf{
				Type:       storage.Filesystem,
				Filesystem: &fs.Config{Path: "/data/backups"},
			},
		}

		result := convertConfig(cfg)

		assert.Nil(t, result.PITR)
		assert.Nil(t, result.Backup)
		assert.Nil(t, result.Restore)
	})

	t.Run("named config", func(t *testing.T) {
		cfg := &config.Config{
			Name:      "my-profile",
			IsProfile: true,
			Storage: config.StorageConf{
				Type:       storage.Filesystem,
				Filesystem: &fs.Config{Path: "/alt/backups"},
			},
		}

		result := convertConfig(cfg)

		assert.Equal(t, "my-profile", result.ConfigName.String())
	})
}

func TestConvertBackupTimeouts(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, convertBackupTimeouts(nil))
	})

	t.Run("with starting timeout", func(t *testing.T) {
		timeout := uint32(45)
		result := convertBackupTimeouts(&config.BackupTimeouts{Starting: &timeout})

		require.NotNil(t, result)
		require.NotNil(t, result.StartingStatus)
		assert.Equal(t, uint32(45), *result.StartingStatus)
		assert.Zero(t, result.BalancerStop)
	})

	t.Run("with balancer stop timeout", func(t *testing.T) {
		result := convertBackupTimeouts(&config.BackupTimeouts{BalancerStopSec: 120})

		require.NotNil(t, result)
		assert.Nil(t, result.StartingStatus)
		assert.Equal(t, uint32(120), result.BalancerStop)
	})

	t.Run("nil starting timeout", func(t *testing.T) {
		result := convertBackupTimeouts(&config.BackupTimeouts{})

		require.NotNil(t, result)
		assert.Nil(t, result.StartingStatus)
		assert.Zero(t, result.BalancerStop)
	})
}

func TestConvertStorageConfig(t *testing.T) {
	t.Run("filesystem", func(t *testing.T) {
		sc := &config.StorageConf{
			Type:       storage.Filesystem,
			Filesystem: &fs.Config{Path: "/data/backups"},
		}

		result := convertStorageConfig(sc)

		assert.Equal(t, StorageTypeFilesystem, result.Type)
		assert.Equal(t, "/data/backups", result.Path)
		assert.Empty(t, result.Region)
	})

	t.Run("unknown type", func(t *testing.T) {
		sc := &config.StorageConf{
			Type: storage.Type("nonexistent"),
		}

		result := convertStorageConfig(sc)

		assert.True(t, result.Type.IsZero())
	})
}

func TestConvertStorageProfile(t *testing.T) {
	cfg := &config.Config{
		Name:      "s3-backup",
		IsProfile: true,
		Storage: config.StorageConf{
			Type:       storage.Filesystem,
			Filesystem: &fs.Config{Path: "/profile/path"},
		},
	}

	result := convertStorageProfile(cfg)

	assert.Equal(t, "s3-backup", result.Name.String())
	assert.Equal(t, StorageTypeFilesystem, result.Storage.Type)
	assert.Equal(t, "/profile/path", result.Storage.Path)
}
