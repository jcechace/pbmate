//go:build integration

// Package integtest contains integration tests for the PBMate SDK.
// These tests require a running Docker daemon and use testcontainers
// to spin up a single-node Percona Server for MongoDB replica set.
//
// Run with: task sdk:integration
package integtest

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

const (
	// mongoImage is the Docker image used for integration tests.
	// Percona Server for MongoDB matches what PBM targets in production.
	mongoImage = "percona/percona-server-mongodb:8.0"

	// adminDB is the MongoDB database where PBM stores all its collections.
	adminDB = "admin"
)

// pbmCollections lists all PBM collections in the admin database.
// Used by cleanup to reset state between tests.
var pbmCollections = []string{
	"pbmConfig",
	"pbmBackups",
	"pbmRestores",
	"pbmCmd",
	"pbmLock",
	"pbmLockOp",
	"pbmPITRChunks",
	"pbmPITR",
	"pbmOpLog",
	"pbmAgents",
	"pbmLog",
}

// testHarness holds the test infrastructure shared across all integration tests.
// It provides an SDK client for testing the public API and a raw MongoDB handle
// for seeding test data and verifying side effects.
type testHarness struct {
	client *sdk.Client     // SDK client under test (public API)
	db     *mongo.Database // admin database for seeding and cleanup
}

// h is the package-level harness initialized in TestMain.
var h *testHarness

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start a single-node Percona Server for MongoDB replica set.
	container, err := mongodb.Run(ctx, mongoImage, mongodb.WithReplicaSet("rs"))
	if err != nil {
		log.Fatalf("start mongodb container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			log.Printf("terminate container: %v", err)
		}
	}()

	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("get connection string: %v", err)
	}

	// Create a raw MongoDB client for seeding and verification.
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(connStr))
	if err != nil {
		log.Fatalf("connect raw mongo client: %v", err)
	}
	defer func() {
		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Printf("disconnect raw mongo client: %v", err)
		}
	}()

	// Create the SDK client under test.
	sdkClient, err := sdk.NewClient(ctx, sdk.WithMongoURI(connStr))
	if err != nil {
		log.Fatalf("create sdk client: %v", err)
	}
	defer func() {
		if err := sdkClient.Close(ctx); err != nil {
			log.Printf("close sdk client: %v", err)
		}
	}()

	h = &testHarness{
		client: sdkClient,
		db:     mongoClient.Database(adminDB),
	}

	os.Exit(m.Run())
}

// cleanup drops all PBM collections to reset state between tests.
// Call this at the start of each test or register via t.Cleanup.
func (h *testHarness) cleanup(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	for _, name := range pbmCollections {
		err := h.db.Collection(name).Drop(ctx)
		require.NoError(t, err, "drop collection %s", name)
	}
}

// collection returns a handle to a named collection in the admin database.
func (h *testHarness) collection(name string) *mongo.Collection {
	return h.db.Collection(name)
}

// Smoke test to verify the harness starts and the SDK client can connect.
func TestHarnessConnected(t *testing.T) {
	h.cleanup(t)

	// A basic SDK call that should work on an empty PBM database.
	// Config.Get returns ErrNotFound when no config exists — that's the
	// expected state, confirming the SDK is connected and querying MongoDB.
	_, err := h.client.Config.Get(context.Background())
	require.ErrorIs(t, err, sdk.ErrNotFound)
}

// TestCleanupDropsCollections verifies that cleanup actually removes seeded data.
func TestCleanupDropsCollections(t *testing.T) {
	h.cleanup(t)

	ctx := context.Background()

	// Insert a dummy document into pbmBackups.
	_, err := h.collection("pbmBackups").InsertOne(ctx, map[string]string{"name": "test"})
	require.NoError(t, err)

	// Verify it exists.
	count, err := h.collection("pbmBackups").CountDocuments(ctx, map[string]string{})
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	// Cleanup and verify it's gone.
	h.cleanup(t)
	count, err = h.collection("pbmBackups").CountDocuments(ctx, map[string]string{})
	require.NoError(t, err)
	require.Equal(t, int64(0), count, "cleanup should drop all documents")
}
