package sdk

import "context"

// ClusterService provides read access to cluster topology,
// agent statuses, and currently running operations.
//
// Example — check cluster health:
//
//	agents, err := client.Cluster.Agents(ctx)
//	for _, a := range agents {
//	    if a.Stale {
//	        fmt.Printf("WARNING: agent %s is stale\n", a.Node)
//	    } else if !a.OK {
//	        fmt.Printf("ERROR: agent %s: %v\n", a.Node, a.Errors)
//	    }
//	}
//
//	ops, _ := client.Cluster.RunningOperations(ctx)
//	if len(ops) > 0 {
//	    fmt.Printf("running: %s on %s\n", ops[0].Type, ops[0].Node)
//	}
type ClusterService interface {
	// Members returns the replica sets (shards) in the cluster, including
	// each set's nodes and their roles.
	Members(ctx context.Context) ([]ReplicaSet, error)

	// Agents returns the status of all PBM agents in the cluster. Agents
	// that have not sent a heartbeat recently are marked as Stale.
	Agents(ctx context.Context) ([]Agent, error)

	// RunningOperations returns currently active PBM operations, derived
	// from non-stale distributed locks. Returns an empty slice when idle.
	RunningOperations(ctx context.Context) ([]Operation, error)

	// ClusterTime returns the current MongoDB cluster timestamp. This is
	// useful for comparing against backup and PITR timestamps.
	ClusterTime(ctx context.Context) (Timestamp, error)
}

// ReplicaSet represents a MongoDB replica set (or shard).
type ReplicaSet struct {
	Name  string // replica set name
	Nodes []Node // member nodes
}

// Node represents a single MongoDB node in a replica set.
type Node struct {
	Host string   // hostname:port
	Role NodeRole // primary, secondary, arbiter, etc.
}

// Agent represents a PBM agent running on a MongoDB node.
type Agent struct {
	Node       string   // MongoDB node hostname this agent runs on
	ReplicaSet string   // replica set the agent belongs to
	Version    string   // PBM agent version
	Role       NodeRole // role of the underlying MongoDB node
	OK         bool     // true if the agent is healthy and reporting normally
	Stale      bool     // true if the agent's heartbeat is older than the cluster time threshold
	Errors     []string // agent-reported errors (connectivity issues, config problems, etc.)
}

// Operation represents a currently running PBM operation,
// derived from active distributed locks.
type Operation struct {
	Type       CommandType // the type of operation (backup, restore, etc.)
	OPID       string      // operation ID
	ReplicaSet string      // replica set holding the lock
	Node       string      // node holding the lock
}
