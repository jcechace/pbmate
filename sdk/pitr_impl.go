package sdk

import (
	"context"
	"fmt"

	"github.com/percona/percona-backup-mongodb/pbm/connect"
)

type pitrServiceImpl struct {
	conn connect.Client
}

var _ PITRService = (*pitrServiceImpl)(nil)

func (s *pitrServiceImpl) Status(ctx context.Context) (*PITRStatus, error) {
	return nil, fmt.Errorf("pitr status: not implemented")
}

func (s *pitrServiceImpl) Timelines(ctx context.Context) ([]Timeline, error) {
	return nil, fmt.Errorf("pitr timelines: not implemented")
}
