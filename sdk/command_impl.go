package sdk

import (
	"context"
	"fmt"

	"github.com/percona/percona-backup-mongodb/pbm/connect"
)

type commandServiceImpl struct {
	conn connect.Client
}

var _ CommandService = (*commandServiceImpl)(nil)

func (s *commandServiceImpl) Send(ctx context.Context, cmd Command) (CommandResult, error) {
	return CommandResult{}, fmt.Errorf("command send: not implemented")
}
