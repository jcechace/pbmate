//go:build integration

package integtest

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/percona/percona-backup-mongodb/pbm/backup"
	"github.com/percona/percona-backup-mongodb/pbm/compress"
	"github.com/percona/percona-backup-mongodb/pbm/config"
	"github.com/percona/percona-backup-mongodb/pbm/ctrl"
	"github.com/percona/percona-backup-mongodb/pbm/defs"
	"github.com/percona/percona-backup-mongodb/pbm/lock"
	pbmlog "github.com/percona/percona-backup-mongodb/pbm/log"
	"github.com/percona/percona-backup-mongodb/pbm/oplog"
	"github.com/percona/percona-backup-mongodb/pbm/restore"
	"github.com/percona/percona-backup-mongodb/pbm/storage"
	"github.com/percona/percona-backup-mongodb/pbm/storage/fs"
	s3storage "github.com/percona/percona-backup-mongodb/pbm/storage/s3"
	"github.com/percona/percona-backup-mongodb/pbm/topo"
)

// --- Backup fixtures ---

// newBackupMeta creates a minimal valid BackupMeta with sensible defaults.
// The name should be an RFC 3339 timestamp (PBM convention).
func newBackupMeta(name string, opts ...func(*backup.BackupMeta)) backup.BackupMeta {
	meta := backup.BackupMeta{
		Name:        name,
		Type:        defs.LogicalBackup,
		Status:      defs.StatusDone,
		Compression: compress.CompressionTypeNone,
		StartTS:     time.Now().Unix(),
		LastWriteTS: primitive.Timestamp{T: uint32(time.Now().Unix()), I: 1},
		Store: backup.Storage{
			StorageConf: config.StorageConf{
				Type: storage.Filesystem,
			},
		},
		MongoVersion: "8.0.0",
		PBMVersion:   "2.13.0",
	}
	for _, fn := range opts {
		fn(&meta)
	}
	return meta
}

func withBackupType(bt defs.BackupType) func(*backup.BackupMeta) {
	return func(m *backup.BackupMeta) { m.Type = bt }
}

func withBackupStatus(s defs.Status) func(*backup.BackupMeta) {
	return func(m *backup.BackupMeta) { m.Status = s }
}

func withBackupProfile(name string) func(*backup.BackupMeta) {
	return func(m *backup.BackupMeta) {
		m.Store.Name = name
		m.Store.IsProfile = true
	}
}

func withBackupCompression(ct compress.CompressionType) func(*backup.BackupMeta) {
	return func(m *backup.BackupMeta) { m.Compression = ct }
}

func withBackupStartTS(ts int64) func(*backup.BackupMeta) {
	return func(m *backup.BackupMeta) { m.StartTS = ts }
}

func withBackupLastWriteTS(t, i uint32) func(*backup.BackupMeta) {
	return func(m *backup.BackupMeta) {
		m.LastWriteTS = primitive.Timestamp{T: t, I: i}
	}
}

func withBackupNamespaces(nss ...string) func(*backup.BackupMeta) {
	return func(m *backup.BackupMeta) { m.Namespaces = nss }
}

func withBackupSrcBackup(src string) func(*backup.BackupMeta) {
	return func(m *backup.BackupMeta) { m.SrcBackup = src }
}

func withBackupSize(size, uncompressed int64) func(*backup.BackupMeta) {
	return func(m *backup.BackupMeta) {
		m.Size = size
		m.SizeUncompressed = uncompressed
	}
}

func withBackupError(msg string) func(*backup.BackupMeta) {
	return func(m *backup.BackupMeta) { m.Err = msg }
}

// --- Restore fixtures ---

// newRestoreMeta creates a minimal valid RestoreMeta.
func newRestoreMeta(name string, opts ...func(*restore.RestoreMeta)) restore.RestoreMeta {
	meta := restore.RestoreMeta{
		Name:    name,
		Backup:  "2024-01-01T00:00:00Z",
		Type:    defs.LogicalBackup,
		Status:  defs.StatusDone,
		StartTS: time.Now().Unix(),
	}
	for _, fn := range opts {
		fn(&meta)
	}
	return meta
}

func withRestoreBackup(name string) func(*restore.RestoreMeta) {
	return func(m *restore.RestoreMeta) { m.Backup = name }
}

func withRestoreStatus(s defs.Status) func(*restore.RestoreMeta) {
	return func(m *restore.RestoreMeta) { m.Status = s }
}

func withRestoreType(bt defs.BackupType) func(*restore.RestoreMeta) {
	return func(m *restore.RestoreMeta) { m.Type = bt }
}

func withRestorePITR(ts int64) func(*restore.RestoreMeta) {
	return func(m *restore.RestoreMeta) { m.PITR = ts }
}

func withRestoreOPID(opid string) func(*restore.RestoreMeta) {
	return func(m *restore.RestoreMeta) { m.OPID = opid }
}

func withRestoreStartTS(ts int64) func(*restore.RestoreMeta) {
	return func(m *restore.RestoreMeta) { m.StartTS = ts }
}

func withRestoreLastTransitionTS(ts int64) func(*restore.RestoreMeta) {
	return func(m *restore.RestoreMeta) { m.LastTransitionTS = ts }
}

func withRestoreNamespaces(nss ...string) func(*restore.RestoreMeta) {
	return func(m *restore.RestoreMeta) { m.Namespaces = nss }
}

func withRestoreReplsets(rs ...restore.RestoreReplset) func(*restore.RestoreMeta) {
	return func(m *restore.RestoreMeta) { m.Replsets = rs }
}

func withRestoreBcpChain(chain ...string) func(*restore.RestoreMeta) {
	return func(m *restore.RestoreMeta) { m.BcpChain = chain }
}

func withRestoreError(msg string) func(*restore.RestoreMeta) {
	return func(m *restore.RestoreMeta) { m.Error = msg }
}

// --- Agent fixtures ---

// newAgentStat creates a minimal valid AgentStat.
func newAgentStat(node, rs string, opts ...func(*topo.AgentStat)) topo.AgentStat {
	stat := topo.AgentStat{
		Node:     node,
		RS:       rs,
		State:    defs.NodeStatePrimary,
		StateStr: "PRIMARY",
		Heartbeat: primitive.Timestamp{
			T: uint32(time.Now().Unix()),
			I: 1,
		},
		AgentVer:      "2.13.0",
		MongoVer:      "8.0.0",
		PBMStatus:     topo.SubsysStatus{OK: true},
		NodeStatus:    topo.SubsysStatus{OK: true},
		StorageStatus: topo.SubsysStatus{OK: true},
	}
	for _, fn := range opts {
		fn(&stat)
	}
	return stat
}

func withAgentState(state defs.NodeState, str string) func(*topo.AgentStat) {
	return func(s *topo.AgentStat) {
		s.State = state
		s.StateStr = str
	}
}

func withAgentHeartbeat(t uint32) func(*topo.AgentStat) {
	return func(s *topo.AgentStat) {
		s.Heartbeat = primitive.Timestamp{T: t, I: 1}
	}
}

func withAgentError(msg string) func(*topo.AgentStat) {
	return func(s *topo.AgentStat) { s.Err = msg }
}

// --- Lock fixtures ---

// newLockData creates a minimal valid LockData.
func newLockData(cmdType ctrl.Command, rs, node string, opts ...func(*lock.LockData)) lock.LockData {
	data := lock.LockData{
		LockHeader: lock.LockHeader{
			Type:    cmdType,
			Replset: rs,
			Node:    node,
			OPID:    primitive.NewObjectID().Hex(),
		},
		Heartbeat: primitive.Timestamp{
			T: uint32(time.Now().Unix()),
			I: 1,
		},
	}
	for _, fn := range opts {
		fn(&data)
	}
	return data
}

func withLockHeartbeat(t uint32) func(*lock.LockData) {
	return func(d *lock.LockData) {
		d.Heartbeat = primitive.Timestamp{T: t, I: 1}
	}
}

// --- Config fixtures ---

// newMainConfig creates a minimal valid main Config with filesystem storage.
func newMainConfig(opts ...func(*config.Config)) config.Config {
	cfg := config.Config{
		Storage: config.StorageConf{
			Type: storage.Filesystem,
			Filesystem: &fs.Config{
				Path: "/tmp/pbm-backups",
			},
		},
	}
	for _, fn := range opts {
		fn(&cfg)
	}
	return cfg
}

func withConfigProfile(name string) func(*config.Config) {
	return func(c *config.Config) {
		c.Name = name
		c.IsProfile = true
	}
}

func withConfigS3Storage(bucket, region string) func(*config.Config) {
	return func(c *config.Config) {
		c.Storage = config.StorageConf{
			Type: storage.S3,
			S3: &s3storage.Config{
				Region: region,
				Bucket: bucket,
			},
		}
	}
}

func withConfigS3Credentials(accessKey, secretKey string) func(*config.Config) {
	return func(c *config.Config) {
		if c.Storage.S3 == nil {
			c.Storage.Type = storage.S3
			c.Storage.S3 = &s3storage.Config{}
		}
		c.Storage.S3.Credentials = s3storage.Credentials{
			AccessKeyID:     storage.MaskedString(accessKey),
			SecretAccessKey: storage.MaskedString(secretKey),
		}
	}
}

func withConfigPITR(enabled bool) func(*config.Config) {
	return func(c *config.Config) {
		level := 3
		c.PITR = &config.PITRConf{
			Enabled:          enabled,
			OplogSpanMin:     10,
			OplogOnly:        true,
			Compression:      compress.CompressionTypeZstandard,
			CompressionLevel: &level,
			Priority:         map[string]float64{"rs0:27017": 1.0},
		}
	}
}

func withConfigBackup(parallel int) func(*config.Config) {
	return func(c *config.Config) {
		timeout := uint32(120)
		c.Backup = &config.BackupConf{
			Compression:            compress.CompressionTypeZstandard,
			NumParallelCollections: parallel,
			OplogSpanMin:           5,
			Timeouts:               &config.BackupTimeouts{Starting: &timeout},
		}
	}
}

func withConfigRestore(batchSize, parallel int) func(*config.Config) {
	return func(c *config.Config) {
		c.Restore = &config.RestoreConf{
			BatchSize:              batchSize,
			NumInsertionWorkers:    2,
			NumParallelCollections: parallel,
			NumDownloadWorkers:     4,
			MaxDownloadBufferMb:    256,
			DownloadChunkMb:        32,
			MongodLocation:         "/usr/bin/mongod",
		}
	}
}

// --- PITR chunk fixtures ---

// newPITRChunk creates a minimal valid OplogChunk.
func newPITRChunk(rs string, startT, endT uint32, opts ...func(*oplog.OplogChunk)) oplog.OplogChunk {
	chunk := oplog.OplogChunk{
		RS:          rs,
		FName:       rs + "/" + "oplog.rs",
		Compression: compress.CompressionTypeNone,
		StartTS:     primitive.Timestamp{T: startT, I: 0},
		EndTS:       primitive.Timestamp{T: endT, I: 0},
	}
	for _, fn := range opts {
		fn(&chunk)
	}
	return chunk
}

func withChunkCompression(ct compress.CompressionType) func(*oplog.OplogChunk) {
	return func(c *oplog.OplogChunk) { c.Compression = ct }
}

func withChunkSize(size int64) func(*oplog.OplogChunk) {
	return func(c *oplog.OplogChunk) { c.Size = size }
}

// --- Log fixtures ---

// newLogEntry creates a minimal valid log Entry with Info severity.
func newLogEntry(msg string, opts ...func(*pbmlog.Entry)) pbmlog.Entry {
	entry := pbmlog.Entry{
		TS:  time.Now().Unix(),
		Msg: msg,
		LogKeys: pbmlog.LogKeys{
			Severity: pbmlog.Info,
		},
	}
	for _, fn := range opts {
		fn(&entry)
	}
	return entry
}

func withLogSeverity(s pbmlog.Severity) func(*pbmlog.Entry) {
	return func(e *pbmlog.Entry) { e.Severity = s }
}

func withLogRS(rs string) func(*pbmlog.Entry) {
	return func(e *pbmlog.Entry) { e.RS = rs }
}

func withLogNode(node string) func(*pbmlog.Entry) {
	return func(e *pbmlog.Entry) { e.Node = node }
}

func withLogEvent(event string) func(*pbmlog.Entry) {
	return func(e *pbmlog.Entry) { e.Event = event }
}

func withLogTS(ts int64) func(*pbmlog.Entry) {
	return func(e *pbmlog.Entry) { e.TS = ts }
}

func withLogObjName(name string) func(*pbmlog.Entry) {
	return func(e *pbmlog.Entry) { e.ObjName = name }
}

func withLogOPID(opid string) func(*pbmlog.Entry) {
	return func(e *pbmlog.Entry) { e.OPID = opid }
}
