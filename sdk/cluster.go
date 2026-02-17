package sdk

import (
	"context"
	"time"
)

// ClusterService provides read access to cluster topology,
// agent statuses, and currently running operations.
type ClusterService interface {
	// Members returns the replica sets (shards) in the cluster.
	Members(ctx context.Context) ([]ReplicaSet, error)

	// Agents returns the status of all PBM agents in the cluster.
	Agents(ctx context.Context) ([]Agent, error)

	// RunningOperations returns currently active PBM operations.
	RunningOperations(ctx context.Context) ([]Operation, error)

	// ClusterTime returns the current MongoDB cluster timestamp.
	ClusterTime(ctx context.Context) (Timestamp, error)
}

// ReplicaSet represents a MongoDB replica set (or shard).
type ReplicaSet struct {
	Name  string
	Nodes []Node
}

// Node represents a single MongoDB node in a replica set.
type Node struct {
	Host string
	Role NodeRole
}

// Agent represents a PBM agent running on a MongoDB node.
type Agent struct {
	Node       string
	ReplicaSet string
	Version    string
	Role       NodeRole
	OK         bool
	Stale      bool
	Errors     []string
}

// Operation represents a currently running PBM operation.
type Operation struct {
	Type    string // e.g. "Backup", "Restore", "Delete"
	OPID    string
	Name    string
	StartTS time.Time
	Status  Status
}
