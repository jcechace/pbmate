package sdk

import "context"

// ConfigService provides read access to PBM configuration and storage profiles.
type ConfigService interface {
	// Get returns the main PBM configuration.
	Get(ctx context.Context) (*Config, error)

	// GetYAML returns the main PBM configuration as raw YAML.
	GetYAML(ctx context.Context) ([]byte, error)

	// ListProfiles returns all named storage profiles.
	ListProfiles(ctx context.Context) ([]StorageProfile, error)

	// GetProfile returns a storage profile by name.
	GetProfile(ctx context.Context, name string) (*StorageProfile, error)

	// GetProfileYAML returns a storage profile as raw YAML.
	GetProfileYAML(ctx context.Context, name string) ([]byte, error)
}

// Config represents the PBM configuration.
type Config struct {
	ConfigName ConfigName
	Storage    StorageConfig
	PITR       *PITRConfig
	Backup     *BackupConfig
	Restore    *RestoreConfig
}

// StorageConfig describes the configured backup storage.
type StorageConfig struct {
	// Type is the storage backend type (s3, gcs, azure, filesystem, etc.).
	Type StorageType

	// Path is the bucket/prefix for cloud storage or directory for filesystem.
	Path string

	// Region is the cloud region (empty for filesystem).
	Region string
}

// PITRConfig holds PITR-specific configuration.
type PITRConfig struct {
	Enabled   bool
	OplogOnly bool
}

// BackupConfig holds backup-specific configuration.
type BackupConfig struct {
	Compression CompressionType
}

// RestoreConfig holds restore-specific configuration.
type RestoreConfig struct {
}

// StorageProfile represents a named storage configuration.
type StorageProfile struct {
	Name    ConfigName
	Storage StorageConfig
}
