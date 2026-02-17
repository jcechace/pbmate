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

func (s *logServiceImpl) Query(ctx context.Context, opts LogQuery) ([]LogEntry, error) {
	return nil, fmt.Errorf("log query: not implemented")
}

func (s *logServiceImpl) Follow(ctx context.Context, opts LogQuery) (<-chan LogEntry, <-chan error) {
	entries := make(chan LogEntry)
	errs := make(chan error, 1)
	errs <- fmt.Errorf("log follow: not implemented")
	close(entries)
	close(errs)
	return entries, errs
}
