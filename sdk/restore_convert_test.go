package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/percona/percona-backup-mongodb/pbm/defs"
	"github.com/percona/percona-backup-mongodb/pbm/restore"
)

func TestConvertRestore(t *testing.T) {
	meta := &restore.RestoreMeta{
		Name:             "restore-2024",
		OPID:             "opid123",
		Backup:           "2024-01-15T10:30:00Z",
		BcpChain:         []string{"base", "incr1"},
		Type:             defs.LogicalBackup,
		Status:           defs.StatusDone,
		StartTS:          1705312200,
		PITR:             1705312300,
		Namespaces:       []string{"db1.coll1"},
		LastTransitionTS: 1705312400,
		Error:            "",
		Replsets: []restore.RestoreReplset{
			{
				Name:             "rs0",
				Status:           defs.StatusDone,
				LastTransitionTS: 1705312400,
				Nodes: []restore.RestoreNode{
					{
						Name:             "rs00:27017",
						Status:           defs.StatusDone,
						LastTransitionTS: 1705312400,
					},
				},
			},
		},
	}

	r := convertRestore(meta)

	assert.Equal(t, "restore-2024", r.Name)
	assert.Equal(t, "opid123", r.OPID)
	assert.Equal(t, "2024-01-15T10:30:00Z", r.Backup)
	assert.Equal(t, []string{"base", "incr1"}, r.BcpChain)
	assert.Equal(t, BackupLogical, r.Type)
	assert.Equal(t, StatusDone, r.Status)
	assert.Equal(t, int64(1705312200), r.StartTS.Unix())
	assert.Equal(t, uint32(1705312300), r.PITRTarget.T)
	assert.Equal(t, uint32(0), r.PITRTarget.I)
	assert.Equal(t, []string{"db1.coll1"}, r.Namespaces)
	assert.Equal(t, int64(1705312400), r.LastTransitionTS.Unix())
	assert.Empty(t, r.Error)

	// FinishTS should be set for terminal status
	assert.Equal(t, r.LastTransitionTS, r.FinishTS)

	// Replsets
	assert.Len(t, r.Replsets, 1)
	rs := r.Replsets[0]
	assert.Equal(t, "rs0", rs.Name)
	assert.Equal(t, StatusDone, rs.Status)

	// Nodes
	assert.Len(t, rs.Nodes, 1)
	assert.Equal(t, "rs00:27017", rs.Nodes[0].Name)
	assert.Equal(t, StatusDone, rs.Nodes[0].Status)
}

func TestConvertRestoreNoPITR(t *testing.T) {
	meta := &restore.RestoreMeta{
		Name:   "test",
		Status: defs.StatusDone,
		PITR:   0,
	}

	r := convertRestore(meta)
	assert.True(t, r.PITRTarget.IsZero())
}

func TestConvertRestoreRunningNoFinishTS(t *testing.T) {
	meta := &restore.RestoreMeta{
		Name:             "test",
		Status:           defs.StatusRunning,
		LastTransitionTS: 1705312400,
	}

	r := convertRestore(meta)
	assert.True(t, r.FinishTS.IsZero(), "FinishTS should be zero for non-terminal status")
}

func TestConvertRestoreReplsetsNil(t *testing.T) {
	meta := &restore.RestoreMeta{Name: "test"}
	r := convertRestore(meta)
	assert.Nil(t, r.Replsets)
}

func TestConvertRestoreNodesNil(t *testing.T) {
	rs := &restore.RestoreReplset{Name: "rs0"}
	result := convertRestoreReplset(rs)
	assert.Nil(t, result.Nodes)
}

func TestIsTerminalStatus(t *testing.T) {
	assert.True(t, isTerminalStatus(defs.StatusDone))
	assert.True(t, isTerminalStatus(defs.StatusError))
	assert.True(t, isTerminalStatus(defs.StatusCancelled))
	assert.True(t, isTerminalStatus(defs.StatusPartlyDone))
	assert.False(t, isTerminalStatus(defs.StatusRunning))
	assert.False(t, isTerminalStatus(defs.StatusInit))
	assert.False(t, isTerminalStatus(defs.StatusStarting))
}
