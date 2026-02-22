package sdk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/percona/percona-backup-mongodb/pbm/backup"
	"github.com/percona/percona-backup-mongodb/pbm/connect"
	"github.com/percona/percona-backup-mongodb/pbm/defs"
	pbmerrors "github.com/percona/percona-backup-mongodb/pbm/errors"
)

type backupServiceImpl struct {
	conn connect.Client
	cmds CommandService
	log  *slog.Logger
}

var _ BackupService = (*backupServiceImpl)(nil)

func (s *backupServiceImpl) List(ctx context.Context, opts ListBackupsOptions) ([]Backup, error) {
	// TODO(pbm-fix): PBM's BackupsList does not support server-side
	// filtering by config name or backup type. Fetch all and filter in
	// memory — backup counts are small enough that this is always practical.
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

func (s *backupServiceImpl) Start(ctx context.Context, cmd StartBackupCommand) (BackupResult, error) {
	// PBM uses RFC 3339 (second precision) for backup names.
	name := time.Now().UTC().Format(time.RFC3339)

	// Inject the auto-generated name into the concrete command type.
	switch c := cmd.(type) {
	case StartLogicalBackup:
		c.name = name
		cmd = c
	case StartIncrementalBackup:
		c.name = name
		cmd = c
	}

	s.log.InfoContext(ctx, "starting backup", "name", name, "type", fmt.Sprintf("%T", cmd))
	result, err := s.cmds.Send(ctx, cmd)
	if err != nil {
		return BackupResult{}, fmt.Errorf("start backup: %w", err)
	}

	return BackupResult{
		CommandResult: result,
		Name:          name,
	}, nil
}

func (s *backupServiceImpl) Wait(ctx context.Context, name string, opts BackupWaitOptions) (*Backup, error) {
	return waitForTerminal(ctx, name, opts.PollInterval, waitParams[*Backup]{
		get:        s.Get,
		status:     func(b *Backup) Status { return b.Status },
		errMsg:     func(b *Backup) string { return b.Error },
		onProgress: opts.OnProgress,
		log:        s.log,
		entity:     "backup",
	})
}

func (s *backupServiceImpl) Delete(ctx context.Context, cmd DeleteBackupCommand) (CommandResult, error) {
	switch c := cmd.(type) {
	case DeleteBackupByName:
		return s.deleteByName(ctx, c)
	case DeleteBackupsBefore:
		return s.deleteBefore(ctx, c)
	default:
		return CommandResult{}, fmt.Errorf("unsupported delete command type: %T", cmd)
	}
}

func (s *backupServiceImpl) deleteByName(ctx context.Context, cmd DeleteBackupByName) (CommandResult, error) {
	if cmd.Name == "" {
		return CommandResult{}, fmt.Errorf("delete backup: name is required")
	}

	s.log.InfoContext(ctx, "deleting backup", "name", cmd.Name)
	result, err := s.cmds.Send(ctx, cmd)
	if err != nil {
		return CommandResult{}, fmt.Errorf("delete backup %q: %w", cmd.Name, err)
	}
	return result, nil
}

func (s *backupServiceImpl) deleteBefore(ctx context.Context, cmd DeleteBackupsBefore) (CommandResult, error) {
	if cmd.OlderThan.IsZero() {
		return CommandResult{}, fmt.Errorf("delete backups: older-than time must be set")
	}
	if cmd.OlderThan.After(time.Now().UTC()) {
		return CommandResult{}, fmt.Errorf("delete backups: older-than time %s is in the future",
			cmd.OlderThan.Format(time.RFC3339))
	}

	s.log.InfoContext(ctx, "deleting backups older than",
		"olderThan", cmd.OlderThan.Format(time.RFC3339),
		"type", cmd.Type,
		"configName", cmd.ConfigName,
	)
	result, err := s.cmds.Send(ctx, cmd)
	if err != nil {
		return CommandResult{}, fmt.Errorf("delete backups older than %s: %w",
			cmd.OlderThan.Format(time.RFC3339), err)
	}
	return result, nil
}

func (s *backupServiceImpl) Cancel(ctx context.Context) (CommandResult, error) {
	result, err := s.cmds.Send(ctx, CancelBackupCommand{})
	if err != nil {
		return CommandResult{}, fmt.Errorf("cancel backup: %w", err)
	}
	return result, nil
}

func (s *backupServiceImpl) CanDelete(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("can delete: name is required")
	}

	mgr := backup.NewDBManager(s.conn)
	bcp, err := mgr.GetBackupByName(ctx, name)
	if err != nil {
		if errors.Is(err, pbmerrors.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("can delete %q: %w", name, err)
	}

	if bcp.Type != defs.IncrementalBackup {
		return translateCanDeleteError(backup.CanDeleteBackup(ctx, s.conn, bcp))
	}

	// Incremental backup: walk up to the chain base so the PITR anchor
	// check covers the entire chain's time span.
	base := bcp
	for base.SrcBackup != "" {
		base, err = mgr.GetBackupByName(ctx, base.SrcBackup)
		if err != nil {
			return fmt.Errorf("can delete %q: resolve chain base: %w", name, err)
		}
	}

	increments, err := backup.FetchAllIncrements(ctx, s.conn, base)
	if err != nil {
		return fmt.Errorf("can delete %q: fetch chain: %w", name, err)
	}

	return translateCanDeleteError(backup.CanDeleteIncrementalChain(ctx, s.conn, base, increments))
}

// translateCanDeleteError converts PBM delete-check sentinel errors to
// SDK-owned equivalents. Errors that cannot occur due to internal routing
// (ErrIncrementalBackup, ErrNonIncrementalBackup, ErrNotBaseIncrement) are
// passed through with context wrapping.
func translateCanDeleteError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, backup.ErrBackupInProgress):
		return ErrBackupInProgress
	case errors.Is(err, backup.ErrBaseForPITR):
		return ErrDeleteProtectedByPITR
	default:
		return fmt.Errorf("can delete: %w", err)
	}
}
