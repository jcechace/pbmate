//go:build integration

package integtest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/percona/percona-backup-mongodb/pbm/ctrl"
)

// lastCommand reads the most recently inserted command from the pbmCmd
// collection. This is used to verify that SDK operations dispatch the
// correct PBM commands without needing a running PBM agent.
func (h *testHarness) lastCommand(t *testing.T) ctrl.Cmd {
	t.Helper()

	ctx := context.Background()
	opts := options.FindOne().SetSort(bson.D{{Key: "ts", Value: -1}})

	var cmd ctrl.Cmd
	err := h.collection("pbmCmd").FindOne(ctx, bson.D{}, opts).Decode(&cmd)
	require.NoError(t, err, "read last command from pbmCmd")
	return cmd
}

// commandCount returns the number of documents in the pbmCmd collection.
func (h *testHarness) commandCount(t *testing.T) int64 {
	t.Helper()
	count, err := h.collection("pbmCmd").CountDocuments(context.Background(), bson.D{})
	require.NoError(t, err, "count commands in pbmCmd")
	return count
}
