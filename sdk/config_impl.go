package sdk

import (
	"context"
	"fmt"

	"github.com/percona/percona-backup-mongodb/pbm/connect"
)

type configServiceImpl struct {
	conn connect.Client
}

var _ ConfigService = (*configServiceImpl)(nil)

func (s *configServiceImpl) Get(ctx context.Context) (*Config, error) {
	return nil, fmt.Errorf("config get: not implemented")
}

func (s *configServiceImpl) GetYAML(ctx context.Context) ([]byte, error) {
	return nil, fmt.Errorf("config get yaml: not implemented")
}

func (s *configServiceImpl) ListProfiles(ctx context.Context) ([]StorageProfile, error) {
	return nil, fmt.Errorf("config list profiles: not implemented")
}

func (s *configServiceImpl) GetProfile(ctx context.Context, name string) (*StorageProfile, error) {
	return nil, fmt.Errorf("config get profile: not implemented")
}

func (s *configServiceImpl) GetProfileYAML(ctx context.Context, name string) ([]byte, error) {
	return nil, fmt.Errorf("config get profile yaml: not implemented")
}
