//go:build integration

package integtest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/percona-backup-mongodb/pbm/ctrl"
	"github.com/percona/percona-backup-mongodb/pbm/defs"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

func TestClusterMembers(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// The testcontainers RS should have at least one replica set.
	members, err := h.client.Cluster.Members(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, members)

	// Single-node RS named "rs" (from testcontainers setup).
	rs := members[0]
	assert.Equal(t, "rs", rs.Name)
	require.NotEmpty(t, rs.Nodes)
	assert.NotEmpty(t, rs.Nodes[0].Host)
}

func TestClusterAgents(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedAgent(t, newAgentStat("rs0:27017", "rs",
		withAgentState(defs.NodeStatePrimary, "PRIMARY"),
	))
	h.seedAgent(t, newAgentStat("rs0:27018", "rs",
		withAgentState(defs.NodeStateSecondary, "SECONDARY"),
	))

	agents, err := h.client.Cluster.Agents(ctx)
	require.NoError(t, err)
	require.Len(t, agents, 2)

	// Find the primary agent.
	var primary *sdk.Agent
	for i := range agents {
		if agents[i].Node == "rs0:27017" {
			primary = &agents[i]
			break
		}
	}
	require.NotNil(t, primary, "should find primary agent")

	assert.Equal(t, "rs", primary.ReplicaSet)
	assert.Equal(t, "2.13.0", primary.Version)
	assert.True(t, primary.Role.Equal(sdk.NodeRolePrimary))
	assert.True(t, primary.OK)
	assert.False(t, primary.Stale)
	assert.Empty(t, primary.Errors)
}

func TestClusterAgentsEmpty(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	agents, err := h.client.Cluster.Agents(ctx)
	require.NoError(t, err)
	assert.Empty(t, agents)
}

func TestClusterAgentsStale(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// PBM's ListAgentStatuses calls AgentStatusGC first, which deletes agents
	// with heartbeat older than clusterTime - 35s. The SDK marks agents as
	// stale at clusterTime - 30s (StaleFrameSec). So the window between
	// "stale" and "GC'd" is only 5 seconds. Set heartbeat to clusterTime - 32
	// to be stale but not garbage-collected.
	ct, err := h.client.Cluster.ClusterTime(ctx)
	require.NoError(t, err)

	h.seedAgent(t, newAgentStat("stale-node:27017", "rs",
		withAgentHeartbeat(ct.T-32),
	))

	agents, err := h.client.Cluster.Agents(ctx)
	require.NoError(t, err)
	require.Len(t, agents, 1)

	assert.Equal(t, "stale-node:27017", agents[0].Node)
	assert.True(t, agents[0].Stale)
}

func TestClusterAgentsWithError(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedAgent(t, newAgentStat("error-node:27017", "rs",
		withAgentError("storage unreachable"),
	))

	agents, err := h.client.Cluster.Agents(ctx)
	require.NoError(t, err)
	require.Len(t, agents, 1)

	assert.False(t, agents[0].OK)
	assert.Contains(t, agents[0].Errors, "storage unreachable")
}

func TestClusterRunningOperations(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed a non-stale lock (fresh heartbeat).
	h.seedLock(t, newLockData(ctrl.CmdBackup, "rs", "rs0:27017"))

	ops, err := h.client.Cluster.RunningOperations(ctx)
	require.NoError(t, err)
	require.Len(t, ops, 1)

	assert.True(t, ops[0].Type.Equal(sdk.CmdTypeBackup))
	assert.Equal(t, "rs", ops[0].ReplicaSet)
	assert.Equal(t, "rs0:27017", ops[0].Node)
	assert.NotEmpty(t, ops[0].OPID)
}

func TestClusterRunningOperationsFiltersStale(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed a lock with a heartbeat from the distant past — should be filtered out.
	h.seedLock(t, newLockData(ctrl.CmdBackup, "rs", "rs0:27017",
		withLockHeartbeat(1), // epoch — definitely stale
	))

	ops, err := h.client.Cluster.RunningOperations(ctx)
	require.NoError(t, err)
	assert.Empty(t, ops, "stale lock should be filtered out")
}

func TestClusterRunningOperationsEmpty(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	ops, err := h.client.Cluster.RunningOperations(ctx)
	require.NoError(t, err)
	assert.Empty(t, ops)
}

func TestClusterCheckLockClear(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// No locks — should return nil.
	err := h.client.Cluster.CheckLock(ctx)
	assert.NoError(t, err)
}

func TestClusterCheckLockBlocked(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed a non-stale lock.
	h.seedLock(t, newLockData(ctrl.CmdRestore, "rs", "rs0:27017"))

	err := h.client.Cluster.CheckLock(ctx)
	require.Error(t, err)

	var concErr *sdk.ConcurrentOperationError
	assert.ErrorAs(t, err, &concErr)
}

func TestClusterClusterTime(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	ts, err := h.client.Cluster.ClusterTime(ctx)
	require.NoError(t, err)

	// Cluster time should be non-zero and recent.
	assert.NotZero(t, ts.T, "cluster time T should be non-zero")
}

func TestClusterServerInfo(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	info, err := h.client.Cluster.ServerInfo(ctx)
	require.NoError(t, err)

	// Mongo version should start with "8.0" (matching the container image).
	assert.True(t, len(info.MongoVersion) > 0)
	assert.Contains(t, info.MongoVersion, "8.0")

	// PBM version should be non-empty.
	assert.NotEmpty(t, info.PBMVersion)
}
