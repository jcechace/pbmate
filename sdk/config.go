package sdk

import (
	"context"
	"io"
)

// ConfigService provides access to PBM configuration and storage profiles.
type ConfigService interface {
	// Get returns the main PBM configuration.
	Get(ctx context.Context) (*Config, error)

	// GetYAML returns the main PBM configuration as raw YAML.
	GetYAML(ctx context.Context) ([]byte, error)

	// SetYAML replaces the main PBM configuration from YAML read from r.
	// This is a direct write — no command dispatch or agent involvement.
	SetYAML(ctx context.Context, r io.Reader) error

	// ListProfiles returns all named storage profiles.
	ListProfiles(ctx context.Context) ([]StorageProfile, error)

	// GetProfile returns a storage profile by name.
	GetProfile(ctx context.Context, name string) (*StorageProfile, error)

	// GetProfileYAML returns a storage profile as raw YAML.
	GetProfileYAML(ctx context.Context, name string) ([]byte, error)

	// SetProfile creates or replaces a named storage profile from YAML read
	// from r. The name parameter identifies the profile; the YAML must contain
	// a storage configuration. This is dispatched as a command — the agent
	// validates storage accessibility before saving.
	SetProfile(ctx context.Context, name string, r io.Reader) (CommandResult, error)

	// RemoveProfile deletes a named storage profile. This is dispatched as a
	// command — the agent clears associated backup metadata before removing.
	RemoveProfile(ctx context.Context, name string) (CommandResult, error)
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
