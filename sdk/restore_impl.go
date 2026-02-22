package sdk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/percona/percona-backup-mongodb/pbm/connect"
	pbmerrors "github.com/percona/percona-backup-mongodb/pbm/errors"
	"github.com/percona/percona-backup-mongodb/pbm/restore"
)

type restoreServiceImpl struct {
	conn connect.Client
	cmds CommandService
	log  *slog.Logger
}

var _ RestoreService = (*restoreServiceImpl)(nil)

func (s *restoreServiceImpl) List(ctx context.Context, opts ListRestoresOptions) ([]Restore, error) {
	limit := int64(opts.Limit)

	metas, err := restore.RestoreList(ctx, s.conn, limit)
	if err != nil {
		return nil, fmt.Errorf("list restores: %w", err)
	}

	result := make([]Restore, len(metas))
	for i := range metas {
		result[i] = convertRestore(&metas[i])
	}
	return result, nil
}

func (s *restoreServiceImpl) Get(ctx context.Context, name string) (*Restore, error) {
	meta, err := restore.GetRestoreMeta(ctx, s.conn, name)
	if err != nil {
		if errors.Is(err, pbmerrors.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get restore %q: %w", name, err)
	}

	r := convertRestore(meta)
	return &r, nil
}

func (s *restoreServiceImpl) GetByOpID(ctx context.Context, opid string) (*Restore, error) {
	meta, err := restore.GetRestoreMetaByOPID(ctx, s.conn, opid)
	if err != nil {
		if errors.Is(err, pbmerrors.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get restore by opid %q: %w", opid, err)
	}

	r := convertRestore(meta)
	return &r, nil
}

func (s *restoreServiceImpl) Start(ctx context.Context, cmd StartRestoreCommand) (RestoreResult, error) {
	name := time.Now().UTC().Format(time.RFC3339Nano)

	// Inject the auto-generated name into the concrete command type.
	switch c := cmd.(type) {
	case StartSnapshotRestore:
		c.name = name
		cmd = c
	case StartPITRRestore:
		c.name = name
		cmd = c
	}

	s.log.InfoContext(ctx, "starting restore", "name", name, "type", fmt.Sprintf("%T", cmd))
	result, err := s.cmds.Send(ctx, cmd)
	if err != nil {
		return RestoreResult{}, fmt.Errorf("start restore: %w", err)
	}

	return RestoreResult{
		CommandResult: result,
		Name:          name,
	}, nil
}

func (s *restoreServiceImpl) Wait(ctx context.Context, name string, opts RestoreWaitOptions) (*Restore, error) {
	return waitForTerminal(ctx, name, opts.PollInterval, waitParams[*Restore]{
		get:        s.Get,
		status:     func(r *Restore) Status { return r.Status },
		errMsg:     func(r *Restore) string { return r.Error },
		onProgress: opts.OnProgress,
		log:        s.log,
		entity:     "restore",
	})
}
