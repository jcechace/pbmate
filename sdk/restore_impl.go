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
	conn    connect.Client
	cmds    *commandServiceImpl
	backups *backupServiceImpl
	log     *slog.Logger
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
		return nil, fmt.Errorf("start restore: %w", err)
	}

	// Extract the backup name from the command to determine result type.
	backupName := restoreCommandBackupName(cmd)

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
		return nil, fmt.Errorf("start restore: %w", err)
	}

	return s.newRestoreResult(ctx, name, result.OPID, backupName), nil
}

// newRestoreResult creates the appropriate RestoreResult implementation based
// on the backup type. Physical and incremental backups produce an unwaitable
// result because mongod shuts down during the restore. If the backup type
// cannot be determined, falls back to a waitable result (MongoDB polling).
func (s *restoreServiceImpl) newRestoreResult(ctx context.Context, name, opid, backupName string) RestoreResult {
	bk, err := s.backups.Get(ctx, backupName)
	if err != nil {
		// Backup lookup failed — fall back to waitable (MongoDB polling).
		// This shouldn't happen since we just validated the command, but
		// if it does, waitable is the safer default.
		s.log.WarnContext(ctx, "failed to look up backup type for restore result, defaulting to waitable",
			"backup", backupName, "error", err)
		return &waitableRestoreResult{name: name, opid: opid, svc: s}
	}

	if bk.IsPhysical() || bk.IsIncremental() {
		return &unwaitableRestoreResult{name: name, opid: opid}
	}

	return &waitableRestoreResult{name: name, opid: opid, svc: s}
}

// restoreCommandBackupName extracts the backup name from a StartRestoreCommand.
func restoreCommandBackupName(cmd StartRestoreCommand) string {
	switch c := cmd.(type) {
	case StartSnapshotRestore:
		return c.BackupName
	case StartPITRRestore:
		return c.BackupName
	default:
		panic(fmt.Sprintf("unreachable: unknown StartRestoreCommand type %T", cmd))
	}
}

// --- RestoreResult implementations ---

var _ RestoreResult = (*waitableRestoreResult)(nil)
var _ RestoreResult = (*unwaitableRestoreResult)(nil)

// waitableRestoreResult polls MongoDB for restore status. Used for logical
// restores where mongod stays up throughout the operation.
type waitableRestoreResult struct {
	name string
	opid string
	svc  *restoreServiceImpl
}

func (r *waitableRestoreResult) Name() string   { return r.name }
func (r *waitableRestoreResult) OPID() string   { return r.opid }
func (r *waitableRestoreResult) Waitable() bool { return true }

func (r *waitableRestoreResult) Wait(ctx context.Context, opts RestoreWaitOptions) (*Restore, error) {
	return waitForTerminal(ctx, r.name, opts.PollInterval, waitParams[*Restore]{
		get:        r.svc.Get,
		status:     func(rs *Restore) Status { return rs.Status },
		errMsg:     func(rs *Restore) string { return rs.Error },
		onProgress: opts.OnProgress,
		log:        r.svc.log,
		entity:     "restore",
	})
}

// unwaitableRestoreResult is returned for restores based on physical or
// incremental backups. These restores shut down mongod, making MongoDB-based
// polling impossible. Wait always returns ErrRestoreUnwaitable.
type unwaitableRestoreResult struct {
	name string
	opid string
}

func (r *unwaitableRestoreResult) Name() string   { return r.name }
func (r *unwaitableRestoreResult) OPID() string   { return r.opid }
func (r *unwaitableRestoreResult) Waitable() bool { return false }

func (r *unwaitableRestoreResult) Wait(_ context.Context, _ RestoreWaitOptions) (*Restore, error) {
	return nil, ErrRestoreUnwaitable
}
