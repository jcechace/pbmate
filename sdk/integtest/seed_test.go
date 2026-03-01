//go:build integration

package integtest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/percona/percona-backup-mongodb/pbm/backup"
	"github.com/percona/percona-backup-mongodb/pbm/config"
	"github.com/percona/percona-backup-mongodb/pbm/lock"
	pbmlog "github.com/percona/percona-backup-mongodb/pbm/log"
	"github.com/percona/percona-backup-mongodb/pbm/oplog"
	"github.com/percona/percona-backup-mongodb/pbm/restore"
	"github.com/percona/percona-backup-mongodb/pbm/topo"
)

// seedBackup inserts a BackupMeta document into the pbmBackups collection.
func (h *testHarness) seedBackup(t *testing.T, meta backup.BackupMeta) {
	t.Helper()
	_, err := h.collection("pbmBackups").InsertOne(context.Background(), meta)
	require.NoError(t, err, "seed backup %q", meta.Name)
}

// seedRestore inserts a RestoreMeta document into the pbmRestores collection.
func (h *testHarness) seedRestore(t *testing.T, meta restore.RestoreMeta) {
	t.Helper()
	_, err := h.collection("pbmRestores").InsertOne(context.Background(), meta)
	require.NoError(t, err, "seed restore %q", meta.Name)
}

// seedConfig inserts a Config document into the pbmConfig collection.
// Use this for both main config (Name="") and profiles (Name="profile-name", IsProfile=true).
func (h *testHarness) seedConfig(t *testing.T, cfg config.Config) {
	t.Helper()
	_, err := h.collection("pbmConfig").InsertOne(context.Background(), cfg)
	require.NoError(t, err, "seed config %q", cfg.Name)
}

// seedAgent inserts an AgentStat document into the pbmAgents collection.
func (h *testHarness) seedAgent(t *testing.T, stat topo.AgentStat) {
	t.Helper()
	_, err := h.collection("pbmAgents").InsertOne(context.Background(), stat)
	require.NoError(t, err, "seed agent %q", stat.Node)
}

// seedLock inserts a LockData document into the pbmLock collection.
func (h *testHarness) seedLock(t *testing.T, data lock.LockData) {
	t.Helper()
	_, err := h.collection("pbmLock").InsertOne(context.Background(), data)
	require.NoError(t, err, "seed lock")
}

// seedLockOp inserts a LockData document into the pbmLockOp collection.
// Use this for non-mutually-exclusive operation locks (e.g. backup-delete).
func (h *testHarness) seedLockOp(t *testing.T, data lock.LockData) {
	t.Helper()
	_, err := h.collection("pbmLockOp").InsertOne(context.Background(), data)
	require.NoError(t, err, "seed lock op")
}

// seedPITRChunk inserts an OplogChunk document into the pbmPITRChunks collection.
func (h *testHarness) seedPITRChunk(t *testing.T, chunk oplog.OplogChunk) {
	t.Helper()
	_, err := h.collection("pbmPITRChunks").InsertOne(context.Background(), chunk)
	require.NoError(t, err, "seed pitr chunk %q", chunk.FName)
}

// seedLog inserts a log Entry document into the pbmLog collection.
func (h *testHarness) seedLog(t *testing.T, entry pbmlog.Entry) {
	t.Helper()
	_, err := h.collection("pbmLog").InsertOne(context.Background(), entry)
	require.NoError(t, err, "seed log entry")
}
