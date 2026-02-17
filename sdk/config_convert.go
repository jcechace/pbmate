package sdk

import (
	"github.com/percona/percona-backup-mongodb/pbm/config"
)

// convertConfig converts a PBM Config to an SDK Config.
func convertConfig(cfg *config.Config) Config {
	result := Config{
		ConfigName: convertConfigName(cfg.Name),
		Storage:    convertStorageConfig(&cfg.Storage),
	}

	if cfg.PITR != nil {
		result.PITR = &PITRConfig{
			Enabled:   cfg.PITR.Enabled,
			OplogOnly: cfg.PITR.OplogOnly,
		}
	}

	if cfg.Backup != nil {
		result.Backup = &BackupConfig{
			Compression: convertCompressionType(cfg.Backup.Compression),
		}
	}

	if cfg.Restore != nil {
		result.Restore = &RestoreConfig{}
	}

	return result
}

// convertStorageConfig converts a PBM StorageConf to an SDK StorageConfig.
// Uses the PBM StorageConf helper methods for Path and Region extraction.
func convertStorageConfig(sc *config.StorageConf) StorageConfig {
	return StorageConfig{
		Type:   convertStorageType(sc.Type),
		Path:   sc.Path(),
		Region: sc.Region(),
	}
}

// convertStorageProfile converts a PBM Config (with IsProfile=true) to an SDK StorageProfile.
func convertStorageProfile(cfg *config.Config) StorageProfile {
	return StorageProfile{
		Name:    convertConfigName(cfg.Name),
		Storage: convertStorageConfig(&cfg.Storage),
	}
}
