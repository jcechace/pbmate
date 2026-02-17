package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/percona/percona-backup-mongodb/pbm/oplog"
)

func TestConvertTimeline(t *testing.T) {
	tl := oplog.Timeline{
		Start: 1700000000,
		End:   1700003600,
		Size:  1024,
	}

	result := convertTimeline(tl)

	assert.Equal(t, Timestamp{T: 1700000000}, result.Start)
	assert.Equal(t, Timestamp{T: 1700003600}, result.End)
}

func TestConvertTimelines(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result := convertTimelines(nil)
		assert.Nil(t, result)
	})

	t.Run("empty input", func(t *testing.T) {
		result := convertTimelines([]oplog.Timeline{})
		assert.Nil(t, result)
	})

	t.Run("multiple timelines", func(t *testing.T) {
		tlns := []oplog.Timeline{
			{Start: 1700000000, End: 1700003600},
			{Start: 1700010000, End: 1700013600},
		}

		result := convertTimelines(tlns)

		assert.Len(t, result, 2)
		assert.Equal(t, Timestamp{T: 1700000000}, result[0].Start)
		assert.Equal(t, Timestamp{T: 1700003600}, result[0].End)
		assert.Equal(t, Timestamp{T: 1700010000}, result[1].Start)
		assert.Equal(t, Timestamp{T: 1700013600}, result[1].End)
	})
}

func TestCollectPITRErrors(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		replsets := []oplog.PITRReplset{
			{Name: "rs0", Status: oplog.StatusRunning},
			{Name: "rs1", Status: oplog.StatusReady},
		}

		result := collectPITRErrors(replsets)
		assert.Empty(t, result)
	})

	t.Run("single error", func(t *testing.T) {
		replsets := []oplog.PITRReplset{
			{Name: "rs0", Status: oplog.StatusError, Error: "storage unreachable"},
		}

		result := collectPITRErrors(replsets)
		assert.Equal(t, "rs0: storage unreachable", result)
	})

	t.Run("multiple errors", func(t *testing.T) {
		replsets := []oplog.PITRReplset{
			{Name: "rs0", Status: oplog.StatusError, Error: "storage unreachable"},
			{Name: "rs1", Status: oplog.StatusRunning},
			{Name: "rs2", Status: oplog.StatusError, Error: "disk full"},
		}

		result := collectPITRErrors(replsets)
		assert.Equal(t, "rs0: storage unreachable; rs2: disk full", result)
	})

	t.Run("error status with empty message", func(t *testing.T) {
		replsets := []oplog.PITRReplset{
			{Name: "rs0", Status: oplog.StatusError, Error: ""},
		}

		result := collectPITRErrors(replsets)
		assert.Empty(t, result)
	})

	t.Run("nil replsets", func(t *testing.T) {
		result := collectPITRErrors(nil)
		assert.Empty(t, result)
	})
}
