package sdk

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/percona/percona-backup-mongodb/pbm/backup"
)

func TestTranslateCanDeleteError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantErr error
		wantMsg string
	}{
		{
			name:    "nil passthrough",
			err:     nil,
			wantErr: nil,
		},
		{
			name:    "backup in progress",
			err:     backup.ErrBackupInProgress,
			wantErr: ErrBackupInProgress,
		},
		{
			name:    "base for PITR",
			err:     backup.ErrBaseForPITR,
			wantErr: ErrDeleteProtectedByPITR,
		},
		{
			name:    "generic error wrapped",
			err:     errors.New("something unexpected"),
			wantMsg: "can delete: something unexpected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := translateCanDeleteError(tt.err)
			switch {
			case tt.wantErr != nil:
				assert.ErrorIs(t, got, tt.wantErr)
			case tt.wantMsg != "":
				assert.EqualError(t, got, tt.wantMsg)
			default:
				assert.NoError(t, got)
			}
		})
	}
}
