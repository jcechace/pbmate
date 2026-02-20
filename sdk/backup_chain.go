package sdk

// BackupChain represents an incremental backup chain: a base backup plus
// its ordered children (oldest-to-newest).
//
// PBM uses a parent-pointer model for incremental backups: each increment's
// SrcBackup field points to its immediate parent. A base backup is an
// incremental with an empty SrcBackup. Chains are always linear in practice
// (no branching), but the grouping logic handles branches correctly.
type BackupChain struct {
	Base     Backup   // the chain's base backup (SrcBackup == "")
	Children []Backup // incremental children in chronological (oldest-first) order
}

// All returns every backup in the chain in chronological order (base first,
// then children oldest-to-newest).
func (c BackupChain) All() []Backup {
	out := make([]Backup, 0, c.Len())
	out = append(out, c.Base)
	out = append(out, c.Children...)
	return out
}

// Len returns the total number of backups in the chain (1 base + N children).
func (c BackupChain) Len() int {
	return 1 + len(c.Children)
}

// GroupIncrementalChains groups incremental backups into chains.
// Non-incremental backups are silently ignored. Chains are returned in the
// order their bases appear in the input. Children within each chain are
// ordered oldest-to-newest (depth-first walk reversed from the input order).
//
// Orphaned increments whose SrcBackup points to a backup not in the input
// are treated as standalone bases (single-element chains).
func GroupIncrementalChains(backups []Backup) []BackupChain {
	// Index all backups by name.
	byName := make(map[string]*Backup, len(backups))
	for i := range backups {
		byName[backups[i].Name] = &backups[i]
	}

	// Build reverse index: parent name → children (in input order).
	childrenOf := make(map[string][]*Backup)
	for i := range backups {
		bk := &backups[i]
		if bk.IsIncremental() && bk.SrcBackup != "" {
			childrenOf[bk.SrcBackup] = append(childrenOf[bk.SrcBackup], bk)
		}
	}

	// walkChain collects all transitive children depth-first, in
	// oldest-to-newest order (reversed from newest-first input convention).
	var walkChain func(parentName string, out *[]Backup)
	walkChain = func(parentName string, out *[]Backup) {
		children := childrenOf[parentName]
		for i := len(children) - 1; i >= 0; i-- {
			*out = append(*out, *children[i])
			walkChain(children[i].Name, out)
		}
	}

	// Identify all chain members so orphans and non-bases can be skipped
	// in the main loop.
	consumed := make(map[string]bool)
	for i := range backups {
		bk := &backups[i]
		if bk.IsIncrementalBase() {
			var chain []Backup
			walkChain(bk.Name, &chain)
			for j := range chain {
				consumed[chain[j].Name] = true
			}
		}
	}

	var chains []BackupChain
	for i := range backups {
		bk := &backups[i]
		if !bk.IsIncremental() || consumed[bk.Name] {
			continue
		}

		if bk.IsIncrementalBase() {
			var children []Backup
			walkChain(bk.Name, &children)
			chains = append(chains, BackupChain{Base: *bk, Children: children})
			continue
		}

		// Orphaned increment: parent not in input. Treat as standalone chain.
		chains = append(chains, BackupChain{Base: *bk})
	}

	return chains
}

// FindChainBase walks up parent pointers to find the base backup of the
// incremental chain containing bk. If bk is not incremental or is already
// a base, it is returned as-is. If a parent is not found in the provided
// slice (orphan), the walk stops and the current backup is returned.
func FindChainBase(bk Backup, backups []Backup) Backup {
	byName := make(map[string]*Backup, len(backups))
	for i := range backups {
		byName[backups[i].Name] = &backups[i]
	}

	cur := bk
	for cur.SrcBackup != "" {
		parent, ok := byName[cur.SrcBackup]
		if !ok {
			break
		}
		cur = *parent
	}
	return cur
}
