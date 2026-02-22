package sdk

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/percona/percona-backup-mongodb/pbm/connect"
	"github.com/percona/percona-backup-mongodb/pbm/defs"
	"github.com/percona/percona-backup-mongodb/pbm/lock"
	"github.com/percona/percona-backup-mongodb/pbm/topo"
	"github.com/percona/percona-backup-mongodb/pbm/version"
)

type clusterServiceImpl struct {
	conn connect.Client
	log  *slog.Logger
}

var _ ClusterService = (*clusterServiceImpl)(nil)

func (s *clusterServiceImpl) Members(ctx context.Context) ([]ReplicaSet, error) {
	shards, err := topo.ClusterMembers(ctx, s.conn.MongoClient())
	if err != nil {
		return nil, fmt.Errorf("list cluster members: %w", err)
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
		return nil, fmt.Errorf("list agents: %w", err)
	}

	ct, err := topo.GetClusterTime(ctx, s.conn)
	if err != nil {
		return nil, fmt.Errorf("list agents: get cluster time: %w", err)
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
		return nil, fmt.Errorf("list running operations: %w", err)
	}

	ct, err := topo.GetClusterTime(ctx, s.conn)
	if err != nil {
		return nil, fmt.Errorf("list running operations: get cluster time: %w", err)
	}

	// TODO(pbm-fix): GetLocks returns all locks including stale ones.
	// Filter out stale locks client-side until PBM provides an active-only API.
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
		return Timestamp{}, fmt.Errorf("get cluster time: %w", err)
	}

	return convertTimestamp(ct), nil
}

func (s *clusterServiceImpl) ServerInfo(ctx context.Context) (*ServerInfo, error) {
	mongoVer, err := version.GetMongoVersion(ctx, s.conn.MongoClient())
	if err != nil {
		return nil, fmt.Errorf("get server info: mongo version: %w", err)
	}

	return &ServerInfo{
		MongoVersion: mongoVer.VersionString,
		PBMVersion:   version.Current().Version,
	}, nil
}
