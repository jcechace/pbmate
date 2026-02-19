package tui

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

func TestFormatStorageSummary(t *testing.T) {
	tests := []struct {
		name     string
		input    sdk.StorageConfig
		expected string
	}{
		{
			name:     "zero type returns empty",
			input:    sdk.StorageConfig{},
			expected: "",
		},
		{
			name:     "type with path",
			input:    sdk.StorageConfig{Type: sdk.StorageTypeS3, Path: "my-bucket/backups"},
			expected: "s3 my-bucket/backups",
		},
		{
			name:     "type without path",
			input:    sdk.StorageConfig{Type: sdk.StorageTypeS3},
			expected: "s3",
		},
		{
			name:     "filesystem with path",
			input:    sdk.StorageConfig{Type: sdk.StorageTypeFilesystem, Path: "/data/backups"},
			expected: "filesystem /data/backups",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatStorageSummary(tt.input))
		})
	}
}

func TestDrainErr(t *testing.T) {
	t.Run("nil channel returns nil", func(t *testing.T) {
		assert.NoError(t, drainErr(nil))
	})

	t.Run("empty channel returns nil", func(t *testing.T) {
		ch := make(chan error, 1)
		assert.NoError(t, drainErr(ch))
	})

	t.Run("buffered error is returned", func(t *testing.T) {
		ch := make(chan error, 1)
		ch <- errors.New("connection lost")
		assert.EqualError(t, drainErr(ch), "connection lost")
	})

	t.Run("closed empty channel returns nil", func(t *testing.T) {
		ch := make(chan error)
		close(ch)
		assert.NoError(t, drainErr(ch))
	})
}
