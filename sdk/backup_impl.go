package sdk

import (
	"context"
	"fmt"

	"github.com/percona/percona-backup-mongodb/pbm/connect"
)

type backupServiceImpl struct {
	conn connect.Client
}

var _ BackupService = (*backupServiceImpl)(nil)

func (s *backupServiceImpl) List(ctx context.Context, opts ListBackupsOptions) ([]Backup, error) {
	return nil, fmt.Errorf("backup list: not implemented")
}

func (s *backupServiceImpl) Get(ctx context.Context, name string) (*Backup, error) {
	return nil, fmt.Errorf("backup get: not implemented")
}

func (s *backupServiceImpl) GetByOpID(ctx context.Context, opid string) (*Backup, error) {
	return nil, fmt.Errorf("backup get by opid: not implemented")
}
