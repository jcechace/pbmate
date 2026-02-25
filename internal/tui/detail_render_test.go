package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

func TestStatusIndicator(t *testing.T) {
	styles := testStyles()

	tests := []struct {
		name   string
		status sdk.Status
	}{
		{"done", sdk.StatusDone},
		{"error", sdk.StatusError},
		{"partly done", sdk.StatusPartlyDone},
		{"cancelled", sdk.StatusCancelled},
		{"running", sdk.StatusRunning},
		{"starting", sdk.StatusStarting},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := statusIndicator(tt.status, &styles)
			assert.NotEmpty(t, result)
			assert.Contains(t, result, "●")
		})
	}

	// Verify that all branches produce non-empty, dot-containing output.
	// (Color differentiation is visual and not testable without a real terminal.)
}

func TestAgentIndicator(t *testing.T) {
	styles := testStyles()

	tests := []struct {
		name    string
		agent   sdk.Agent
		wantDot string // "●" for filled, "○" for stale
	}{
		{"healthy", sdk.Agent{OK: true}, "●"},
		{"stale", sdk.Agent{Stale: true, OK: true}, "○"},
		{"errors", sdk.Agent{OK: false, Errors: []string{"err"}}, "●"},
		{"not ok", sdk.Agent{OK: false}, "●"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agentIndicator(&tt.agent, &styles)
			assert.Contains(t, result, tt.wantDot)
		})
	}

	// Healthy uses filled dot, stale uses hollow dot — structurally different.
	healthy := agentIndicator(&sdk.Agent{OK: true}, &styles)
	stale := agentIndicator(&sdk.Agent{Stale: true, OK: true}, &styles)

	assert.NotEqual(t, healthy, stale, "healthy (●) vs stale (○) should differ")
}
