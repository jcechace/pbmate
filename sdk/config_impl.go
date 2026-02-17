package sdk

import (
	"context"
	"errors"
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/percona/percona-backup-mongodb/pbm/config"
	"github.com/percona/percona-backup-mongodb/pbm/connect"
)

type configServiceImpl struct {
	conn connect.Client
}

var _ ConfigService = (*configServiceImpl)(nil)

func (s *configServiceImpl) Get(ctx context.Context) (*Config, error) {
	cfg, err := config.GetConfig(ctx, s.conn)
	if err != nil {
		if errors.Is(err, config.ErrMissedConfig) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get config: %w", err)
	}

	result := convertConfig(cfg)
	return &result, nil
}

func (s *configServiceImpl) GetYAML(ctx context.Context) ([]byte, error) {
	cfg, err := config.GetConfig(ctx, s.conn)
	if err != nil {
		if errors.Is(err, config.ErrMissedConfig) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get config yaml: %w", err)
	}

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal config yaml: %w", err)
	}
	return b, nil
}

func (s *configServiceImpl) ListProfiles(ctx context.Context) ([]StorageProfile, error) {
	profiles, err := config.ListProfiles(ctx, s.conn)
	if err != nil {
		return nil, fmt.Errorf("list profiles: %w", err)
	}

	result := make([]StorageProfile, len(profiles))
	for i := range profiles {
		result[i] = convertStorageProfile(&profiles[i])
	}
	return result, nil
}

func (s *configServiceImpl) GetProfile(ctx context.Context, name string) (*StorageProfile, error) {
	profile, err := config.GetProfile(ctx, s.conn, name)
	if err != nil {
		if errors.Is(err, config.ErrMissedConfigProfile) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get profile %q: %w", name, err)
	}

	result := convertStorageProfile(profile)
	return &result, nil
}

func (s *configServiceImpl) GetProfileYAML(ctx context.Context, name string) ([]byte, error) {
	profile, err := config.GetProfile(ctx, s.conn, name)
	if err != nil {
		if errors.Is(err, config.ErrMissedConfigProfile) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get profile yaml %q: %w", name, err)
	}

	b, err := yaml.Marshal(profile)
	if err != nil {
		return nil, fmt.Errorf("marshal profile yaml: %w", err)
	}
	return b, nil
}
