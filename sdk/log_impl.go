package sdk

import (
	"context"
	"fmt"

	"github.com/percona/percona-backup-mongodb/pbm/connect"
)

type logServiceImpl struct {
	conn connect.Client
}

var _ LogService = (*logServiceImpl)(nil)

func (s *logServiceImpl) Get(ctx context.Context, limit int64) ([]LogEntry, error) {
	return nil, fmt.Errorf("log get: not implemented")
}

func (s *logServiceImpl) Follow(ctx context.Context) (<-chan LogEntry, <-chan error) {
	entries := make(chan LogEntry)
	errs := make(chan error, 1)
	errs <- fmt.Errorf("log follow: not implemented")
	close(entries)
	close(errs)
	return entries, errs
}
