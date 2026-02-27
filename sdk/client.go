package sdk

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/percona/percona-backup-mongodb/pbm/connect"
)

// Client provides access to PBM operations through domain-specific services.
// Create one with [NewClient] and close it with [Client.Close] when done.
//
// Each field exposes a service interface for a specific PBM domain.
// Services are safe for concurrent use.
//
// Example usage:
//
//	client, err := sdk.NewClient(ctx, sdk.WithMongoURI("mongodb://localhost:27017"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close(ctx)
//
//	backups, _ := client.Backups.List(ctx, sdk.ListBackupsOptions{Limit: 10})
//	agents, _ := client.Cluster.Agents(ctx)
type Client struct {
	Backups  BackupService  // backup operations: list, get, start, cancel, delete
	Restores RestoreService // restore operations: list, get, start, wait
	Config   ConfigService  // PBM configuration and storage profiles
	Cluster  ClusterService // cluster topology, agents, running operations
	PITR     PITRService    // PITR status and oplog timelines
	Logs     LogService     // PBM log access and streaming

	conn connect.Client
}

type options struct {
	mongoURI       string
	appName        string
	logger         *slog.Logger
	connectTimeout *time.Duration
}

// Option configures how the Client is created.
type Option func(*options)

// WithMongoURI configures the client to connect via a MongoDB URI.
func WithMongoURI(uri string) Option {
	return func(o *options) { o.mongoURI = uri }
}

// WithAppName sets the application name used in the backend connection.
func WithAppName(name string) Option {
	return func(o *options) { o.appName = name }
}

// WithLogger sets the structured logger used by the SDK.
// If not set, the SDK produces no log output.
func WithLogger(l *slog.Logger) Option {
	return func(o *options) { o.logger = l }
}

// WithConnectTimeout sets a timeout for the initial connection to MongoDB.
// This bounds the time spent on server discovery, authentication, and topology
// resolution during [NewClient]. If not set, the connection uses the MongoDB
// driver's default server selection timeout (30s).
func WithConnectTimeout(d time.Duration) Option {
	return func(o *options) { o.connectTimeout = &d }
}

// NewClient creates a new PBM client with the given options.
// At least one connection option (e.g. [WithMongoURI]) must be provided.
// The caller must call [Client.Close] when the client is no longer needed.
//
// Example:
//
//	client, err := sdk.NewClient(ctx,
//	    sdk.WithMongoURI("mongodb://localhost:27017"),
//	    sdk.WithLogger(slog.Default()),
//	)
//	if err != nil {
//	    return fmt.Errorf("connect to PBM: %w", err)
//	}
//	defer client.Close(ctx)
func NewClient(ctx context.Context, opts ...Option) (*Client, error) {
	o := &options{appName: "pbmate-sdk"}
	for _, opt := range opts {
		opt(o)
	}

	switch {
	case o.mongoURI != "":
		return newMongoClient(ctx, o)
	default:
		return nil, fmt.Errorf("no connection backend configured")
	}
}

func newMongoClient(ctx context.Context, o *options) (*Client, error) {
	log := o.logger
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}

	// TODO(pbm-fix): Once PBM's connect.Connect accepts variadic MongoOption,
	// use connect.ServerSelectionTimeout instead of a context deadline.
	if o.connectTimeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *o.connectTimeout)
		defer cancel()
	}

	conn, err := connect.Connect(ctx, o.mongoURI, o.appName)
	if err != nil {
		return nil, fmt.Errorf("connect to PBM: %w", err)
	}

	log.InfoContext(ctx, "connected to PBM")

	cmds := &commandServiceImpl{conn: conn, log: log.With("service", "command")}

	c := &Client{conn: conn}
	backups := &backupServiceImpl{conn: conn, cmds: cmds, log: log.With("service", "backup")}
	c.Backups = backups
	c.Restores = &restoreServiceImpl{conn: conn, cmds: cmds, backups: backups, log: log.With("service", "restore")}
	c.Config = &configServiceImpl{conn: conn, cmds: cmds, log: log.With("service", "config")}
	c.Cluster = &clusterServiceImpl{conn: conn, log: log.With("service", "cluster")}
	c.PITR = &pitrServiceImpl{conn: conn, cmds: cmds, log: log.With("service", "pitr")}
	c.Logs = &logServiceImpl{conn: conn, log: log.With("service", "log")}
	return c, nil
}

// Close disconnects from the backend. The Client must not be used after Close.
func (c *Client) Close(ctx context.Context) error {
	if err := c.conn.Disconnect(ctx); err != nil {
		return fmt.Errorf("close client: %w", err)
	}
	return nil
}
