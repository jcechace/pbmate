package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusMarshalRoundTrip(t *testing.T) {
	all := []Status{
		StatusInit, StatusReady, StatusStarting, StatusRunning,
		StatusDumpDone, StatusCopyReady, StatusCopyDone,
		StatusPartlyDone, StatusDone, StatusCancelled, StatusError,
		StatusDown, StatusCleanupCluster,
	}
	for _, s := range all {
		t.Run(s.String(), func(t *testing.T) {
			b, err := s.MarshalText()
			require.NoError(t, err)

			var got Status
			err = got.UnmarshalText(b)
			require.NoError(t, err)
			assert.Equal(t, s, got)
		})
	}
}

func TestStatusUnmarshalUnknown(t *testing.T) {
	var s Status
	err := s.UnmarshalText([]byte("bogus"))
	assert.Error(t, err)
	assert.True(t, s.IsZero())
}

func TestBackupTypeMarshalRoundTrip(t *testing.T) {
	all := []BackupType{
		BackupTypeLogical, BackupTypePhysical,
		BackupTypeIncremental, BackupTypeExternal,
	}
	for _, bt := range all {
		t.Run(bt.String(), func(t *testing.T) {
			b, err := bt.MarshalText()
			require.NoError(t, err)

			var got BackupType
			err = got.UnmarshalText(b)
			require.NoError(t, err)
			assert.Equal(t, bt, got)
		})
	}
}

func TestBackupTypeUnmarshalUnknown(t *testing.T) {
	var bt BackupType
	err := bt.UnmarshalText([]byte("bogus"))
	assert.Error(t, err)
	assert.True(t, bt.IsZero())
}

func TestCompressionTypeMarshalRoundTrip(t *testing.T) {
	all := []CompressionType{
		CompressionTypeNone, CompressionTypeGZIP, CompressionTypePGZIP,
		CompressionTypeSNAPPY, CompressionTypeLZ4, CompressionTypeS2,
		CompressionTypeZSTD,
	}
	for _, ct := range all {
		t.Run(ct.String(), func(t *testing.T) {
			b, err := ct.MarshalText()
			require.NoError(t, err)

			var got CompressionType
			err = got.UnmarshalText(b)
			require.NoError(t, err)
			assert.Equal(t, ct, got)
		})
	}
}

func TestCompressionTypeUnmarshalUnknown(t *testing.T) {
	var ct CompressionType
	err := ct.UnmarshalText([]byte("bogus"))
	assert.Error(t, err)
	assert.True(t, ct.IsZero())
}

func TestStorageTypeMarshalRoundTrip(t *testing.T) {
	all := []StorageType{
		StorageTypeS3, StorageTypeMinio, StorageTypeGCS,
		StorageTypeAzure, StorageTypeFilesystem,
		StorageTypeBlackhole, StorageTypeOSS,
	}
	for _, st := range all {
		t.Run(st.String(), func(t *testing.T) {
			b, err := st.MarshalText()
			require.NoError(t, err)

			var got StorageType
			err = got.UnmarshalText(b)
			require.NoError(t, err)
			assert.Equal(t, st, got)
		})
	}
}

func TestStorageTypeUnmarshalUnknown(t *testing.T) {
	var st StorageType
	err := st.UnmarshalText([]byte("bogus"))
	assert.Error(t, err)
	assert.True(t, st.IsZero())
}

func TestNodeRoleMarshalRoundTrip(t *testing.T) {
	all := []NodeRole{
		NodeRolePrimary, NodeRoleSecondary,
		NodeRoleArbiter, NodeRoleHidden, NodeRoleDelayed,
	}
	for _, nr := range all {
		t.Run(nr.String(), func(t *testing.T) {
			b, err := nr.MarshalText()
			require.NoError(t, err)

			var got NodeRole
			err = got.UnmarshalText(b)
			require.NoError(t, err)
			assert.Equal(t, nr, got)
		})
	}
}

func TestNodeRoleUnmarshalUnknown(t *testing.T) {
	var nr NodeRole
	err := nr.UnmarshalText([]byte("bogus"))
	assert.Error(t, err)
	assert.True(t, nr.IsZero())
}

func TestLogSeverityMarshalRoundTrip(t *testing.T) {
	all := []LogSeverity{
		LogSeverityDebug, LogSeverityInfo,
		LogSeverityWarning, LogSeverityError, LogSeverityFatal,
	}
	for _, ls := range all {
		t.Run(ls.String(), func(t *testing.T) {
			b, err := ls.MarshalText()
			require.NoError(t, err)

			var got LogSeverity
			err = got.UnmarshalText(b)
			require.NoError(t, err)
			assert.Equal(t, ls, got)
		})
	}
}

func TestLogSeverityUnmarshalUnknown(t *testing.T) {
	var ls LogSeverity
	err := ls.UnmarshalText([]byte("bogus"))
	assert.Error(t, err)
	assert.True(t, ls.IsZero())
}

func TestCommandTypeMarshalRoundTrip(t *testing.T) {
	all := []CommandType{
		CmdTypeBackup, CmdTypeRestore, CmdTypeReplay,
		CmdTypeCancelBackup, CmdTypeResync, CmdTypePITR,
		CmdTypeDelete, CmdTypeDeletePITR, CmdTypeCleanup,
		CmdTypeAddProfile, CmdTypeRemoveProfile,
	}
	for _, ct := range all {
		t.Run(ct.String(), func(t *testing.T) {
			b, err := ct.MarshalText()
			require.NoError(t, err)

			var got CommandType
			err = got.UnmarshalText(b)
			require.NoError(t, err)
			assert.Equal(t, ct, got)
		})
	}
}

func TestCommandTypeUnmarshalUnknown(t *testing.T) {
	var ct CommandType
	err := ct.UnmarshalText([]byte("bogus"))
	assert.Error(t, err)
	assert.True(t, ct.IsZero())
}

func TestConfigNameMarshalRoundTrip(t *testing.T) {
	t.Run("main config", func(t *testing.T) {
		b, err := MainConfig.MarshalText()
		require.NoError(t, err)

		var got ConfigName
		err = got.UnmarshalText(b)
		require.NoError(t, err)
		assert.Equal(t, MainConfig, got)
	})

	t.Run("named profile", func(t *testing.T) {
		cn, err := NewConfigName("my-profile")
		require.NoError(t, err)

		b, err := cn.MarshalText()
		require.NoError(t, err)

		var got ConfigName
		err = got.UnmarshalText(b)
		require.NoError(t, err)
		assert.Equal(t, cn, got)
	})
}

func TestConfigNameUnmarshalEmpty(t *testing.T) {
	// Empty string should normalize to MainConfig, not error.
	var cn ConfigName
	err := cn.UnmarshalText([]byte(""))
	require.NoError(t, err)
	assert.Equal(t, MainConfig, cn)
}

func TestConfigNameNewEmpty(t *testing.T) {
	_, err := NewConfigName("")
	assert.Error(t, err)
}

func TestTimestampBefore(t *testing.T) {
	tests := []struct {
		name string
		a, b Timestamp
		want bool
	}{
		{"T less", Timestamp{T: 100, I: 0}, Timestamp{T: 200, I: 0}, true},
		{"T greater", Timestamp{T: 200, I: 0}, Timestamp{T: 100, I: 0}, false},
		{"T equal I less", Timestamp{T: 100, I: 1}, Timestamp{T: 100, I: 5}, true},
		{"T equal I greater", Timestamp{T: 100, I: 5}, Timestamp{T: 100, I: 1}, false},
		{"T equal I equal", Timestamp{T: 100, I: 3}, Timestamp{T: 100, I: 3}, false},
		{"zero before nonzero", Timestamp{}, Timestamp{T: 1}, true},
		{"both zero", Timestamp{}, Timestamp{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.a.Before(tt.b))
		})
	}
}

func TestTimestampAfter(t *testing.T) {
	tests := []struct {
		name string
		a, b Timestamp
		want bool
	}{
		{"T greater", Timestamp{T: 200}, Timestamp{T: 100}, true},
		{"T less", Timestamp{T: 100}, Timestamp{T: 200}, false},
		{"T equal I greater", Timestamp{T: 100, I: 5}, Timestamp{T: 100, I: 1}, true},
		{"T equal I equal", Timestamp{T: 100, I: 3}, Timestamp{T: 100, I: 3}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.a.After(tt.b))
		})
	}
}
