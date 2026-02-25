package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetConfigFormResultIsNew(t *testing.T) {
	tests := []struct {
		name   string
		result setConfigFormResult
		want   bool
	}{
		{
			name:   "new profile",
			result: setConfigFormResult{target: setConfigTargetProfile, profile: setConfigProfileNew},
			want:   true,
		},
		{
			name:   "existing profile",
			result: setConfigFormResult{target: setConfigTargetProfile, profile: "my-s3"},
			want:   false,
		},
		{
			name:   "main target",
			result: setConfigFormResult{target: setConfigTargetMain, profile: setConfigProfileNew},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.result.isNew())
		})
	}
}

func TestSetConfigFormResultEffectiveProfile(t *testing.T) {
	tests := []struct {
		name   string
		result setConfigFormResult
		want   string
	}{
		{
			name:   "main target returns empty",
			result: setConfigFormResult{target: setConfigTargetMain},
			want:   "",
		},
		{
			name:   "existing profile returns name",
			result: setConfigFormResult{target: setConfigTargetProfile, profile: "my-s3"},
			want:   "my-s3",
		},
		{
			name:   "new profile returns newName",
			result: setConfigFormResult{target: setConfigTargetProfile, profile: setConfigProfileNew, newName: "fresh-profile"},
			want:   "fresh-profile",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.result.effectiveProfile())
		})
	}
}
