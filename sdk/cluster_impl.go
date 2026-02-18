package sdk

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/percona/percona-backup-mongodb/pbm/connect"
	"github.com/percona/percona-backup-mongodb/pbm/defs"
	"github.com/percona/percona-backup-mongodb/pbm/lock"
	"github.com/percona/percona-backup-mongodb/pbm/topo"
)

type clusterServiceImpl struct {
	conn connect.Client
	log  *slog.Logger
}

var _ ClusterService = (*clusterServiceImpl)(nil)

func (s *clusterServiceImpl) Members(ctx context.Context) ([]ReplicaSet, error) {
	shards, err := topo.ClusterMembers(ctx, s.conn.MongoClient())
	if err != nil {
		return nil, fmt.Errorf("cluster members: %w", err)
	}

	result := make([]ReplicaSet, len(shards))
	for i := range shards {
		result[i] = convertReplicaSet(shards[i])
	}
	return result, nil
}

func (s *clusterServiceImpl) Agents(ctx context.Context) ([]Agent, error) {
	agents, err := topo.ListAgentStatuses(ctx, s.conn)
	if err != nil {
		return nil, fmt.Errorf("cluster agents: %w", err)
	}

	ct, err := topo.GetClusterTime(ctx, s.conn)
	if err != nil {
		return nil, fmt.Errorf("cluster agents: get cluster time: %w", err)
	}

	result := make([]Agent, len(agents))
	for i := range agents {
		result[i] = convertAgent(agents[i], ct.T)
	}
	return result, nil
}

func (s *clusterServiceImpl) RunningOperations(ctx context.Context) ([]Operation, error) {
	locks, err := lock.GetLocks(ctx, s.conn, &lock.LockHeader{})
	if err != nil {
		return nil, fmt.Errorf("cluster running operations: %w", err)
	}

	ct, err := topo.GetClusterTime(ctx, s.conn)
	if err != nil {
		return nil, fmt.Errorf("cluster running operations: get cluster time: %w", err)
	}

	// Filter out stale locks.
	var result []Operation
	for i := range locks {
		if locks[i].Heartbeat.T+defs.StaleFrameSec >= ct.T {
			result = append(result, convertOperation(locks[i]))
		}
	}
	return result, nil
}

func (s *clusterServiceImpl) ClusterTime(ctx context.Context) (Timestamp, error) {
	ct, err := topo.GetClusterTime(ctx, s.conn)
	if err != nil {
		return Timestamp{}, fmt.Errorf("cluster time: %w", err)
	}

	return convertTimestamp(ct), nil
}
