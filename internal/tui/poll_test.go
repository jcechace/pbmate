package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConnectBackoff(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
		{5, 30 * time.Second},  // capped
		{6, 30 * time.Second},  // stays capped
		{10, 30 * time.Second}, // stays capped
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tt.want, connectBackoff(tt.attempt))
		})
	}
}
