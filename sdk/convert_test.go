package sdk

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/percona/percona-backup-mongodb/pbm/compress"
	"github.com/percona/percona-backup-mongodb/pbm/defs"
	"github.com/percona/percona-backup-mongodb/pbm/storage"
)

func TestConvertTimestamp(t *testing.T) {
	ts := convertTimestamp(primitive.Timestamp{T: 1700000000, I: 42})
	assert.Equal(t, uint32(1700000000), ts.T)
	assert.Equal(t, uint32(42), ts.I)
}

func TestConvertTimestampZero(t *testing.T) {
	ts := convertTimestamp(primitive.Timestamp{})
	assert.True(t, ts.IsZero())
}

func TestConvertTimestampToPBM(t *testing.T) {
	pts := convertTimestampToPBM(Timestamp{T: 1700000000, I: 42})
	assert.Equal(t, uint32(1700000000), pts.T)
	assert.Equal(t, uint32(42), pts.I)
}

func TestConvertTimestampRoundTrip(t *testing.T) {
	orig := primitive.Timestamp{T: 1700000000, I: 7}
	result := convertTimestampToPBM(convertTimestamp(orig))
	assert.Equal(t, orig, result)
}

func TestConvertUnixToTime(t *testing.T) {
	tt := convertUnixToTime(1700000000)
	require.False(t, tt.IsZero())
	assert.Equal(t, int64(1700000000), tt.Unix())
	assert.Equal(t, time.UTC, tt.Location())
}

func TestConvertUnixToTimeZero(t *testing.T) {
	tt := convertUnixToTime(0)
	assert.True(t, tt.IsZero())
}

func TestConvertStatus(t *testing.T) {
	tests := []struct {
		input defs.Status
		want  Status
	}{
		{defs.StatusDone, StatusDone},
		{defs.StatusError, StatusError},
		{defs.StatusRunning, StatusRunning},
		{defs.StatusCancelled, StatusCancelled},
		{defs.StatusInit, StatusInit},
	}
	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			assert.Equal(t, tt.want, convertStatus(tt.input))
		})
	}
}

func TestConvertStatusUnknown(t *testing.T) {
	s := convertStatus(defs.Status("bogus"))
	assert.True(t, s.IsZero())
}

func TestStatusIsTerminal(t *testing.T) {
	terminal := []Status{StatusDone, StatusError, StatusCancelled, StatusPartlyDone}
	for _, s := range terminal {
		assert.True(t, s.IsTerminal(), "expected %s to be terminal", s)
	}

	nonTerminal := []Status{
		StatusInit, StatusReady, StatusStarting, StatusRunning,
		StatusDumpDone, StatusCopyReady, StatusCopyDone,
		StatusDown, StatusCleanupCluster,
	}
	for _, s := range nonTerminal {
		assert.False(t, s.IsTerminal(), "expected %s to be non-terminal", s)
	}

	assert.False(t, Status{}.IsTerminal(), "zero value should be non-terminal")
}

func TestConvertBackupType(t *testing.T) {
	tests := []struct {
		input defs.BackupType
		want  BackupType
	}{
		{defs.LogicalBackup, BackupTypeLogical},
		{defs.PhysicalBackup, BackupTypePhysical},
		{defs.IncrementalBackup, BackupTypeIncremental},
		{defs.ExternalBackup, BackupTypeExternal},
	}
	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			assert.Equal(t, tt.want, convertBackupType(tt.input))
		})
	}
}

func TestConvertBackupTypeUnknown(t *testing.T) {
	bt := convertBackupType(defs.BackupType("bogus"))
	assert.True(t, bt.IsZero())
}

func TestConvertCompressionType(t *testing.T) {
	tests := []struct {
		input compress.CompressionType
		want  CompressionType
	}{
		{compress.CompressionTypeNone, CompressionTypeNone},
		{compress.CompressionTypeGZIP, CompressionTypeGZIP},
		{compress.CompressionTypePGZIP, CompressionTypePGZIP},
		{compress.CompressionTypeSNAPPY, CompressionTypeSNAPPY},
		{compress.CompressionTypeLZ4, CompressionTypeLZ4},
		{compress.CompressionTypeS2, CompressionTypeS2},
		{compress.CompressionTypeZstandard, CompressionTypeZSTD},
	}
	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			assert.Equal(t, tt.want, convertCompressionType(tt.input))
		})
	}
}

func TestConvertCompressionTypeUnknown(t *testing.T) {
	ct := convertCompressionType(compress.CompressionType("bogus"))
	assert.True(t, ct.IsZero())
}

func TestConvertStorageType(t *testing.T) {
	tests := []struct {
		input storage.Type
		want  StorageType
	}{
		{storage.S3, StorageTypeS3},
		{storage.GCS, StorageTypeGCS},
		{storage.Azure, StorageTypeAzure},
		{storage.Filesystem, StorageTypeFilesystem},
		{storage.Minio, StorageTypeMinio},
		{storage.Blackhole, StorageTypeBlackhole},
		{storage.OSS, StorageTypeOSS},
	}
	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			assert.Equal(t, tt.want, convertStorageType(tt.input))
		})
	}
}

func TestConvertStorageTypeUnknown(t *testing.T) {
	st := convertStorageType(storage.Type("bogus"))
	assert.True(t, st.IsZero())
}

func TestConvertConfigName(t *testing.T) {
	assert.Equal(t, MainConfig, convertConfigName(""))
	assert.Equal(t, "my-profile", convertConfigName("my-profile").String())
}

func TestOperationError(t *testing.T) {
	err := &OperationError{Name: "2024-01-15T10:30:00Z", Message: "storage unreachable"}
	assert.Equal(t, `operation "2024-01-15T10:30:00Z" failed: storage unreachable`, err.Error())

	// Verify errors.As works for type assertion by callers.
	var opErr *OperationError
	assert.True(t, errors.As(err, &opErr))
	assert.Equal(t, "2024-01-15T10:30:00Z", opErr.Name)
	assert.Equal(t, "storage unreachable", opErr.Message)
}
