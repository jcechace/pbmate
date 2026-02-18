package sdk

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/percona/percona-backup-mongodb/pbm/connect"
	pbmerrors "github.com/percona/percona-backup-mongodb/pbm/errors"
	"github.com/percona/percona-backup-mongodb/pbm/restore"
)

type restoreServiceImpl struct {
	conn connect.Client
	cmds CommandService
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

	result, err := s.cmds.Send(ctx, cmd)
	if err != nil {
		return RestoreResult{}, fmt.Errorf("start restore: %w", err)
	}

	return RestoreResult{
		CommandResult: result,
		Name:          cmd.Name,
	}, nil
}
