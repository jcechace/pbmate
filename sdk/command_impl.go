package sdk

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/percona/percona-backup-mongodb/pbm/connect"
	"github.com/percona/percona-backup-mongodb/pbm/ctrl"
	"github.com/percona/percona-backup-mongodb/pbm/defs"
	"github.com/percona/percona-backup-mongodb/pbm/lock"
	"github.com/percona/percona-backup-mongodb/pbm/topo"
)

type commandServiceImpl struct {
	conn connect.Client
	log  *slog.Logger
}

var _ CommandService = (*commandServiceImpl)(nil)

func (s *commandServiceImpl) Send(ctx context.Context, cmd Command) (CommandResult, error) {
	if err := s.CheckLock(ctx); err != nil {
		return CommandResult{}, err
	}

	pbmCmd, err := convertCommandToPBM(cmd)
	if err != nil {
		return CommandResult{}, fmt.Errorf("convert command: %w", err)
	}

	s.log.InfoContext(ctx, "dispatching command", "kind", cmd.kind())
	s.log.DebugContext(ctx, "command details", "command", cmd)
	opid, err := s.dispatch(ctx, pbmCmd)
	if err != nil {
		return CommandResult{}, fmt.Errorf("dispatch command: %w", err)
	}

	s.log.InfoContext(ctx, "command dispatched", "kind", cmd.kind(), "opid", opid)
	return CommandResult{OPID: opid}, nil
}

func (s *commandServiceImpl) CheckLock(ctx context.Context) error {
	s.log.DebugContext(ctx, "checking for concurrent operations")
	locks, err := lock.GetLocks(ctx, s.conn, &lock.LockHeader{})
	if err != nil {
		return fmt.Errorf("check running operations: %w", err)
	}

	if len(locks) == 0 {
		return nil
	}

	clusterTime, err := topo.GetClusterTime(ctx, s.conn)
	if err != nil {
		return fmt.Errorf("get cluster time: %w", err)
	}

	for _, l := range locks {
		if l.Heartbeat.T+defs.StaleFrameSec >= clusterTime.T {
			cmdType, _ := ParseCommandType(string(l.Type))
			return &ConcurrentOperationError{
				Type: cmdType,
				OPID: l.OPID,
			}
		}
	}

	return nil
}

// dispatch inserts a command into the PBM command stream collection.
// This replicates PBM's internal sendCommand pattern because that function
// is unexported and PBM only exports type-specific wrappers (SendCancelBackup,
// SendDeleteBackupByName, etc.) — none for generic backup/restore dispatch.
//
// NOTE: Direct MongoDB interaction should only be used when no reasonable
// exported PBM API exists. This is currently the only such exception.
func (s *commandServiceImpl) dispatch(ctx context.Context, cmd ctrl.Cmd) (string, error) {
	cmd.TS = time.Now().UTC().Unix()

	res, err := s.conn.CmdStreamCollection().InsertOne(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("insert command: %w", err)
	}

	opid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("unexpected opid type: %T", res.InsertedID)
	}

	return opid.Hex(), nil
}
