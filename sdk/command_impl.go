package sdk

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/percona/percona-backup-mongodb/pbm/connect"
	"github.com/percona/percona-backup-mongodb/pbm/ctrl"
)

type commandServiceImpl struct {
	conn connect.Client
	log  *slog.Logger
}

// validateAndCheckLock validates the command and checks for concurrent PBM
// operations. Returns nil when the command is valid and no non-stale lock
// exists.
func (s *commandServiceImpl) validateAndCheckLock(ctx context.Context, cmd validator) error {
	if err := cmd.Validate(); err != nil {
		return err
	}
	return s.checkLock(ctx)
}

func (s *commandServiceImpl) checkLock(ctx context.Context) error {
	return checkLock(ctx, s.conn, s.log)
}

// TODO(pbm-fix): PBM's internal sendCommand is unexported and only
// type-specific wrappers are exported (SendCancelBackup, etc.) — none
// for generic backup/restore dispatch. This replicates sendCommand via
// direct collection insert. Remove when PBM exports a generic dispatch API.
func (s *commandServiceImpl) dispatch(ctx context.Context, cmd ctrl.Cmd) (CommandResult, error) {
	cmd.TS = time.Now().UTC().Unix()

	res, err := s.conn.CmdStreamCollection().InsertOne(ctx, cmd)
	if err != nil {
		return CommandResult{}, fmt.Errorf("insert command: %w", err)
	}

	opid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return CommandResult{}, fmt.Errorf("unexpected opid type: %T", res.InsertedID)
	}

	s.log.InfoContext(ctx, "command dispatched", "opid", opid.Hex())
	return CommandResult{OPID: opid.Hex()}, nil
}
