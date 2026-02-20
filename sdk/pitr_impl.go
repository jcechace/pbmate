package sdk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/percona/percona-backup-mongodb/pbm/config"
	"github.com/percona/percona-backup-mongodb/pbm/connect"
	pbmerrors "github.com/percona/percona-backup-mongodb/pbm/errors"
	"github.com/percona/percona-backup-mongodb/pbm/oplog"
)

type pitrServiceImpl struct {
	conn connect.Client
	log  *slog.Logger
}

var _ PITRService = (*pitrServiceImpl)(nil)

func (s *pitrServiceImpl) Status(ctx context.Context) (*PITRStatus, error) {
	// Check if PITR is enabled in config.
	enabled, _, err := config.IsPITREnabled(ctx, s.conn)
	if err != nil {
		return nil, fmt.Errorf("get pitr status: check config: %w", err)
	}

	status := &PITRStatus{
		Enabled: enabled,
	}

	if !enabled {
		return status, nil
	}

	// Check if oplog slicing is actively running (non-stale PITR locks).
	running, err := oplog.IsOplogSlicing(ctx, s.conn)
	if err != nil {
		return nil, fmt.Errorf("get pitr status: check slicing: %w", err)
	}
	status.Running = running

	// Get the list of nodes actively slicing.
	if running {
		nodes, err := oplog.FetchSlicersWithActiveLocks(ctx, s.conn)
		if err != nil {
			return nil, fmt.Errorf("get pitr status: fetch slicers: %w", err)
		}
		status.Nodes = nodes
	}

	// Collect any per-replset errors from PITR meta.
	meta, err := oplog.GetMeta(ctx, s.conn)
	if err != nil {
		if !errors.Is(err, pbmerrors.ErrNotFound) {
			return nil, fmt.Errorf("get pitr status: get meta: %w", err)
		}
		// No meta document — not an error, just no replset status to report.
	} else {
		status.Error = collectPITRErrors(meta.Replsets)
	}

	return status, nil
}

func (s *pitrServiceImpl) Timelines(ctx context.Context) ([]Timeline, error) {
	tlns, err := oplog.PITRTimelines(ctx, s.conn)
	if err != nil {
		return nil, fmt.Errorf("get pitr timelines: %w", err)
	}

	return convertTimelines(tlns), nil
}
