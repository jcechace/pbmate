package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// --- parseCustomDate ---

func TestParseCustomDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantY   int
		wantM   time.Month
		wantD   int
		wantH   int
		wantMin int
		wantErr bool
	}{
		{
			name:  "date only",
			input: "2026-02-20",
			wantY: 2026, wantM: time.February, wantD: 20,
		},
		{
			name:  "date and time",
			input: "2026-02-20 14:30",
			wantY: 2026, wantM: time.February, wantD: 20, wantH: 14, wantMin: 30,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   "not-a-date",
			wantErr: true,
		},
		{
			name:    "ISO format with seconds not supported",
			input:   "2026-02-20T14:30:00",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parseCustomDate(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantY, parsed.Year())
			assert.Equal(t, tt.wantM, parsed.Month())
			assert.Equal(t, tt.wantD, parsed.Day())
			assert.Equal(t, tt.wantH, parsed.Hour())
			assert.Equal(t, tt.wantMin, parsed.Minute())
			assert.Equal(t, time.UTC, parsed.Location())
		})
	}
}

// --- presetDuration ---

func TestPresetDuration(t *testing.T) {
	tests := []struct {
		name   string
		preset bulkDeletePreset
		want   time.Duration
	}{
		{"now", presetNow, 0},
		{"1 day", preset1Day, 24 * time.Hour},
		{"3 days", preset3Days, 3 * 24 * time.Hour},
		{"1 week", preset1Week, 7 * 24 * time.Hour},
		{"2 weeks", preset2Weeks, 14 * 24 * time.Hour},
		{"1 month", preset1Month, 30 * 24 * time.Hour},
		{"custom returns -1", presetCustom, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &bulkDeleteFormResult{preset: tt.preset}
			assert.Equal(t, tt.want, r.presetDuration())
		})
	}
}

// --- toBackupCommand with presets ---

func TestToBackupCommandPreset(t *testing.T) {
	tests := []struct {
		name       string
		result     bulkDeleteFormResult
		wantType   string // "OlderThan" or "Before"
		wantBkType bool   // whether backup type filter is set
	}{
		{
			name: "1 week all types main",
			result: bulkDeleteFormResult{
				target:     bulkDeleteBackups,
				preset:     preset1Week,
				backupType: "all",
				configName: defaultConfigName,
			},
			wantType: "OlderThan",
		},
		{
			name: "now with logical filter",
			result: bulkDeleteFormResult{
				target:     bulkDeleteBackups,
				preset:     presetNow,
				backupType: "logical",
				configName: defaultConfigName,
			},
			wantType:   "OlderThan",
			wantBkType: true,
		},
		{
			name: "custom date",
			result: bulkDeleteFormResult{
				target:     bulkDeleteBackups,
				preset:     presetCustom,
				customDate: "2026-01-15",
				backupType: "all",
				configName: defaultConfigName,
			},
			wantType: "Before",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := tt.result.toBackupCommand()
			require.NoError(t, err)

			switch tt.wantType {
			case "OlderThan":
				c, ok := cmd.(sdk.DeleteBackupsOlderThan)
				require.True(t, ok, "expected DeleteBackupsOlderThan")
				if tt.wantBkType {
					assert.False(t, c.Type.IsZero())
				}
			case "Before":
				c, ok := cmd.(sdk.DeleteBackupsBefore)
				require.True(t, ok, "expected DeleteBackupsBefore")
				assert.False(t, c.OlderThan.IsZero())
			}
		})
	}
}

func TestToBackupCommandCustomDateError(t *testing.T) {
	r := &bulkDeleteFormResult{
		target:     bulkDeleteBackups,
		preset:     presetCustom,
		customDate: "not-a-date",
		backupType: "all",
		configName: defaultConfigName,
	}
	_, err := r.toBackupCommand()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid date")
}

func TestToBackupCommandProfile(t *testing.T) {
	profile, err := sdk.NewConfigName("archive")
	require.NoError(t, err)

	r := &bulkDeleteFormResult{
		target:     bulkDeleteBackups,
		preset:     preset1Day,
		backupType: "all",
		configName: "archive",
	}
	cmd, err := r.toBackupCommand()
	require.NoError(t, err)

	c, ok := cmd.(sdk.DeleteBackupsOlderThan)
	require.True(t, ok)
	assert.True(t, c.ConfigName.Equal(profile))
}

func TestToBackupCommandBackupTypes(t *testing.T) {
	tests := []struct {
		name       string
		backupType string
		wantZero   bool
	}{
		{"all", "all", true},
		{"logical", "logical", false},
		{"physical", "physical", false},
		{"incremental", "incremental", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &bulkDeleteFormResult{
				target:     bulkDeleteBackups,
				preset:     preset1Day,
				backupType: tt.backupType,
				configName: defaultConfigName,
			}
			cmd, err := r.toBackupCommand()
			require.NoError(t, err)

			c, ok := cmd.(sdk.DeleteBackupsOlderThan)
			require.True(t, ok)
			assert.Equal(t, tt.wantZero, c.Type.IsZero(),
				"Type.IsZero() should be %v for %q", tt.wantZero, tt.backupType)
		})
	}
}

// --- toPITRCommand ---

func TestToPITRCommandPreset(t *testing.T) {
	r := &bulkDeleteFormResult{
		target: bulkDeletePITR,
		preset: preset1Week,
	}
	cmd, err := r.toPITRCommand()
	require.NoError(t, err)

	c, ok := cmd.(sdk.DeletePITROlderThan)
	require.True(t, ok)
	assert.Equal(t, 7*24*time.Hour, c.OlderThan)
}

func TestToPITRCommandNow(t *testing.T) {
	r := &bulkDeleteFormResult{
		target: bulkDeletePITR,
		preset: presetNow,
	}
	cmd, err := r.toPITRCommand()
	require.NoError(t, err)

	c, ok := cmd.(sdk.DeletePITROlderThan)
	require.True(t, ok)
	assert.Equal(t, time.Duration(0), c.OlderThan)
}

func TestToPITRCommandCustom(t *testing.T) {
	r := &bulkDeleteFormResult{
		target:     bulkDeletePITR,
		preset:     presetCustom,
		customDate: "2026-01-15",
	}
	cmd, err := r.toPITRCommand()
	require.NoError(t, err)

	c, ok := cmd.(sdk.DeletePITRBefore)
	require.True(t, ok)
	assert.Equal(t, 2026, c.OlderThan.Year())
	assert.Equal(t, time.January, c.OlderThan.Month())
	assert.Equal(t, 15, c.OlderThan.Day())
}

func TestToPITRCommandCustomError(t *testing.T) {
	r := &bulkDeleteFormResult{
		target:     bulkDeletePITR,
		preset:     presetCustom,
		customDate: "bad",
	}
	_, err := r.toPITRCommand()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid date")
}

// --- confirmTitle ---

func TestConfirmTitle(t *testing.T) {
	tests := []struct {
		name   string
		result bulkDeleteFormResult
		want   string
	}{
		{
			name:   "backups 1 week",
			result: bulkDeleteFormResult{target: bulkDeleteBackups, preset: preset1Week},
			want:   "Delete backups older than 1 week?",
		},
		{
			name:   "PITR now",
			result: bulkDeleteFormResult{target: bulkDeletePITR, preset: presetNow},
			want:   "Delete PITR chunks older than now (all)?",
		},
		{
			name:   "backups custom with date",
			result: bulkDeleteFormResult{target: bulkDeleteBackups, preset: presetCustom, customDate: "2026-01-15"},
			want:   "Delete backups older than 2026-01-15?",
		},
		{
			name:   "PITR custom without date",
			result: bulkDeleteFormResult{target: bulkDeletePITR, preset: presetCustom},
			want:   "Delete PITR chunks older than custom date?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.result.confirmTitle())
		})
	}
}

// --- presetLabel ---

func TestPresetLabel(t *testing.T) {
	tests := []struct {
		preset bulkDeletePreset
		custom string
		want   string
	}{
		{presetNow, "", "now (all)"},
		{preset1Day, "", "1 day"},
		{preset3Days, "", "3 days"},
		{preset1Week, "", "1 week"},
		{preset2Weeks, "", "2 weeks"},
		{preset1Month, "", "1 month"},
		{presetCustom, "2026-01-15", "2026-01-15"},
		{presetCustom, "", "custom date"},
	}

	for _, tt := range tests {
		t.Run(string(tt.preset)+"_"+tt.custom, func(t *testing.T) {
			r := &bulkDeleteFormResult{preset: tt.preset, customDate: tt.custom}
			assert.Equal(t, tt.want, r.presetLabel())
		})
	}
}
