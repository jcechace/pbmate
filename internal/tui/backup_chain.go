package tui

import (
	"maps"
	"slices"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// profileDisplayName returns a human-readable name for a profile.
func profileDisplayName(profile string) string {
	if profile == "main" {
		return "Main"
	}
	return profile
}

// groupBackupsByProfile groups backups by their ConfigName.
func groupBackupsByProfile(backups []sdk.Backup) map[string][]sdk.Backup {
	m := make(map[string][]sdk.Backup)
	for _, bk := range backups {
		name := bk.ConfigName.String()
		if name == "" {
			name = "main"
		}
		m[name] = append(m[name], bk)
	}
	return m
}

// sortedProfileNames returns profile names sorted with "main" always first,
// then remaining profiles in alphabetical order.
func sortedProfileNames(grouped map[string][]sdk.Backup) []string {
	var names []string
	hasMain := false
	for name := range maps.Keys(grouped) {
		if name == "main" {
			hasMain = true
		} else {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	if hasMain {
		names = append([]string{"main"}, names...)
	}
	return names
}

// resolveIncrChain finds the base of the incremental chain containing bk and
// counts total chain members (base + all transitive children). The backups
// slice should contain all backups for the same profile.
func resolveIncrChain(bk *sdk.Backup, backups []sdk.Backup) (base *sdk.Backup, count int) {
	byName := make(map[string]*sdk.Backup, len(backups))
	for i := range backups {
		byName[backups[i].Name] = &backups[i]
	}

	// Walk up to find the base.
	base = bk
	for base.SrcBackup != "" {
		parent, ok := byName[base.SrcBackup]
		if !ok {
			break // orphaned — treat current as base
		}
		base = parent
	}

	// Build reverse index and count chain members.
	childrenOf := make(map[string][]*sdk.Backup)
	for i := range backups {
		b := &backups[i]
		if b.Type.Equal(sdk.BackupTypeIncremental) && b.SrcBackup != "" {
			childrenOf[b.SrcBackup] = append(childrenOf[b.SrcBackup], b)
		}
	}

	count = 1 // base itself
	var walk func(name string)
	walk = func(name string) {
		for _, child := range childrenOf[name] {
			count++
			walk(child.Name)
		}
	}
	walk(base.Name)

	return base, count
}

// chainOrderedItems builds backup items for a profile, grouping incremental
// chains under their base. The input backups must be ordered newest-first
// (as returned by the SDK). Non-incremental backups and incremental bases
// keep their chronological position. Chain children are pulled out of the
// flat list and placed directly under their base, ordered oldest-to-newest.
//
// Orphaned increments (whose SrcBackup points to a backup not in this list)
// are emitted as regular itemBackup entries in their original position.
func chainOrderedItems(profile string, backups []sdk.Backup) []backupItem {
	// Index backups by name for chain lookups.
	byName := make(map[string]*sdk.Backup, len(backups))
	for i := range backups {
		byName[backups[i].Name] = &backups[i]
	}

	// Build reverse index: parent name → children (in original newest-first order).
	childrenOf := make(map[string][]*sdk.Backup)
	for i := range backups {
		bk := &backups[i]
		if bk.Type.Equal(sdk.BackupTypeIncremental) && bk.SrcBackup != "" {
			childrenOf[bk.SrcBackup] = append(childrenOf[bk.SrcBackup], bk)
		}
	}

	// walkChain collects all transitive children of a parent, depth-first,
	// in oldest-to-newest order (reversed from the newest-first input).
	var walkChain func(parentName string, out *[]*sdk.Backup)
	walkChain = func(parentName string, out *[]*sdk.Backup) {
		children := childrenOf[parentName]
		// Reverse to get oldest-first within each level.
		for i := len(children) - 1; i >= 0; i-- {
			child := children[i]
			*out = append(*out, child)
			walkChain(child.Name, out)
		}
	}

	// First pass: identify all chain children up front so they are skipped
	// in the main loop. Without this, children that appear before their base
	// in the newest-first order would be emitted as top-level orphans.
	consumed := make(map[string]bool)
	for i := range backups {
		bk := &backups[i]
		if bk.Type.Equal(sdk.BackupTypeIncremental) && bk.SrcBackup == "" {
			var chain []*sdk.Backup
			walkChain(bk.Name, &chain)
			for _, child := range chain {
				consumed[child.Name] = true
			}
		}
	}

	// Second pass: emit items, grouping chain children under their base.
	var items []backupItem
	for i := range backups {
		bk := &backups[i]

		if consumed[bk.Name] {
			continue
		}

		// Incremental base: emit base + chain children.
		if bk.Type.Equal(sdk.BackupTypeIncremental) && bk.SrcBackup == "" {
			items = append(items, backupItem{
				kind:    itemBackup,
				profile: profile,
				backup:  bk,
			})
			var chain []*sdk.Backup
			walkChain(bk.Name, &chain)
			for _, child := range chain {
				items = append(items, backupItem{
					kind:    itemIncrChild,
					profile: profile,
					backup:  child,
				})
			}
			continue
		}

		// Regular backup (or orphaned increment).
		items = append(items, backupItem{
			kind:    itemBackup,
			profile: profile,
			backup:  bk,
		})
	}

	return items
}
