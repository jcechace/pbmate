package sdk

import (
	"context"
	"fmt"

	"github.com/percona/percona-backup-mongodb/pbm/connect"
)

type clusterServiceImpl struct {
	conn connect.Client
}

var _ ClusterService = (*clusterServiceImpl)(nil)

func (s *clusterServiceImpl) Members(ctx context.Context) ([]ReplicaSet, error) {
	return nil, fmt.Errorf("cluster members: not implemented")
}

func (s *clusterServiceImpl) Agents(ctx context.Context) ([]Agent, error) {
	return nil, fmt.Errorf("cluster agents: not implemented")
}

func (s *clusterServiceImpl) RunningOperations(ctx context.Context) ([]Operation, error) {
	return nil, fmt.Errorf("cluster running operations: not implemented")
}

func (s *clusterServiceImpl) ClusterTime(ctx context.Context) (Timestamp, error) {
	return Timestamp{}, fmt.Errorf("cluster time: not implemented")
}
