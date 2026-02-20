package sdk

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/percona/percona-backup-mongodb/pbm/connect"
	"github.com/percona/percona-backup-mongodb/pbm/log"
)

type logServiceImpl struct {
	conn connect.Client
	log  *slog.Logger
}

var _ LogService = (*logServiceImpl)(nil)

func (s *logServiceImpl) Get(ctx context.Context, limit int64) ([]LogEntry, error) {
	// Default to Info severity (includes Fatal, Error, Warning, Info).
	req := &log.LogRequest{
		LogKeys: log.LogKeys{
			Severity: log.Info,
		},
	}

	entries, err := log.LogGet(ctx, s.conn, req, limit)
	if err != nil {
		return nil, fmt.Errorf("get logs: %w", err)
	}

	result := make([]LogEntry, len(entries.Data))
	for i := range entries.Data {
		result[i] = convertLogEntry(&entries.Data[i])
	}
	return result, nil
}

func (s *logServiceImpl) Follow(ctx context.Context) (<-chan LogEntry, <-chan error) {
	// Default to Info severity (includes Fatal, Error, Warning, Info).
	req := &log.LogRequest{
		LogKeys: log.LogKeys{
			Severity: log.Info,
		},
	}

	pbmEntries, pbmErrs := log.Follow(ctx, s.conn, req, false)

	entries := make(chan LogEntry)
	errs := make(chan error, 1)

	go func() {
		defer close(entries)
		defer close(errs)

		for e := range pbmEntries {
			// Select on ctx.Done to avoid blocking on the unbuffered
			// send if the consumer stopped reading (e.g. follow mode
			// was toggled off while an entry was ready to deliver).
			select {
			case entries <- convertLogEntry(e):
			case <-ctx.Done():
				return
			}
		}

		// Forward any error from the PBM follow channel.
		for err := range pbmErrs {
			errs <- fmt.Errorf("follow logs: %w", err)
			return
		}
	}()

	return entries, errs
}
