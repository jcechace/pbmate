package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mkBackup creates a minimal Backup for chain testing.
func mkBackup(name string, typ BackupType, src string) Backup {
	return Backup{
		Name:      name,
		Type:      typ,
		SrcBackup: src,
	}
}

func TestBackupChain_Len(t *testing.T) {
	t.Run("base only", func(t *testing.T) {
		c := BackupChain{Base: mkBackup("base", BackupTypeIncremental, "")}
		assert.Equal(t, 1, c.Len())
	})

	t.Run("base with children", func(t *testing.T) {
		c := BackupChain{
			Base: mkBackup("base", BackupTypeIncremental, ""),
			Children: []Backup{
				mkBackup("c1", BackupTypeIncremental, "base"),
				mkBackup("c2", BackupTypeIncremental, "c1"),
			},
		}
		assert.Equal(t, 3, c.Len())
	})
}

func TestBackupChain_All(t *testing.T) {
	c := BackupChain{
		Base: mkBackup("base", BackupTypeIncremental, ""),
		Children: []Backup{
			mkBackup("c1", BackupTypeIncremental, "base"),
			mkBackup("c2", BackupTypeIncremental, "c1"),
		},
	}
	all := c.All()
	require.Len(t, all, 3)
	assert.Equal(t, "base", all[0].Name)
	assert.Equal(t, "c1", all[1].Name)
	assert.Equal(t, "c2", all[2].Name)
}

func TestGroupIncrementalChains(t *testing.T) {
	incr := BackupTypeIncremental
	logical := BackupTypeLogical

	t.Run("empty input", func(t *testing.T) {
		chains := GroupIncrementalChains(nil)
		assert.Empty(t, chains)
	})

	t.Run("no incremental backups", func(t *testing.T) {
		backups := []Backup{
			mkBackup("bk1", logical, ""),
			mkBackup("bk2", BackupTypePhysical, ""),
		}
		chains := GroupIncrementalChains(backups)
		assert.Empty(t, chains)
	})

	t.Run("single base", func(t *testing.T) {
		backups := []Backup{
			mkBackup("base", incr, ""),
		}
		chains := GroupIncrementalChains(backups)
		require.Len(t, chains, 1)
		assert.Equal(t, "base", chains[0].Base.Name)
		assert.Empty(t, chains[0].Children)
		assert.Equal(t, 1, chains[0].Len())
	})

	t.Run("linear chain of 3 (newest-first input)", func(t *testing.T) {
		backups := []Backup{
			mkBackup("child2", incr, "child1"),
			mkBackup("child1", incr, "base"),
			mkBackup("base", incr, ""),
		}
		chains := GroupIncrementalChains(backups)
		require.Len(t, chains, 1)

		assert.Equal(t, "base", chains[0].Base.Name)
		require.Len(t, chains[0].Children, 2)
		assert.Equal(t, "child1", chains[0].Children[0].Name)
		assert.Equal(t, "child2", chains[0].Children[1].Name)
	})

	t.Run("two separate chains", func(t *testing.T) {
		backups := []Backup{
			mkBackup("b-child", incr, "b-base"),
			mkBackup("a-child", incr, "a-base"),
			mkBackup("b-base", incr, ""),
			mkBackup("a-base", incr, ""),
		}
		chains := GroupIncrementalChains(backups)
		require.Len(t, chains, 2)

		assert.Equal(t, "b-base", chains[0].Base.Name)
		require.Len(t, chains[0].Children, 1)
		assert.Equal(t, "b-child", chains[0].Children[0].Name)

		assert.Equal(t, "a-base", chains[1].Base.Name)
		require.Len(t, chains[1].Children, 1)
		assert.Equal(t, "a-child", chains[1].Children[0].Name)
	})

	t.Run("mixed incremental and non-incremental", func(t *testing.T) {
		backups := []Backup{
			mkBackup("logical-bk", logical, ""),
			mkBackup("child1", incr, "base"),
			mkBackup("base", incr, ""),
		}
		chains := GroupIncrementalChains(backups)
		require.Len(t, chains, 1)
		assert.Equal(t, "base", chains[0].Base.Name)
		require.Len(t, chains[0].Children, 1)
		assert.Equal(t, "child1", chains[0].Children[0].Name)
	})

	t.Run("orphaned increment forms standalone chain", func(t *testing.T) {
		backups := []Backup{
			mkBackup("orphan", incr, "missing-parent"),
		}
		chains := GroupIncrementalChains(backups)
		require.Len(t, chains, 1)
		assert.Equal(t, "orphan", chains[0].Base.Name)
		assert.Empty(t, chains[0].Children)
	})

	t.Run("deep chain preserves oldest-first order", func(t *testing.T) {
		backups := []Backup{
			mkBackup("leaf", incr, "mid"),
			mkBackup("mid", incr, "base"),
			mkBackup("base", incr, ""),
		}
		chains := GroupIncrementalChains(backups)
		require.Len(t, chains, 1)

		all := chains[0].All()
		require.Len(t, all, 3)
		assert.Equal(t, "base", all[0].Name)
		assert.Equal(t, "mid", all[1].Name)
		assert.Equal(t, "leaf", all[2].Name)
	})

	t.Run("branching chain", func(t *testing.T) {
		// base → child1, base → child2
		backups := []Backup{
			mkBackup("child2", incr, "base"),
			mkBackup("child1", incr, "base"),
			mkBackup("base", incr, ""),
		}
		chains := GroupIncrementalChains(backups)
		require.Len(t, chains, 1)
		assert.Equal(t, "base", chains[0].Base.Name)
		assert.Equal(t, 3, chains[0].Len())
	})
}

func TestFindChainBase(t *testing.T) {
	incr := BackupTypeIncremental
	logical := BackupTypeLogical

	t.Run("base returns itself", func(t *testing.T) {
		backups := []Backup{
			mkBackup("base", incr, ""),
		}
		base := FindChainBase(backups[0], backups)
		assert.Equal(t, "base", base.Name)
	})

	t.Run("walks up to base", func(t *testing.T) {
		backups := []Backup{
			mkBackup("child2", incr, "child1"),
			mkBackup("child1", incr, "base"),
			mkBackup("base", incr, ""),
		}
		base := FindChainBase(backups[0], backups)
		assert.Equal(t, "base", base.Name)
	})

	t.Run("orphan stops at itself", func(t *testing.T) {
		backups := []Backup{
			mkBackup("orphan", incr, "missing"),
		}
		base := FindChainBase(backups[0], backups)
		assert.Equal(t, "orphan", base.Name)
	})

	t.Run("non-incremental returns itself", func(t *testing.T) {
		backups := []Backup{
			mkBackup("logical-bk", logical, ""),
		}
		base := FindChainBase(backups[0], backups)
		assert.Equal(t, "logical-bk", base.Name)
	})

	t.Run("mid-chain walks to base", func(t *testing.T) {
		backups := []Backup{
			mkBackup("leaf", incr, "mid"),
			mkBackup("mid", incr, "base"),
			mkBackup("base", incr, ""),
		}
		base := FindChainBase(backups[1], backups)
		assert.Equal(t, "base", base.Name)
	})
}
