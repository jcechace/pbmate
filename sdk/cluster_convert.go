package sdk

import (
	"log/slog"
	"strings"

	"github.com/percona/percona-backup-mongodb/pbm/defs"
	"github.com/percona/percona-backup-mongodb/pbm/lock"
	"github.com/percona/percona-backup-mongodb/pbm/topo"
)

// convertReplicaSet converts a PBM Shard to an SDK ReplicaSet.
// Node hostnames are parsed from the Shard.Host URI ("rs/host1:port,host2:port").
// Individual node roles are not available at the topology level.
func convertReplicaSet(s topo.Shard) ReplicaSet {
	return ReplicaSet{
		Name:  s.RS,
		Nodes: parseHostNodes(s.Host),
	}
}

// parseHostNodes parses a PBM host URI into a slice of Nodes.
// The format is "rs/host1:port,host2:port" or just "host1:port,host2:port".
// Roles are not available from topology data and default to zero value.
func parseHostNodes(host string) []Node {
	// Strip the replset prefix if present.
	_, after, found := strings.Cut(host, "/")
	if found {
		host = after
	}

	if host == "" {
		return nil
	}

	parts := strings.Split(host, ",")
	nodes := make([]Node, len(parts))
	for i, h := range parts {
		nodes[i] = Node{Host: h}
	}
	return nodes
}

// convertAgent converts a PBM AgentStat to an SDK Agent.
func convertAgent(a topo.AgentStat, clusterTime uint32) Agent {
	ok, errs := a.OK()

	errStrs := make([]string, 0, len(errs))
	for _, e := range errs {
		errStrs = append(errStrs, e.Error())
	}
	if a.Err != "" {
		errStrs = append(errStrs, a.Err)
	}

	return Agent{
		Node:       a.Node,
		ReplicaSet: a.RS,
		Version:    a.AgentVer,
		Role:       agentNodeRole(a),
		OK:         ok && a.Err == "",
		Stale:      isStaleAgent(a, clusterTime),
		Errors:     errStrs,
	}
}

// agentNodeRole derives the SDK NodeRole from a PBM AgentStat.
// Priority: arbiter > delayed > hidden > primary/secondary.
func agentNodeRole(a topo.AgentStat) NodeRole {
	switch {
	case a.Arbiter:
		return NodeRoleArbiter
	case a.DelaySecs > 0:
		return NodeRoleDelayed
	case a.Hidden:
		return NodeRoleHidden
	case a.State == defs.NodeStatePrimary:
		return NodeRolePrimary
	case a.State == defs.NodeStateSecondary:
		return NodeRoleSecondary
	default:
		return NodeRole{}
	}
}

// isStaleAgent checks if an agent's heartbeat is stale relative to cluster time.
func isStaleAgent(a topo.AgentStat, clusterTime uint32) bool {
	return a.Heartbeat.T+defs.StaleFrameSec < clusterTime
}

// convertOperation converts a PBM LockData to an SDK Operation.
func convertOperation(ld lock.LockData) Operation {
	ct, err := ParseCommandType(string(ld.Type))
	if err != nil {
		slog.Warn("unknown PBM command type", "value", string(ld.Type))
	}
	return Operation{
		Type:       ct,
		OPID:       ld.OPID,
		ReplicaSet: ld.Replset,
		Node:       ld.Node,
	}
}
