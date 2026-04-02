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
			Enabled:          cfg.PITR.Enabled,
			OplogOnly:        cfg.PITR.OplogOnly,
			OplogSpanMin:     cfg.PITR.OplogSpanMin,
			Priority:         cfg.PITR.Priority,
			Compression:      convertCompressionType(cfg.PITR.Compression),
			CompressionLevel: cfg.PITR.CompressionLevel,
		}
	}

	if cfg.Backup != nil {
		result.Backup = &BackupConfig{
			Compression:            convertCompressionType(cfg.Backup.Compression),
			CompressionLevel:       cfg.Backup.CompressionLevel,
			NumParallelCollections: cfg.Backup.NumParallelCollections,
			OplogSpanMin:           cfg.Backup.OplogSpanMin,
			Priority:               cfg.Backup.Priority,
			Timeouts:               convertBackupTimeouts(cfg.Backup.Timeouts),
		}
	}

	if cfg.Restore != nil {
		result.Restore = &RestoreConfig{
			BatchSize:              cfg.Restore.BatchSize,
			NumInsertionWorkers:    cfg.Restore.NumInsertionWorkers,
			NumParallelCollections: cfg.Restore.NumParallelCollections,
			NumDownloadWorkers:     cfg.Restore.NumDownloadWorkers,
			MaxDownloadBufferMb:    cfg.Restore.MaxDownloadBufferMb,
			DownloadChunkMb:        cfg.Restore.DownloadChunkMb,
			MongodLocation:         cfg.Restore.MongodLocation,
			MongodLocationMap:      cfg.Restore.MongodLocationMap,
			FallbackEnabled:        cfg.Restore.FallbackEnabled,
			AllowPartlyDone:        cfg.Restore.AllowPartlyDone,
		}
	}

	return result
}

// convertBackupTimeouts converts PBM BackupTimeouts to SDK BackupTimeouts.
// Returns nil for nil input.
func convertBackupTimeouts(t *config.BackupTimeouts) *BackupTimeouts {
	if t == nil {
		return nil
	}
	return &BackupTimeouts{
		StartingStatus: t.Starting,
		BalancerStop:   t.BalancerStopSec,
	}
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
