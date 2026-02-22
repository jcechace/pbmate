package sdk

import (
	"context"
	"io"
)

// ConfigService provides access to PBM configuration and storage profiles.
//
// PBM uses a main configuration (identified by [MainConfig]) for the default
// storage and settings, plus optional named storage profiles for additional
// backup destinations.
//
// Example — read configuration and list profiles:
//
//	cfg, err := client.Config.Get(ctx)
//	fmt.Printf("storage: %s %s\n", cfg.Storage.Type, cfg.Storage.Path)
//
//	profiles, _ := client.Config.ListProfiles(ctx)
//	for _, p := range profiles {
//	    fmt.Printf("profile %s: %s\n", p.Name, p.Storage.Type)
//	}
type ConfigService interface {
	// Get returns the main PBM configuration. Returns an error if PBM
	// has not been configured yet.
	Get(ctx context.Context) (*Config, error)

	// GetYAML returns the main PBM configuration as raw YAML bytes.
	GetYAML(ctx context.Context) ([]byte, error)

	// SetYAML replaces the main PBM configuration from YAML read from r.
	// This is a direct write — no command dispatch or agent involvement.
	SetYAML(ctx context.Context, r io.Reader) error

	// ListProfiles returns all named storage profiles. Returns an empty
	// slice if no profiles are configured.
	ListProfiles(ctx context.Context) ([]StorageProfile, error)

	// GetProfile returns a storage profile by name. Returns [ErrNotFound]
	// if the profile does not exist.
	GetProfile(ctx context.Context, name string) (*StorageProfile, error)

	// GetProfileYAML returns a storage profile as raw YAML bytes. Returns
	// [ErrNotFound] if the profile does not exist.
	GetProfileYAML(ctx context.Context, name string) ([]byte, error)

	// SetProfile creates or replaces a named storage profile from YAML read
	// from r. The name parameter identifies the profile; the YAML must contain
	// a storage configuration. This is dispatched as a command — the agent
	// validates storage accessibility before saving.
	SetProfile(ctx context.Context, name string, r io.Reader) (CommandResult, error)

	// RemoveProfile deletes a named storage profile. This is dispatched as a
	// command — the agent clears associated backup metadata before removing.
	RemoveProfile(ctx context.Context, name string) (CommandResult, error)

	// Resync instructs PBM agents to re-read backup metadata from storage.
	// This is useful after manual storage changes or configuration updates.
	// The cmd parameter determines which storage to resync — see
	// [ResyncMain], [ResyncProfile], and [ResyncAllProfiles].
	Resync(ctx context.Context, cmd ResyncCommand) (CommandResult, error)
}

// Config represents the PBM configuration. Optional sections (PITR, Backup,
// Restore) are nil when not configured.
type Config struct {
	ConfigName ConfigName     // always MainConfig for the main configuration
	Storage    StorageConfig  // primary backup storage settings
	PITR       *PITRConfig    // nil if PITR section is not configured
	Backup     *BackupConfig  // nil if backup section is not configured
	Restore    *RestoreConfig // nil if restore section is not configured
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
	Enabled   bool // whether PITR is enabled in the PBM config
	OplogOnly bool // if true, only oplog is captured (no base backups)
}

// BackupConfig holds backup-specific configuration.
type BackupConfig struct {
	Compression CompressionType // default compression for new backups
}

// RestoreConfig holds restore-specific configuration. Currently empty;
// reserved for future restore-related settings.
type RestoreConfig struct {
}

// StorageProfile represents a named storage configuration that can be used
// as an alternative backup destination alongside the main storage.
type StorageProfile struct {
	Name    ConfigName    // profile name (used in StartBackupOptions.ConfigName)
	Storage StorageConfig // storage settings for this profile
}
