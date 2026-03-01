//go:build integration

package integtest

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

func TestConfigGet(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedConfig(t, newMainConfig())

	cfg, err := h.client.Config.Get(ctx)
	require.NoError(t, err)

	// Main config identity.
	assert.True(t, cfg.ConfigName.Equal(sdk.MainConfig))

	// Storage fields.
	assert.True(t, cfg.Storage.Type.Equal(sdk.StorageTypeFilesystem))
	assert.Equal(t, "/tmp/pbm-backups", cfg.Storage.Path)
	assert.Empty(t, cfg.Storage.Region)
}

func TestConfigGetNotFound(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	_, err := h.client.Config.Get(ctx)
	require.ErrorIs(t, err, sdk.ErrNotFound)
}

func TestConfigGetWithAllSections(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedConfig(t, newMainConfig(
		withConfigPITR(true),
		withConfigBackup(4),
		withConfigRestore(500, 8),
	))

	cfg, err := h.client.Config.Get(ctx)
	require.NoError(t, err)

	// PITR section.
	require.NotNil(t, cfg.PITR)
	assert.True(t, cfg.PITR.Enabled)
	assert.True(t, cfg.PITR.OplogOnly)
	assert.Equal(t, float64(10), cfg.PITR.OplogSpanMin)
	assert.True(t, cfg.PITR.Compression.Equal(sdk.CompressionTypeZSTD))
	require.NotNil(t, cfg.PITR.CompressionLevel)
	assert.Equal(t, 3, *cfg.PITR.CompressionLevel)
	assert.Equal(t, 1.0, cfg.PITR.Priority["rs0:27017"])

	// Backup section.
	require.NotNil(t, cfg.Backup)
	assert.True(t, cfg.Backup.Compression.Equal(sdk.CompressionTypeZSTD))
	assert.Equal(t, 4, cfg.Backup.NumParallelCollections)
	assert.Equal(t, float64(5), cfg.Backup.OplogSpanMin)
	require.NotNil(t, cfg.Backup.Timeouts)
	require.NotNil(t, cfg.Backup.Timeouts.StartingStatus)
	assert.Equal(t, uint32(120), *cfg.Backup.Timeouts.StartingStatus)

	// Restore section.
	require.NotNil(t, cfg.Restore)
	assert.Equal(t, 500, cfg.Restore.BatchSize)
	assert.Equal(t, 2, cfg.Restore.NumInsertionWorkers)
	assert.Equal(t, 8, cfg.Restore.NumParallelCollections)
	assert.Equal(t, 4, cfg.Restore.NumDownloadWorkers)
	assert.Equal(t, 256, cfg.Restore.MaxDownloadBufferMb)
	assert.Equal(t, 32, cfg.Restore.DownloadChunkMb)
	assert.Equal(t, "/usr/bin/mongod", cfg.Restore.MongodLocation)
}

func TestConfigGetDefaultSections(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed minimal config — no PITR/Backup/Restore sections.
	// PBM's GetConfig initializes nil sub-configs to empty structs
	// and sets default compression (S2).
	h.seedConfig(t, newMainConfig())

	cfg, err := h.client.Config.Get(ctx)
	require.NoError(t, err)

	// Sub-sections are non-nil (PBM fills defaults).
	require.NotNil(t, cfg.PITR)
	assert.False(t, cfg.PITR.Enabled)

	require.NotNil(t, cfg.Backup)
	// PBM sets default compression to S2.
	assert.True(t, cfg.Backup.Compression.Equal(sdk.CompressionTypeS2))

	require.NotNil(t, cfg.Restore)
	assert.Equal(t, 0, cfg.Restore.BatchSize)
}

func TestConfigGetYAML(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedConfig(t, newMainConfig())

	yamlBytes, err := h.client.Config.GetYAML(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, yamlBytes)

	yaml := string(yamlBytes)
	assert.Contains(t, yaml, "storage:")
	assert.Contains(t, yaml, "filesystem")
}

func TestConfigGetYAMLNotFound(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	_, err := h.client.Config.GetYAML(ctx)
	require.ErrorIs(t, err, sdk.ErrNotFound)
}

func TestConfigSetYAML(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed an initial config so SetYAML has something to replace.
	h.seedConfig(t, newMainConfig())

	// Set a new config via YAML.
	newYAML := `
storage:
  type: filesystem
  filesystem:
    path: /data/new-backups
`
	err := h.client.Config.SetYAML(ctx, strings.NewReader(newYAML))
	require.NoError(t, err)

	// Read back and verify the change.
	cfg, err := h.client.Config.Get(ctx)
	require.NoError(t, err)
	assert.True(t, cfg.Storage.Type.Equal(sdk.StorageTypeFilesystem))
	assert.Equal(t, "/data/new-backups", cfg.Storage.Path)
}

func TestConfigSetYAMLOnEmptyDB(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// SetYAML should work even when no config exists yet.
	newYAML := `
storage:
  type: filesystem
  filesystem:
    path: /tmp/fresh-config
`
	err := h.client.Config.SetYAML(ctx, strings.NewReader(newYAML))
	require.NoError(t, err)

	cfg, err := h.client.Config.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "/tmp/fresh-config", cfg.Storage.Path)
}

func TestConfigListProfiles(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed main config + 2 profiles.
	h.seedConfig(t, newMainConfig())
	h.seedConfig(t, newMainConfig(
		withConfigProfile("s3-prod"),
		withConfigS3Storage("my-bucket", "us-east-1"),
	))
	h.seedConfig(t, newMainConfig(
		withConfigProfile("local-dev"),
	))

	profiles, err := h.client.Config.ListProfiles(ctx)
	require.NoError(t, err)
	require.Len(t, profiles, 2)

	// Collect names for order-independent assertion.
	names := make([]string, len(profiles))
	for i, p := range profiles {
		names[i] = p.Name.String()
	}
	assert.Contains(t, names, "s3-prod")
	assert.Contains(t, names, "local-dev")
}

func TestConfigListProfilesEmpty(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Only main config, no profiles.
	h.seedConfig(t, newMainConfig())

	profiles, err := h.client.Config.ListProfiles(ctx)
	require.NoError(t, err)
	assert.Empty(t, profiles)
}

func TestConfigGetProfile(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedConfig(t, newMainConfig(
		withConfigProfile("aws-prod"),
		withConfigS3Storage("prod-backups", "eu-west-1"),
	))

	profile, err := h.client.Config.GetProfile(ctx, "aws-prod")
	require.NoError(t, err)

	expected, _ := sdk.NewConfigName("aws-prod")
	assert.True(t, profile.Name.Equal(expected))
	assert.True(t, profile.Storage.Type.Equal(sdk.StorageTypeS3))
	assert.Equal(t, "eu-west-1", profile.Storage.Region)
	assert.Contains(t, profile.Storage.Path, "prod-backups")
}

func TestConfigGetProfileNotFound(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	_, err := h.client.Config.GetProfile(ctx, "nonexistent")
	require.ErrorIs(t, err, sdk.ErrNotFound)
}

func TestConfigGetProfileYAML(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedConfig(t, newMainConfig(
		withConfigProfile("yaml-profile"),
		withConfigS3Storage("yaml-bucket", "ap-south-1"),
	))

	yamlBytes, err := h.client.Config.GetProfileYAML(ctx, "yaml-profile")
	require.NoError(t, err)
	require.NotEmpty(t, yamlBytes)

	yaml := string(yamlBytes)
	assert.Contains(t, yaml, "storage:")
	assert.Contains(t, yaml, "s3")
}

func TestConfigGetProfileYAMLNotFound(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	_, err := h.client.Config.GetProfileYAML(ctx, "ghost")
	require.ErrorIs(t, err, sdk.ErrNotFound)
}

// --- GetYAML / GetProfileYAML credential masking ---

func TestConfigGetYAMLMasked(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedConfig(t, newMainConfig(
		withConfigS3Storage("my-bucket", "us-east-1"),
		withConfigS3Credentials("AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
	))

	yamlBytes, err := h.client.Config.GetYAML(ctx)
	require.NoError(t, err)

	yaml := string(yamlBytes)
	// Default (masked): credentials should be "***", not the real values.
	assert.Contains(t, yaml, "access-key-id: '***'")
	assert.Contains(t, yaml, "secret-access-key: '***'")
	assert.NotContains(t, yaml, "AKIAIOSFODNN7EXAMPLE")
	assert.NotContains(t, yaml, "wJalrXUtnFEMI")
}

func TestConfigGetYAMLUnmasked(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedConfig(t, newMainConfig(
		withConfigS3Storage("my-bucket", "us-east-1"),
		withConfigS3Credentials("AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
	))

	yamlBytes, err := h.client.Config.GetYAML(ctx, sdk.WithUnmasked())
	require.NoError(t, err)

	yaml := string(yamlBytes)
	// Unmasked: real credential values should appear.
	assert.Contains(t, yaml, "AKIAIOSFODNN7EXAMPLE")
	assert.Contains(t, yaml, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
	assert.NotContains(t, yaml, "'***'")
}

func TestConfigGetProfileYAMLUnmasked(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedConfig(t, newMainConfig(
		withConfigProfile("s3-profile"),
		withConfigS3Storage("profile-bucket", "eu-west-1"),
		withConfigS3Credentials("PROFILEKEY123", "PROFILESECRET456"),
	))

	yamlBytes, err := h.client.Config.GetProfileYAML(ctx, "s3-profile", sdk.WithUnmasked())
	require.NoError(t, err)

	yaml := string(yamlBytes)
	assert.Contains(t, yaml, "PROFILEKEY123")
	assert.Contains(t, yaml, "PROFILESECRET456")
	assert.NotContains(t, yaml, "'***'")
}
