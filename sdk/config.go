package sdk

import (
	"context"
	"io"
)

// MarshalOptions controls YAML marshaling behavior for configuration output.
type MarshalOptions struct {
	unmasked bool
}

// MarshalOption is a functional option for configuring YAML marshal behavior.
type MarshalOption func(*MarshalOptions)

// WithUnmasked returns a [MarshalOption] that produces YAML with real
// credential values instead of the default masked "***" output. Use this
// when the YAML will be roundtripped (e.g. opened in an editor and re-applied).
// The default (no option) returns masked YAML safe for display.
func WithUnmasked() MarshalOption {
	return func(o *MarshalOptions) { o.unmasked = true }
}

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
	// By default, credential values are masked ("***") for safe display.
	// Pass [WithUnmasked] to get real credential values for roundtripping.
	GetYAML(ctx context.Context, opts ...MarshalOption) ([]byte, error)

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
	// By default, credential values are masked ("***") for safe display.
	// Pass [WithUnmasked] to get real credential values for roundtripping.
	GetProfileYAML(ctx context.Context, name string, opts ...MarshalOption) ([]byte, error)

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
	Enabled          bool               // whether PITR is enabled in the PBM config
	OplogOnly        bool               // if true, only oplog is captured (no base backups)
	OplogSpanMin     float64            // oplog slice interval in minutes (0 = PBM default)
	Priority         map[string]float64 // node priority map for PITR slicing
	Compression      CompressionType    // PITR-specific compression algorithm
	CompressionLevel *int               // PITR-specific compression level
}

// BackupTimeouts holds timeout settings for backup operations.
type BackupTimeouts struct {
	// StartingStatus is the timeout in seconds to wait for a backup agent
	// to pick up the command. Nil means PBM uses its built-in default.
	StartingStatus *uint32

	// BalancerStop is the timeout in seconds to wait for the balancer to stop
	// before starting a backup. 0 means wait indefinitely (PBM default).
	BalancerStop uint32
}

// BackupConfig holds backup-specific configuration.
type BackupConfig struct {
	Compression            CompressionType    // default compression algorithm
	CompressionLevel       *int               // default compression level
	NumParallelCollections int                // default parallelism for collection dump
	OplogSpanMin           float64            // oplog span for backup slicer (0 = PBM default)
	Priority               map[string]float64 // node priority map for backup selection
	Timeouts               *BackupTimeouts    // backup operation timeouts
}

// RestoreConfig holds restore-specific configuration.
type RestoreConfig struct {
	BatchSize              int               // restore batch size
	NumInsertionWorkers    int               // insertion workers per collection
	NumParallelCollections int               // parallel collection count
	NumDownloadWorkers     int               // download parallelism
	MaxDownloadBufferMb    int               // download buffer size in MB
	DownloadChunkMb        int               // download chunk size in MB
	MongodLocation         string            // path to mongod binary (physical restores)
	MongodLocationMap      map[string]string // per-replset mongod paths (physical restores)
	FallbackEnabled        *bool             // enable fallback mode
	AllowPartlyDone        *bool             // allow partial success
}

// StorageProfile represents a named storage configuration that can be used
// as an alternative backup destination alongside the main storage.
type StorageProfile struct {
	Name    ConfigName    // profile name (used in StartBackupOptions.ConfigName)
	Storage StorageConfig // storage settings for this profile
}
