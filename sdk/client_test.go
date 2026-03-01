package sdk

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientNoBackend(t *testing.T) {
	_, err := NewClient(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no connection backend configured")
}
