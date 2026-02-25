package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// newTestBackupsModel creates a minimal backupsModel for testing cursor
// stability. Viewports are zero-sized (harmless — rebuildListContent and
// rebuildDetailContent just write empty strings).
func newTestBackupsModel() backupsModel {
	s := testStyles()
	return backupsModel{
		styles:    &s,
		collapsed: make(map[string]bool),
		listVP:    newPanelViewport(),
		detailVP:  newPanelViewport(),
	}
}

func TestRebuildItemsCursorStability(t *testing.T) {
	makeBackup := func(name, profile string) sdk.Backup {
		cn := sdk.ConfigName{}
		if profile != "main" {
			cn, _ = sdk.NewConfigName(profile)
		}
		return sdk.Backup{
			Name:       name,
			Status:     sdk.StatusDone,
			Type:       sdk.BackupTypeLogical,
			ConfigName: cn,
		}
	}

	backups := []sdk.Backup{
		makeBackup("2024-01-15T10:00:00Z", "main"),
		makeBackup("2024-01-14T10:00:00Z", "main"),
		makeBackup("2024-01-13T10:00:00Z", "s3-profile"),
	}

	timelines := []sdk.Timeline{
		{Start: sdk.Timestamp{T: 100}, End: sdk.Timestamp{T: 200}},
	}

	t.Run("cursor on backup survives rebuild", func(t *testing.T) {
		m := newTestBackupsModel()
		m.timelines = timelines
		m.grouped = groupBackupsByProfile(backups)
		m.profiles = sortedProfileNames(m.grouped)
		m.rebuildItems()

		// Find the second main backup and place cursor there.
		targetName := "2024-01-14T10:00:00Z"
		for i, item := range m.items {
			if item.kind == itemBackup && item.backup.Name == targetName {
				m.backupCursor = i
				break
			}
		}
		require.NotZero(t, m.backupCursor)

		// Rebuild with same data — cursor should stay on the same backup.
		m.rebuildItems()
		sel := m.selectedItem()
		require.NotNil(t, sel)
		assert.Equal(t, itemBackup, sel.kind)
		assert.Equal(t, targetName, sel.backup.Name)
	})

	t.Run("cursor on removed backup falls to zero", func(t *testing.T) {
		m := newTestBackupsModel()
		m.timelines = timelines
		m.grouped = groupBackupsByProfile(backups)
		m.profiles = sortedProfileNames(m.grouped)
		m.rebuildItems()

		// Place cursor on the s3-profile backup.
		targetName := "2024-01-13T10:00:00Z"
		for i, item := range m.items {
			if item.kind == itemBackup && item.backup.Name == targetName {
				m.backupCursor = i
				break
			}
		}

		// Remove s3-profile backups from the data and rebuild.
		remaining := backups[:2] // only main backups
		m.grouped = groupBackupsByProfile(remaining)
		m.profiles = sortedProfileNames(m.grouped)
		m.rebuildItems()

		// Cursor should fall back to 0 (first item).
		assert.Equal(t, 0, m.backupCursor)
	})

	t.Run("cursor on profile header survives rebuild", func(t *testing.T) {
		m := newTestBackupsModel()
		m.timelines = nil
		m.grouped = groupBackupsByProfile(backups)
		m.profiles = sortedProfileNames(m.grouped)
		m.rebuildItems()

		// Place cursor on the s3-profile header.
		for i, item := range m.items {
			if item.kind == itemProfileHeader && item.profile == "s3-profile" {
				m.backupCursor = i
				break
			}
		}

		m.rebuildItems()
		sel := m.selectedItem()
		require.NotNil(t, sel)
		assert.Equal(t, itemProfileHeader, sel.kind)
		assert.Equal(t, "s3-profile", sel.profile)
	})

	t.Run("cursor on PITR timeline survives rebuild", func(t *testing.T) {
		m := newTestBackupsModel()
		m.timelines = timelines
		m.grouped = groupBackupsByProfile(backups)
		m.profiles = sortedProfileNames(m.grouped)
		m.rebuildItems()

		// Place cursor on the first (only) PITR timeline.
		for i, item := range m.items {
			if item.kind == itemPITR {
				m.backupCursor = i
				break
			}
		}

		m.rebuildItems()
		sel := m.selectedItem()
		require.NotNil(t, sel)
		assert.Equal(t, itemPITR, sel.kind)
		assert.Equal(t, uint32(100), sel.timeline.Start.T)
	})
}

func TestSelectedItem(t *testing.T) {
	m := newTestBackupsModel()
	bk := sdk.Backup{Name: "test-backup", Status: sdk.StatusDone, Type: sdk.BackupTypeLogical}
	m.items = []backupItem{
		{kind: itemProfileHeader, profile: "main"},
		{kind: itemBackup, backup: &bk, profile: "main"},
	}

	t.Run("returns item at cursor", func(t *testing.T) {
		m.mode = listBackups
		m.backupCursor = 1
		item := m.selectedItem()
		require.NotNil(t, item)
		assert.Equal(t, itemBackup, item.kind)
		assert.Equal(t, "test-backup", item.backup.Name)
	})

	t.Run("returns nil in restore mode", func(t *testing.T) {
		m.mode = listRestores
		m.backupCursor = 1
		assert.Nil(t, m.selectedItem())
	})

	t.Run("returns nil for out of bounds", func(t *testing.T) {
		m.mode = listBackups
		m.backupCursor = 99
		assert.Nil(t, m.selectedItem())
	})
}

func TestSetRestoreDataCursorClamping(t *testing.T) {
	m := newTestBackupsModel()
	m.mode = listRestores
	m.restoreCursor = 5

	m.setRestoreData(restoresData{
		restores: []sdk.Restore{
			{Name: "r1", Status: sdk.StatusDone},
			{Name: "r2", Status: sdk.StatusDone},
		},
	})

	assert.Equal(t, 1, m.restoreCursor, "cursor should clamp to last index")
}
