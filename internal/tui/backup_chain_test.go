package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// makeBackup creates a minimal sdk.Backup for testing chain logic.
func makeBackup(name string, typ sdk.BackupType, src string, config sdk.ConfigName) sdk.Backup {
	return sdk.Backup{
		Name:       name,
		Type:       typ,
		SrcBackup:  src,
		ConfigName: config,
	}
}

func TestProfileDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "main becomes Main", input: "main", expected: "Main"},
		{name: "named profile passes through", input: "archive", expected: "archive"},
		{name: "empty string passes through", input: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, profileDisplayName(tt.input))
		})
	}
}

func TestGroupBackupsByProfile(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		result := groupBackupsByProfile(nil)
		assert.Empty(t, result)
	})

	t.Run("zero config name defaults to main", func(t *testing.T) {
		backups := []sdk.Backup{
			makeBackup("bk1", sdk.BackupTypeLogical, "", sdk.ConfigName{}),
		}
		result := groupBackupsByProfile(backups)
		require.Contains(t, result, "main")
		assert.Len(t, result["main"], 1)
		assert.Equal(t, "bk1", result["main"][0].Name)
	})

	t.Run("groups by profile name", func(t *testing.T) {
		archive, _ := sdk.NewConfigName("archive")
		backups := []sdk.Backup{
			makeBackup("bk1", sdk.BackupTypeLogical, "", sdk.MainConfig),
			makeBackup("bk2", sdk.BackupTypeLogical, "", archive),
			makeBackup("bk3", sdk.BackupTypePhysical, "", sdk.MainConfig),
		}
		result := groupBackupsByProfile(backups)
		assert.Len(t, result, 2)
		assert.Len(t, result["main"], 2)
		assert.Len(t, result["archive"], 1)
	})
}

func TestSortedProfileNames(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		result := sortedProfileNames(map[string][]sdk.Backup{})
		assert.Empty(t, result)
	})

	t.Run("main comes first", func(t *testing.T) {
		grouped := map[string][]sdk.Backup{
			"zebra":   {},
			"main":    {},
			"archive": {},
		}
		result := sortedProfileNames(grouped)
		require.Len(t, result, 3)
		assert.Equal(t, "main", result[0])
		assert.Equal(t, "archive", result[1])
		assert.Equal(t, "zebra", result[2])
	})

	t.Run("no main profile", func(t *testing.T) {
		grouped := map[string][]sdk.Backup{
			"beta":  {},
			"alpha": {},
		}
		result := sortedProfileNames(grouped)
		require.Len(t, result, 2)
		assert.Equal(t, "alpha", result[0])
		assert.Equal(t, "beta", result[1])
	})

	t.Run("only main", func(t *testing.T) {
		grouped := map[string][]sdk.Backup{
			"main": {},
		}
		result := sortedProfileNames(grouped)
		assert.Equal(t, []string{"main"}, result)
	})
}

func TestResolveIncrChain(t *testing.T) {
	incr := sdk.BackupTypeIncremental
	logical := sdk.BackupTypeLogical

	t.Run("single base returns count 1", func(t *testing.T) {
		backups := []sdk.Backup{
			makeBackup("base", incr, "", sdk.MainConfig),
		}
		base, count := resolveIncrChain(&backups[0], backups)
		assert.Equal(t, "base", base.Name)
		assert.Equal(t, 1, count)
	})

	t.Run("linear chain of 3", func(t *testing.T) {
		backups := []sdk.Backup{
			makeBackup("child2", incr, "child1", sdk.MainConfig),
			makeBackup("child1", incr, "base", sdk.MainConfig),
			makeBackup("base", incr, "", sdk.MainConfig),
		}

		// From child2, should walk up to base.
		base, count := resolveIncrChain(&backups[0], backups)
		assert.Equal(t, "base", base.Name)
		assert.Equal(t, 3, count)

		// From child1, should also walk up to base.
		base, count = resolveIncrChain(&backups[1], backups)
		assert.Equal(t, "base", base.Name)
		assert.Equal(t, 3, count)

		// From base itself.
		base, count = resolveIncrChain(&backups[2], backups)
		assert.Equal(t, "base", base.Name)
		assert.Equal(t, 3, count)
	})

	t.Run("branching chain", func(t *testing.T) {
		// base → child1, base → child2 (two children of the same base)
		backups := []sdk.Backup{
			makeBackup("child2", incr, "base", sdk.MainConfig),
			makeBackup("child1", incr, "base", sdk.MainConfig),
			makeBackup("base", incr, "", sdk.MainConfig),
		}
		base, count := resolveIncrChain(&backups[0], backups)
		assert.Equal(t, "base", base.Name)
		assert.Equal(t, 3, count)
	})

	t.Run("orphaned increment treated as base", func(t *testing.T) {
		// SrcBackup points to a backup not in the list.
		backups := []sdk.Backup{
			makeBackup("orphan", incr, "missing-parent", sdk.MainConfig),
		}
		base, count := resolveIncrChain(&backups[0], backups)
		assert.Equal(t, "orphan", base.Name)
		assert.Equal(t, 1, count)
	})

	t.Run("non-incremental backup", func(t *testing.T) {
		backups := []sdk.Backup{
			makeBackup("logical-bk", logical, "", sdk.MainConfig),
		}
		base, count := resolveIncrChain(&backups[0], backups)
		assert.Equal(t, "logical-bk", base.Name)
		assert.Equal(t, 1, count)
	})
}

func TestChainOrderedItems(t *testing.T) {
	incr := sdk.BackupTypeIncremental
	logical := sdk.BackupTypeLogical
	physical := sdk.BackupTypePhysical

	t.Run("empty input", func(t *testing.T) {
		items := chainOrderedItems("main", nil)
		assert.Empty(t, items)
	})

	t.Run("non-incremental backups pass through in order", func(t *testing.T) {
		backups := []sdk.Backup{
			makeBackup("bk3", logical, "", sdk.MainConfig),
			makeBackup("bk2", physical, "", sdk.MainConfig),
			makeBackup("bk1", logical, "", sdk.MainConfig),
		}
		items := chainOrderedItems("main", backups)
		require.Len(t, items, 3)
		for _, item := range items {
			assert.Equal(t, itemBackup, item.kind)
			assert.Equal(t, "main", item.profile)
		}
		assert.Equal(t, "bk3", items[0].backup.Name)
		assert.Equal(t, "bk2", items[1].backup.Name)
		assert.Equal(t, "bk1", items[2].backup.Name)
	})

	t.Run("linear chain grouped under base", func(t *testing.T) {
		// Newest-first order: child2, child1, base
		backups := []sdk.Backup{
			makeBackup("child2", incr, "child1", sdk.MainConfig),
			makeBackup("child1", incr, "base", sdk.MainConfig),
			makeBackup("base", incr, "", sdk.MainConfig),
		}
		items := chainOrderedItems("main", backups)
		require.Len(t, items, 3)

		// Base emitted at its original position (last in newest-first).
		assert.Equal(t, itemBackup, items[0].kind)
		assert.Equal(t, "base", items[0].backup.Name)

		// Children in oldest-to-newest order under the base.
		assert.Equal(t, itemIncrChild, items[1].kind)
		assert.Equal(t, "child1", items[1].backup.Name)

		assert.Equal(t, itemIncrChild, items[2].kind)
		assert.Equal(t, "child2", items[2].backup.Name)
	})

	t.Run("children before base in input are not duplicated", func(t *testing.T) {
		// This was the bug fixed by the two-pass approach: children appearing
		// before their base in newest-first order must not be emitted as orphans.
		backups := []sdk.Backup{
			makeBackup("child1", incr, "base", sdk.MainConfig),
			makeBackup("base", incr, "", sdk.MainConfig),
		}
		items := chainOrderedItems("main", backups)
		require.Len(t, items, 2)

		assert.Equal(t, itemBackup, items[0].kind)
		assert.Equal(t, "base", items[0].backup.Name)

		assert.Equal(t, itemIncrChild, items[1].kind)
		assert.Equal(t, "child1", items[1].backup.Name)
	})

	t.Run("mixed incremental and non-incremental", func(t *testing.T) {
		// Newest-first: logical-bk, child1, base
		backups := []sdk.Backup{
			makeBackup("logical-bk", logical, "", sdk.MainConfig),
			makeBackup("child1", incr, "base", sdk.MainConfig),
			makeBackup("base", incr, "", sdk.MainConfig),
		}
		items := chainOrderedItems("main", backups)
		require.Len(t, items, 3)

		// logical-bk stays in position.
		assert.Equal(t, itemBackup, items[0].kind)
		assert.Equal(t, "logical-bk", items[0].backup.Name)

		// Chain grouped: base then child.
		assert.Equal(t, itemBackup, items[1].kind)
		assert.Equal(t, "base", items[1].backup.Name)

		assert.Equal(t, itemIncrChild, items[2].kind)
		assert.Equal(t, "child1", items[2].backup.Name)
	})

	t.Run("orphaned increment emitted as regular item", func(t *testing.T) {
		backups := []sdk.Backup{
			makeBackup("orphan", incr, "missing-parent", sdk.MainConfig),
		}
		items := chainOrderedItems("main", backups)
		require.Len(t, items, 1)
		assert.Equal(t, itemBackup, items[0].kind)
		assert.Equal(t, "orphan", items[0].backup.Name)
	})

	t.Run("two separate chains", func(t *testing.T) {
		// Two chains interleaved in newest-first order.
		backups := []sdk.Backup{
			makeBackup("b-child", incr, "b-base", sdk.MainConfig),
			makeBackup("a-child", incr, "a-base", sdk.MainConfig),
			makeBackup("b-base", incr, "", sdk.MainConfig),
			makeBackup("a-base", incr, "", sdk.MainConfig),
		}
		items := chainOrderedItems("main", backups)
		require.Len(t, items, 4)

		// b-base and its child grouped together.
		assert.Equal(t, "b-base", items[0].backup.Name)
		assert.Equal(t, itemBackup, items[0].kind)
		assert.Equal(t, "b-child", items[1].backup.Name)
		assert.Equal(t, itemIncrChild, items[1].kind)

		// a-base and its child grouped together.
		assert.Equal(t, "a-base", items[2].backup.Name)
		assert.Equal(t, itemBackup, items[2].kind)
		assert.Equal(t, "a-child", items[3].backup.Name)
		assert.Equal(t, itemIncrChild, items[3].kind)
	})

	t.Run("profile is set on all items", func(t *testing.T) {
		backups := []sdk.Backup{
			makeBackup("bk1", logical, "", sdk.MainConfig),
			makeBackup("child", incr, "base", sdk.MainConfig),
			makeBackup("base", incr, "", sdk.MainConfig),
		}
		items := chainOrderedItems("archive", backups)
		for _, item := range items {
			assert.Equal(t, "archive", item.profile)
		}
	})

	t.Run("deep chain preserves depth-first oldest-first order", func(t *testing.T) {
		// Chain: base → mid → leaf (newest-first input: leaf, mid, base)
		backups := []sdk.Backup{
			makeBackup("leaf", incr, "mid", sdk.MainConfig),
			makeBackup("mid", incr, "base", sdk.MainConfig),
			makeBackup("base", incr, "", sdk.MainConfig),
		}
		items := chainOrderedItems("main", backups)
		require.Len(t, items, 3)

		assert.Equal(t, "base", items[0].backup.Name)
		assert.Equal(t, "mid", items[1].backup.Name)
		assert.Equal(t, "leaf", items[2].backup.Name)
	})
}
