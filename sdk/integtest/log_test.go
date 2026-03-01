//go:build integration

package integtest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pbmlog "github.com/percona/percona-backup-mongodb/pbm/log"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

func TestLogGet(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	ts := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	h.seedLog(t, newLogEntry("backup started",
		withLogTS(ts.Unix()),
		withLogSeverity(pbmlog.Info),
		withLogRS("rs"),
		withLogNode("rs/localhost:27017"),
		withLogEvent("backup"),
		withLogObjName("2024-01-01T00:00:00Z"),
	))

	entries, err := h.client.Logs.Get(ctx, sdk.GetLogsOptions{})
	require.NoError(t, err)
	require.Len(t, entries, 1)

	e := entries[0]
	assert.Equal(t, "backup started", e.Message)
	assert.Equal(t, ts, e.Timestamp)
	assert.True(t, e.Severity.Equal(sdk.LogSeverityInfo))
	assert.Equal(t, "rs", e.Attrs[sdk.LogKeyReplicaSet])
	assert.Equal(t, "rs/localhost:27017", e.Attrs[sdk.LogKeyNode])
	assert.Equal(t, "backup", e.Attrs[sdk.LogKeyEvent])
	assert.Equal(t, "2024-01-01T00:00:00Z", e.Attrs[sdk.LogKeyObjName])
}

func TestLogGetEmpty(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	entries, err := h.client.Logs.Get(ctx, sdk.GetLogsOptions{})
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLogGetWithLimit(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		h.seedLog(t, newLogEntry("msg",
			withLogTS(int64(1000+i)),
		))
	}

	entries, err := h.client.Logs.Get(ctx, sdk.GetLogsOptions{Limit: 2})
	require.NoError(t, err)
	require.Len(t, entries, 2)
}

func TestLogGetFilterBySeverity(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	// Seed entries at every severity level.
	h.seedLog(t, newLogEntry("fatal", withLogTS(1005), withLogSeverity(pbmlog.Fatal)))
	h.seedLog(t, newLogEntry("error", withLogTS(1004), withLogSeverity(pbmlog.Error)))
	h.seedLog(t, newLogEntry("warning", withLogTS(1003), withLogSeverity(pbmlog.Warning)))
	h.seedLog(t, newLogEntry("info", withLogTS(1002), withLogSeverity(pbmlog.Info)))
	h.seedLog(t, newLogEntry("debug", withLogTS(1001), withLogSeverity(pbmlog.Debug)))

	// Filter for Warning — PBM uses $lte on severity int, so includes
	// Fatal(0), Error(1), Warning(2) but not Info(3) or Debug(4).
	entries, err := h.client.Logs.Get(ctx, sdk.GetLogsOptions{
		LogFilter: sdk.LogFilter{Severity: sdk.LogSeverityWarning},
	})
	require.NoError(t, err)
	require.Len(t, entries, 3)

	// Verify we got exactly the right severity levels.
	severities := make(map[string]bool)
	for _, e := range entries {
		severities[e.Severity.String()] = true
	}
	assert.True(t, severities["F"], "should include Fatal")
	assert.True(t, severities["E"], "should include Error")
	assert.True(t, severities["W"], "should include Warning")
}

func TestLogGetFilterByReplicaSet(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedLog(t, newLogEntry("rs0 msg", withLogTS(1002), withLogRS("rs0")))
	h.seedLog(t, newLogEntry("rs1 msg", withLogTS(1001), withLogRS("rs1")))

	entries, err := h.client.Logs.Get(ctx, sdk.GetLogsOptions{
		LogFilter: sdk.LogFilter{ReplicaSet: "rs0"},
	})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "rs0 msg", entries[0].Message)
}

func TestLogGetFilterByEvent(t *testing.T) {
	h.cleanup(t)
	ctx := context.Background()

	h.seedLog(t, newLogEntry("backup msg", withLogTS(1002), withLogEvent("backup")))
	h.seedLog(t, newLogEntry("restore msg", withLogTS(1001), withLogEvent("restore")))

	entries, err := h.client.Logs.Get(ctx, sdk.GetLogsOptions{
		LogFilter: sdk.LogFilter{Event: "backup"},
	})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "backup msg", entries[0].Message)
}
