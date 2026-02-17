package sdk

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/percona/percona-backup-mongodb/pbm/ctrl"
	"github.com/percona/percona-backup-mongodb/pbm/defs"
	"github.com/percona/percona-backup-mongodb/pbm/lock"
	"github.com/percona/percona-backup-mongodb/pbm/topo"
)

func TestConvertReplicaSet(t *testing.T) {
	s := topo.Shard{
		ID:   "rs0",
		RS:   "rs0",
		Host: "rs0/mongo1:27018,mongo2:27018,mongo3:27018",
	}

	result := convertReplicaSet(s)

	assert.Equal(t, "rs0", result.Name)
	assert.Len(t, result.Nodes, 3)
	assert.Equal(t, "mongo1:27018", result.Nodes[0].Host)
	assert.Equal(t, "mongo2:27018", result.Nodes[1].Host)
	assert.Equal(t, "mongo3:27018", result.Nodes[2].Host)
	// Roles are not available from topology data.
	assert.True(t, result.Nodes[0].Role.IsZero())
}

func TestParseHostNodes(t *testing.T) {
	t.Run("with replset prefix", func(t *testing.T) {
		nodes := parseHostNodes("rs0/host1:27017,host2:27017")
		assert.Len(t, nodes, 2)
		assert.Equal(t, "host1:27017", nodes[0].Host)
		assert.Equal(t, "host2:27017", nodes[1].Host)
	})

	t.Run("without replset prefix", func(t *testing.T) {
		nodes := parseHostNodes("host1:27017")
		assert.Len(t, nodes, 1)
		assert.Equal(t, "host1:27017", nodes[0].Host)
	})

	t.Run("empty string", func(t *testing.T) {
		nodes := parseHostNodes("")
		assert.Nil(t, nodes)
	})

	t.Run("replset prefix only", func(t *testing.T) {
		nodes := parseHostNodes("rs0/")
		assert.Nil(t, nodes)
	})
}

func TestAgentNodeRole(t *testing.T) {
	tests := []struct {
		name     string
		agent    topo.AgentStat
		expected NodeRole
	}{
		{
			name:     "primary",
			agent:    topo.AgentStat{State: defs.NodeStatePrimary},
			expected: RolePrimary,
		},
		{
			name:     "secondary",
			agent:    topo.AgentStat{State: defs.NodeStateSecondary},
			expected: RoleSecondary,
		},
		{
			name:     "arbiter",
			agent:    topo.AgentStat{Arbiter: true, State: defs.NodeStateArbiter},
			expected: RoleArbiter,
		},
		{
			name:     "hidden",
			agent:    topo.AgentStat{Hidden: true, State: defs.NodeStateSecondary},
			expected: RoleHidden,
		},
		{
			name:     "delayed",
			agent:    topo.AgentStat{DelaySecs: 3600, State: defs.NodeStateSecondary},
			expected: RoleDelayed,
		},
		{
			name: "delayed takes precedence over hidden",
			agent: topo.AgentStat{
				DelaySecs: 3600,
				Hidden:    true,
				State:     defs.NodeStateSecondary,
			},
			expected: RoleDelayed,
		},
		{
			name:     "unknown state",
			agent:    topo.AgentStat{State: defs.NodeStateRecovering},
			expected: NodeRole{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agentNodeRole(tt.agent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertAgent(t *testing.T) {
	clusterTime := uint32(1700000100)

	t.Run("healthy agent", func(t *testing.T) {
		a := topo.AgentStat{
			Node:     "mongo1:27018",
			RS:       "rs0",
			AgentVer: "v2.3.0",
			State:    defs.NodeStatePrimary,
			Heartbeat: primitive.Timestamp{
				T: 1700000095, // within StaleFrameSec (30)
			},
			PBMStatus:     topo.SubsysStatus{OK: true},
			NodeStatus:    topo.SubsysStatus{OK: true},
			StorageStatus: topo.SubsysStatus{OK: true},
		}

		result := convertAgent(a, clusterTime)

		assert.Equal(t, "mongo1:27018", result.Node)
		assert.Equal(t, "rs0", result.ReplicaSet)
		assert.Equal(t, "v2.3.0", result.Version)
		assert.Equal(t, RolePrimary, result.Role)
		assert.True(t, result.OK)
		assert.False(t, result.Stale)
		assert.Empty(t, result.Errors)
	})

	t.Run("stale agent", func(t *testing.T) {
		a := topo.AgentStat{
			Node:     "mongo2:27018",
			RS:       "rs0",
			AgentVer: "v2.3.0",
			State:    defs.NodeStateSecondary,
			Heartbeat: primitive.Timestamp{
				T: 1700000050, // 50 seconds old, stale (50 + 30 < 100)
			},
			PBMStatus:     topo.SubsysStatus{OK: true},
			NodeStatus:    topo.SubsysStatus{OK: true},
			StorageStatus: topo.SubsysStatus{OK: true},
		}

		result := convertAgent(a, clusterTime)

		assert.True(t, result.Stale)
	})

	t.Run("agent with errors", func(t *testing.T) {
		a := topo.AgentStat{
			Node:     "mongo3:27018",
			RS:       "rs0",
			AgentVer: "v2.3.0",
			State:    defs.NodeStateSecondary,
			Heartbeat: primitive.Timestamp{
				T: 1700000095,
			},
			PBMStatus:     topo.SubsysStatus{OK: true},
			NodeStatus:    topo.SubsysStatus{OK: true},
			StorageStatus: topo.SubsysStatus{OK: false, Err: "s3 connection timeout"},
			Err:           "agent-level error",
		}

		result := convertAgent(a, clusterTime)

		assert.False(t, result.OK)
		assert.Len(t, result.Errors, 2) // subsystem error + agent error
	})
}

func TestIsStaleAgent(t *testing.T) {
	t.Run("not stale", func(t *testing.T) {
		a := topo.AgentStat{
			Heartbeat: primitive.Timestamp{T: 1700000090},
		}
		assert.False(t, isStaleAgent(a, 1700000100)) // 90 + 30 >= 100
	})

	t.Run("stale", func(t *testing.T) {
		a := topo.AgentStat{
			Heartbeat: primitive.Timestamp{T: 1700000060},
		}
		assert.True(t, isStaleAgent(a, 1700000100)) // 60 + 30 < 100
	})

	t.Run("exactly at boundary", func(t *testing.T) {
		a := topo.AgentStat{
			Heartbeat: primitive.Timestamp{T: 1700000070},
		}
		// 70 + 30 = 100, not strictly less than 100
		assert.False(t, isStaleAgent(a, 1700000100))
	})
}

func TestConvertOperation(t *testing.T) {
	ld := lock.LockData{
		LockHeader: lock.LockHeader{
			Type:    ctrl.CmdBackup,
			Replset: "rs0",
			Node:    "mongo1:27018",
			OPID:    "abc123",
		},
		Heartbeat: primitive.Timestamp{T: 1700000095},
	}

	result := convertOperation(ld)

	assert.Equal(t, CmdTypeBackup, result.Type)
	assert.Equal(t, "abc123", result.OPID)
	assert.Equal(t, "rs0", result.ReplicaSet)
	assert.Equal(t, "mongo1:27018", result.Node)
}

func TestConvertOperationUnknownType(t *testing.T) {
	ld := lock.LockData{
		LockHeader: lock.LockHeader{
			Type: ctrl.Command("unknownCmd"),
			OPID: "xyz",
		},
	}

	result := convertOperation(ld)

	assert.True(t, result.Type.IsZero())
	assert.Equal(t, "xyz", result.OPID)
}

func TestConvertAgentSubsystemErrors(t *testing.T) {
	// Test that errors from AgentStat.OK() are properly collected.
	a := topo.AgentStat{
		Node:          "mongo1:27018",
		RS:            "rs0",
		State:         defs.NodeStatePrimary,
		Heartbeat:     primitive.Timestamp{T: 100},
		PBMStatus:     topo.SubsysStatus{OK: false, Err: "pbm error"},
		NodeStatus:    topo.SubsysStatus{OK: false, Err: "node error"},
		StorageStatus: topo.SubsysStatus{OK: false, Err: "storage error"},
	}

	result := convertAgent(a, 100)

	assert.False(t, result.OK)
	// The exact number and format of errors depends on AgentStat.OK() implementation.
	// We just verify that errors are collected.
	assert.NotEmpty(t, result.Errors)
	fmt.Println("errors:", result.Errors) // for debugging in test output
}
