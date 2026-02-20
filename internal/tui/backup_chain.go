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
func resolveIncrChain(bk *sdk.Backup, backups []sdk.Backup) (baseName string, count int) {
	base := sdk.FindChainBase(*bk, backups)

	for _, c := range sdk.GroupIncrementalChains(backups) {
		if c.Base.Name == base.Name {
			return c.Base.Name, c.Len()
		}
	}

	// Non-incremental or not found in any chain.
	return base.Name, 1
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
	chains := sdk.GroupIncrementalChains(backups)

	// Index original slice for pointer lookups (backupItem stores *sdk.Backup).
	byName := make(map[string]*sdk.Backup, len(backups))
	for i := range backups {
		byName[backups[i].Name] = &backups[i]
	}

	// Build lookup structures from chains.
	chainByBase := make(map[string]sdk.BackupChain, len(chains))
	consumed := make(map[string]bool)
	for _, c := range chains {
		chainByBase[c.Base.Name] = c
		for _, child := range c.Children {
			consumed[child.Name] = true
		}
	}

	var items []backupItem
	for i := range backups {
		bk := &backups[i]

		if consumed[bk.Name] {
			continue
		}

		items = append(items, backupItem{
			kind:    itemBackup,
			profile: profile,
			backup:  bk,
		})

		// If this is a chain base, emit children right after.
		if c, ok := chainByBase[bk.Name]; ok {
			for _, child := range c.Children {
				items = append(items, backupItem{
					kind:    itemIncrChild,
					profile: profile,
					backup:  byName[child.Name],
				})
			}
		}
	}

	return items
}
