package sdk

import (
	"context"
	"fmt"

	"github.com/percona/percona-backup-mongodb/pbm/connect"
)

// Client provides access to PBM operations through domain-specific services.
type Client struct {
	Backups  BackupService
	Restores RestoreService
	Config   ConfigService
	Cluster  ClusterService
	PITR     PITRService
	Logs     LogService

	conn connect.Client
}

type options struct {
	mongoURI string
	appName  string
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

// NewClient creates a new PBM client with the given options.
// At least one connection option (e.g. WithMongoURI) must be provided.
// The caller must call Close when the client is no longer needed.
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
	conn, err := connect.Connect(ctx, o.mongoURI, o.appName)
	if err != nil {
		return nil, fmt.Errorf("connect to PBM: %w", err)
	}

	c := &Client{conn: conn}
	c.Backups = &backupServiceImpl{conn: conn}
	c.Restores = &restoreServiceImpl{conn: conn}
	c.Config = &configServiceImpl{conn: conn}
	c.Cluster = &clusterServiceImpl{conn: conn}
	c.PITR = &pitrServiceImpl{conn: conn}
	c.Logs = &logServiceImpl{conn: conn}
	return c, nil
}

// Close disconnects from the backend. The Client must not be used after Close.
func (c *Client) Close(ctx context.Context) error {
	return c.conn.Disconnect(ctx)
}
