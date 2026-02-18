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

func (s *restoreServiceImpl) Start(ctx context.Context, opts StartRestoreOptions) (RestoreResult, error) {
	cmd := RestoreCommand{
		Name:       time.Now().UTC().Format(time.RFC3339Nano),
		BackupName: opts.BackupName,
		PITRTarget: opts.PITRTarget,
		Namespaces: opts.Namespaces,
	}

	s.log.InfoContext(ctx, "starting restore", "name", cmd.Name, "backup", cmd.BackupName)
	result, err := s.cmds.Send(ctx, cmd)
	if err != nil {
		return RestoreResult{}, fmt.Errorf("start restore: %w", err)
	}

	return RestoreResult{
		CommandResult: result,
		Name:          cmd.Name,
	}, nil
}

func (s *restoreServiceImpl) Wait(ctx context.Context, name string, opts RestoreWaitOptions) (*Restore, error) {
	interval := opts.PollInterval
	if interval == 0 {
		interval = time.Second
	}

	var last *Restore
	timer := time.NewTimer(0) // fires immediately for first check
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return last, ctx.Err()
		case <-timer.C:
		}

		r, err := s.Get(ctx, name)
		if err != nil {
			if ctx.Err() != nil {
				return last, ctx.Err()
			}
			if !errors.Is(err, ErrNotFound) {
				return last, fmt.Errorf("wait for restore %q: %w", name, err)
			}
			s.log.DebugContext(ctx, "restore not found yet, retrying", "name", name)
		} else {
			last = r
			s.log.DebugContext(ctx, "polling restore status", "name", name, "status", r.Status)
			if opts.OnProgress != nil {
				opts.OnProgress(r)
			}
			if r.Status.IsTerminal() {
				s.log.InfoContext(ctx, "restore reached terminal status", "name", name, "status", r.Status)
				if r.Status.Equal(StatusError) || r.Status.Equal(StatusPartlyDone) {
					return r, &OperationError{Name: name, Message: r.Error}
				}
				return r, nil
			}
		}

		timer.Reset(interval)
	}
}
