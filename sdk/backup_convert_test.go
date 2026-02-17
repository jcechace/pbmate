package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/percona/percona-backup-mongodb/pbm/backup"
	"github.com/percona/percona-backup-mongodb/pbm/compress"
	"github.com/percona/percona-backup-mongodb/pbm/defs"
)

func TestConvertBackup(t *testing.T) {
	boolTrue := true
	meta := &backup.BackupMeta{
		Name:             "2024-01-15T10:30:00Z",
		OPID:             "abc123",
		Type:             defs.LogicalBackup,
		Status:           defs.StatusDone,
		Compression:      compress.CompressionTypeGZIP,
		Store:            backup.Storage{Name: "my-profile"},
		StartTS:          1705312200,
		LastWriteTS:      primitive.Timestamp{T: 1705312300, I: 1},
		LastTransitionTS: 1705312400,
		Size:             1024000,
		SizeUncompressed: 2048000,
		Namespaces:       []string{"db1.coll1"},
		SrcBackup:        "parent-backup",
		MongoVersion:     "7.0.4",
		FCV:              "7.0",
		PBMVersion:       "2.4.0",
		Err:              "",
		Replsets: []backup.BackupReplset{
			{
				Name:             "rs0",
				Status:           defs.StatusDone,
				Node:             "rs00:27017",
				LastWriteTS:      primitive.Timestamp{T: 1705312300, I: 1},
				LastTransitionTS: 1705312400,
				IsConfigSvr:      &boolTrue,
				Error:            "",
			},
		},
	}

	b := convertBackup(meta)

	assert.Equal(t, "2024-01-15T10:30:00Z", b.Name)
	assert.Equal(t, "abc123", b.OPID)
	assert.Equal(t, BackupLogical, b.Type)
	assert.Equal(t, StatusDone, b.Status)
	assert.Equal(t, CompressionGZIP, b.Compression)
	assert.Equal(t, "my-profile", b.ConfigName.String())
	assert.Equal(t, int64(1705312200), b.StartTS.Unix())
	assert.Equal(t, uint32(1705312300), b.LastWriteTS.T)
	assert.Equal(t, uint32(1), b.LastWriteTS.I)
	assert.Equal(t, int64(1705312400), b.LastTransitionTS.Unix())
	assert.Equal(t, int64(1024000), b.Size)
	assert.Equal(t, int64(2048000), b.SizeUncompressed)
	assert.Equal(t, []string{"db1.coll1"}, b.Namespaces)
	assert.Equal(t, "parent-backup", b.SrcBackup)
	assert.Equal(t, "7.0.4", b.MongoVersion)
	assert.Equal(t, "7.0", b.FCV)
	assert.Equal(t, "2.4.0", b.PBMVersion)
	assert.Empty(t, b.Error)

	// Replsets
	assert.Len(t, b.Replsets, 1)
	rs := b.Replsets[0]
	assert.Equal(t, "rs0", rs.Name)
	assert.Equal(t, StatusDone, rs.Status)
	assert.Equal(t, "rs00:27017", rs.Node)
	assert.Equal(t, uint32(1705312300), rs.LastWriteTS.T)
	assert.True(t, rs.IsConfigSvr)
	assert.Empty(t, rs.Error)
}

func TestConvertBackupMainConfig(t *testing.T) {
	meta := &backup.BackupMeta{
		Name:   "test",
		Status: defs.StatusDone,
		Store:  backup.Storage{Name: ""},
	}

	b := convertBackup(meta)
	assert.Equal(t, MainConfig, b.ConfigName)
}

func TestConvertBackupReplsetsNil(t *testing.T) {
	meta := &backup.BackupMeta{
		Name:   "test",
		Status: defs.StatusDone,
	}

	b := convertBackup(meta)
	assert.Nil(t, b.Replsets)
}

func TestConvertBackupReplsetIsConfigSvrNil(t *testing.T) {
	rs := &backup.BackupReplset{
		Name:        "rs0",
		Status:      defs.StatusDone,
		IsConfigSvr: nil,
	}

	result := convertBackupReplset(rs)
	assert.False(t, result.IsConfigSvr)
}

func TestDerefBool(t *testing.T) {
	assert.False(t, derefBool(nil))

	v := true
	assert.True(t, derefBool(&v))

	v = false
	assert.False(t, derefBool(&v))
}
