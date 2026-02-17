package sdk

import (
	"context"
	"fmt"

	"github.com/percona/percona-backup-mongodb/pbm/connect"
)

type restoreServiceImpl struct {
	conn connect.Client
}

var _ RestoreService = (*restoreServiceImpl)(nil)

func (s *restoreServiceImpl) List(ctx context.Context, opts ListRestoresOptions) ([]Restore, error) {
	return nil, fmt.Errorf("restore list: not implemented")
}

func (s *restoreServiceImpl) Get(ctx context.Context, name string) (*Restore, error) {
	return nil, fmt.Errorf("restore get: not implemented")
}

func (s *restoreServiceImpl) GetByOpID(ctx context.Context, opid string) (*Restore, error) {
	return nil, fmt.Errorf("restore get by opid: not implemented")
}
