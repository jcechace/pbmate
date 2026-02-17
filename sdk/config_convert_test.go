package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/percona/percona-backup-mongodb/pbm/compress"
	"github.com/percona/percona-backup-mongodb/pbm/config"
	"github.com/percona/percona-backup-mongodb/pbm/storage"
	"github.com/percona/percona-backup-mongodb/pbm/storage/fs"
)

func TestConvertConfig(t *testing.T) {
	t.Run("full config", func(t *testing.T) {
		cfg := &config.Config{
			Name: "",
			Storage: config.StorageConf{
				Type:       storage.Filesystem,
				Filesystem: &fs.Config{Path: "/data/backups"},
			},
			PITR: &config.PITRConf{
				Enabled:   true,
				OplogOnly: false,
			},
			Backup: &config.BackupConf{
				Compression: compress.CompressionType("zstd"),
			},
			Restore: &config.RestoreConf{},
		}

		result := convertConfig(cfg)

		assert.Equal(t, MainConfig, result.ConfigName)
		assert.Equal(t, StorageFilesystem, result.Storage.Type)
		assert.Equal(t, "/data/backups", result.Storage.Path)
		assert.Empty(t, result.Storage.Region)

		assert.NotNil(t, result.PITR)
		assert.True(t, result.PITR.Enabled)
		assert.False(t, result.PITR.OplogOnly)

		assert.NotNil(t, result.Backup)
		assert.Equal(t, CompressionZSTD, result.Backup.Compression)

		assert.NotNil(t, result.Restore)
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

func TestConvertStorageConfig(t *testing.T) {
	t.Run("filesystem", func(t *testing.T) {
		sc := &config.StorageConf{
			Type:       storage.Filesystem,
			Filesystem: &fs.Config{Path: "/data/backups"},
		}

		result := convertStorageConfig(sc)

		assert.Equal(t, StorageFilesystem, result.Type)
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
	assert.Equal(t, StorageFilesystem, result.Storage.Type)
	assert.Equal(t, "/profile/path", result.Storage.Path)
}
