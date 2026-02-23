package sdk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/percona/percona-backup-mongodb/pbm/connect"
	"github.com/percona/percona-backup-mongodb/pbm/ctrl"
	pbmerrors "github.com/percona/percona-backup-mongodb/pbm/errors"
	"github.com/percona/percona-backup-mongodb/pbm/restore"
)

type restoreServiceImpl struct {
	conn connect.Client
	cmds *commandServiceImpl
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
	if err := s.cmds.validateAndCheckLock(ctx, cmd); err != nil {
		return RestoreResult{}, fmt.Errorf("start restore: %w", err)
	}

	// PBM uses RFC 3339 Nano (sub-second precision) for restore names,
	// unlike backup names which use second-precision RFC 3339.
	name := time.Now().UTC().Format(time.RFC3339Nano)

	// Inject the auto-generated name and convert to PBM command.
	var pbmCmd ctrl.Cmd
	switch c := cmd.(type) {
	case StartSnapshotRestore:
		c.name = name
		pbmCmd = convertStartSnapshotRestoreToPBM(c)
	case StartPITRRestore:
		c.name = name
		pbmCmd = convertStartPITRRestoreToPBM(c)
	default:
		panic(fmt.Sprintf("unreachable: unknown StartRestoreCommand type %T", cmd))
	}

	s.log.InfoContext(ctx, "starting restore", "name", name, "type", fmt.Sprintf("%T", cmd))
	result, err := s.cmds.dispatch(ctx, pbmCmd)
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
