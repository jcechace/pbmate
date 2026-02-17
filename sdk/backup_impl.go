package sdk

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	mongoopts "go.mongodb.org/mongo-driver/mongo/options"

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
	limit := int64(opts.Limit)

	filter := bson.D{}
	if !opts.ConfigName.IsZero() {
		filter = append(filter, bson.E{Key: "store.name", Value: configNameToPBM(opts.ConfigName)})
	}
	if !opts.Type.IsZero() {
		filter = append(filter, bson.E{Key: "type", Value: opts.Type.String()})
	}

	findOpts := mongoopts.Find().SetSort(bson.D{{Key: "start_ts", Value: -1}})
	if limit > 0 {
		findOpts.SetLimit(limit)
	}

	cur, err := s.conn.BcpCollection().Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("list backups: %w", err)
	}
	defer func() { _ = cur.Close(ctx) }()

	var metas []backup.BackupMeta
	if err := cur.All(ctx, &metas); err != nil {
		return nil, fmt.Errorf("decode backups: %w", err)
	}

	result := make([]Backup, len(metas))
	for i := range metas {
		result[i] = convertBackup(&metas[i])
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
