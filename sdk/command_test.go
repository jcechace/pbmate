package sdk

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func intPtr(n int) *int { return &n }

func TestStartLogicalBackupValidate(t *testing.T) {
	t.Run("valid full backup", func(t *testing.T) {
		cmd := StartLogicalBackup{}
		assert.NoError(t, cmd.Validate())
	})

	t.Run("valid selective backup without users-and-roles", func(t *testing.T) {
		cmd := StartLogicalBackup{Namespaces: []string{"db1.*"}}
		assert.NoError(t, cmd.Validate())
	})

	t.Run("valid selective backup with users-and-roles", func(t *testing.T) {
		cmd := StartLogicalBackup{
			Namespaces:    []string{"db1.*", "db2.*"},
			UsersAndRoles: true,
		}
		assert.NoError(t, cmd.Validate())
	})

	t.Run("users-and-roles without namespaces", func(t *testing.T) {
		cmd := StartLogicalBackup{UsersAndRoles: true}
		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "namespaces must be set")
	})

	t.Run("users-and-roles with specific collection", func(t *testing.T) {
		cmd := StartLogicalBackup{
			Namespaces:    []string{"db1.*", "db2.specific"},
			UsersAndRoles: true,
		}
		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "whole-database")
		assert.Contains(t, err.Error(), "db2.specific")
	})
}

func TestStartPhysicalBackupValidate(t *testing.T) {
	tests := []struct {
		name string
		cmd  StartPhysicalBackup
	}{
		{
			name: "zero value",
			cmd:  StartPhysicalBackup{},
		},
		{
			name: "with all fields",
			cmd: StartPhysicalBackup{
				ConfigName:       MainConfig,
				Compression:      CompressionTypeZSTD,
				CompressionLevel: intPtr(3),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NoError(t, tt.cmd.Validate())
		})
	}
}

func TestStartIncrementalBackupValidate(t *testing.T) {
	tests := []struct {
		name string
		cmd  StartIncrementalBackup
	}{
		{
			name: "base",
			cmd:  StartIncrementalBackup{Base: true},
		},
		{
			name: "non-base (extends chain)",
			cmd:  StartIncrementalBackup{Base: false},
		},
		{
			name: "with all fields",
			cmd: StartIncrementalBackup{
				ConfigName:       MainConfig,
				Compression:      CompressionTypeLZ4,
				CompressionLevel: intPtr(6),
				Base:             true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NoError(t, tt.cmd.Validate())
		})
	}
}

func TestStartSnapshotRestoreValidate(t *testing.T) {
	t.Run("valid minimal", func(t *testing.T) {
		cmd := StartSnapshotRestore{BackupName: "2026-01-15T10:30:00Z"}
		assert.NoError(t, cmd.Validate())
	})

	t.Run("valid with all fields", func(t *testing.T) {
		cmd := StartSnapshotRestore{
			BackupName:    "2026-01-15T10:30:00Z",
			Namespaces:    []string{"db1.*"},
			NamespaceFrom: "srcDB.srcColl",
			NamespaceTo:   "dstDB.dstColl",
			UsersAndRoles: true,
		}
		assert.NoError(t, cmd.Validate())
	})

	t.Run("missing backup name", func(t *testing.T) {
		cmd := StartSnapshotRestore{}
		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "backup name is required")
	})

	t.Run("namespace remap half set", func(t *testing.T) {
		cmd := StartSnapshotRestore{
			BackupName:    "2026-01-15T10:30:00Z",
			NamespaceFrom: "srcDB.srcColl",
		}
		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "namespace-from and namespace-to must be set together")
	})

	t.Run("users-and-roles without namespaces", func(t *testing.T) {
		cmd := StartSnapshotRestore{
			BackupName:    "2026-01-15T10:30:00Z",
			UsersAndRoles: true,
		}
		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "namespaces must be set")
	})

	t.Run("users-and-roles with specific collection", func(t *testing.T) {
		cmd := StartSnapshotRestore{
			BackupName:    "2026-01-15T10:30:00Z",
			Namespaces:    []string{"db1.coll"},
			UsersAndRoles: true,
		}
		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "whole-database")
	})
}

func TestStartPITRRestoreValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cmd := StartPITRRestore{
			BackupName: "2026-01-15T10:30:00Z",
			Target:     Timestamp{T: 1700000000, I: 1},
		}
		assert.NoError(t, cmd.Validate())
	})

	t.Run("missing backup name", func(t *testing.T) {
		cmd := StartPITRRestore{Target: Timestamp{T: 1700000000}}
		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "backup name is required")
	})

	t.Run("missing target", func(t *testing.T) {
		cmd := StartPITRRestore{BackupName: "2026-01-15T10:30:00Z"}
		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "target timestamp is required")
	})

	t.Run("namespace remap half set", func(t *testing.T) {
		cmd := StartPITRRestore{
			BackupName:  "2026-01-15T10:30:00Z",
			Target:      Timestamp{T: 1700000000},
			NamespaceTo: "dstDB.dstColl",
		}
		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "namespace-from and namespace-to must be set together")
	})

	t.Run("users-and-roles with whole-database namespaces", func(t *testing.T) {
		cmd := StartPITRRestore{
			BackupName:    "2026-01-15T10:30:00Z",
			Target:        Timestamp{T: 1700000000},
			Namespaces:    []string{"db1.*"},
			UsersAndRoles: true,
		}
		assert.NoError(t, cmd.Validate())
	})
}

func TestDeleteBackupByNameValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cmd := DeleteBackupByName{Name: "2026-01-15T10:30:00Z"}
		assert.NoError(t, cmd.Validate())
	})

	t.Run("empty name", func(t *testing.T) {
		cmd := DeleteBackupByName{}
		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})
}

func TestDeleteBackupsBeforeValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cmd := DeleteBackupsBefore{OlderThan: time.Now().Add(-24 * time.Hour)}
		assert.NoError(t, cmd.Validate())
	})

	t.Run("zero older-than", func(t *testing.T) {
		cmd := DeleteBackupsBefore{}
		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "older-than time must be set")
	})

	t.Run("future older-than", func(t *testing.T) {
		cmd := DeleteBackupsBefore{OlderThan: time.Now().Add(24 * time.Hour)}
		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "in the future")
	})
}

func TestDeletePITRBeforeValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cmd := DeletePITRBefore{OlderThan: time.Now().Add(-24 * time.Hour)}
		assert.NoError(t, cmd.Validate())
	})

	t.Run("zero older-than", func(t *testing.T) {
		cmd := DeletePITRBefore{}
		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "older-than time must be set")
	})

	t.Run("future older-than", func(t *testing.T) {
		cmd := DeletePITRBefore{OlderThan: time.Now().Add(24 * time.Hour)}
		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "in the future")
	})
}

func TestDeletePITRAllValidate(t *testing.T) {
	cmd := DeletePITRAll{}
	assert.NoError(t, cmd.Validate())
}

func TestResyncProfileValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cmd := ResyncProfile{Name: "my-s3"}
		assert.NoError(t, cmd.Validate())
	})

	t.Run("empty name", func(t *testing.T) {
		cmd := ResyncProfile{}
		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})
}

func TestResyncMainValidate(t *testing.T) {
	cmd := ResyncMain{}
	assert.NoError(t, cmd.Validate())
}

func TestResyncAllProfilesValidate(t *testing.T) {
	cmd := ResyncAllProfiles{}
	assert.NoError(t, cmd.Validate())
}

func TestRemoveProfileCommandValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cmd := RemoveProfileCommand{Name: "my-s3"}
		assert.NoError(t, cmd.Validate())
	})

	t.Run("empty name", func(t *testing.T) {
		cmd := RemoveProfileCommand{}
		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})
}

func TestCancelBackupCommandValidate(t *testing.T) {
	cmd := CancelBackupCommand{}
	assert.NoError(t, cmd.Validate())
}
