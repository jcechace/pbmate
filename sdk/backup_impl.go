package sdk

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/percona/percona-backup-mongodb/pbm/backup"
	"github.com/percona/percona-backup-mongodb/pbm/connect"
	pbmerrors "github.com/percona/percona-backup-mongodb/pbm/errors"
)

type backupServiceImpl struct {
	conn connect.Client
	cmds CommandService
}

var _ BackupService = (*backupServiceImpl)(nil)

func (s *backupServiceImpl) List(ctx context.Context, opts ListBackupsOptions) ([]Backup, error) {
	// Fetch all backups via PBM's internal query (sorted newest-first).
	// Filtering by ConfigName/Type is done in memory — backup counts are
	// small enough that this is always practical.
	metas, err := backup.BackupsList(ctx, s.conn, 0)
	if err != nil {
		return nil, fmt.Errorf("list backups: %w", err)
	}

	var result []Backup
	for i := range metas {
		if !opts.ConfigName.IsZero() && configNameToPBM(opts.ConfigName) != metas[i].Store.Name {
			continue
		}
		if !opts.Type.IsZero() && opts.Type.String() != string(metas[i].Type) {
			continue
		}

		result = append(result, convertBackup(&metas[i]))

		if opts.Limit > 0 && len(result) >= opts.Limit {
			break
		}
	}

	return result, nil
}

func (s *backupServiceImpl) Get(ctx context.Context, name string) (*Backup, error) {
	mgr := backup.NewDBManager(s.conn)
	meta, err := mgr.GetBackupByName(ctx, name)
	if err != nil {
		if errors.Is(err, pbmerrors.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get backup %q: %w", name, err)
	}

	b := convertBackup(meta)
	return &b, nil
}

func (s *backupServiceImpl) GetByOpID(ctx context.Context, opid string) (*Backup, error) {
	meta, err := backup.GetBackupByOPID(ctx, s.conn, opid)
	if err != nil {
		if errors.Is(err, pbmerrors.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get backup by opid %q: %w", opid, err)
	}

	b := convertBackup(meta)
	return &b, nil
}

func (s *backupServiceImpl) Start(ctx context.Context, opts StartBackupOptions) (BackupResult, error) {
	cmd := BackupCommand{
		Name:        time.Now().UTC().Format(time.RFC3339),
		Type:        opts.Type,
		ConfigName:  opts.ConfigName,
		Compression: opts.Compression,
		Namespaces:  opts.Namespaces,
		IncrBase:    opts.IncrBase,
	}

	result, err := s.cmds.Send(ctx, cmd)
	if err != nil {
		return BackupResult{}, fmt.Errorf("start backup: %w", err)
	}

	return BackupResult{
		CommandResult: result,
		Name:          cmd.Name,
	}, nil
}

func (s *backupServiceImpl) Cancel(ctx context.Context) (CommandResult, error) {
	result, err := s.cmds.Send(ctx, CancelBackupCommand{})
	if err != nil {
		return CommandResult{}, fmt.Errorf("cancel backup: %w", err)
	}
	return result, nil
}
