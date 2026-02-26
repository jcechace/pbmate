package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// --- toLogFilter ---

func TestToLogFilter(t *testing.T) {
	tests := []struct {
		name    string
		result  logFilterFormResult
		wantSev sdk.LogSeverity
		wantRS  string
		wantEvt string
	}{
		{
			name:    "defaults",
			result:  logFilterFormResult{severity: "I", replicaSet: logFilterAll, event: logFilterAll},
			wantSev: sdk.LogSeverity{}, // zero = Info default in SDK
		},
		{
			name:    "debug severity",
			result:  logFilterFormResult{severity: "D"},
			wantSev: sdk.LogSeverityDebug,
		},
		{
			name:    "warning severity",
			result:  logFilterFormResult{severity: "W"},
			wantSev: sdk.LogSeverityWarning,
		},
		{
			name:    "error severity",
			result:  logFilterFormResult{severity: "E"},
			wantSev: sdk.LogSeverityError,
		},
		{
			name:    "fatal severity",
			result:  logFilterFormResult{severity: "F"},
			wantSev: sdk.LogSeverityFatal,
		},
		{
			name:   "with replica set",
			result: logFilterFormResult{severity: "I", replicaSet: "rs0"},
			wantRS: "rs0",
		},
		{
			name:    "with event",
			result:  logFilterFormResult{severity: "I", event: "backup"},
			wantEvt: "backup",
		},
		{
			name:    "all filters set",
			result:  logFilterFormResult{severity: "W", replicaSet: "rs1", event: "restore"},
			wantSev: sdk.LogSeverityWarning,
			wantRS:  "rs1",
			wantEvt: "restore",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.result.toLogFilter()
			assert.Equal(t, tt.wantSev, f.Severity)
			assert.Equal(t, tt.wantRS, f.ReplicaSet)
			assert.Equal(t, tt.wantEvt, f.Event)
		})
	}
}

// --- fromLogFilter ---

func TestFromLogFilter(t *testing.T) {
	tests := []struct {
		name    string
		filter  sdk.LogFilter
		wantSev string
		wantRS  string
		wantEvt string
	}{
		{
			name:    "zero filter",
			filter:  sdk.LogFilter{},
			wantSev: "I", wantRS: logFilterAll, wantEvt: logFilterAll,
		},
		{
			name:    "debug",
			filter:  sdk.LogFilter{Severity: sdk.LogSeverityDebug},
			wantSev: "D", wantRS: logFilterAll, wantEvt: logFilterAll,
		},
		{
			name:    "warning with RS and event",
			filter:  sdk.LogFilter{Severity: sdk.LogSeverityWarning, ReplicaSet: "rs0", Event: "pitr"},
			wantSev: "W", wantRS: "rs0", wantEvt: "pitr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := fromLogFilter(tt.filter)
			assert.Equal(t, tt.wantSev, r.severity)
			assert.Equal(t, tt.wantRS, r.replicaSet)
			assert.Equal(t, tt.wantEvt, r.event)
			assert.True(t, r.confirmed)
		})
	}
}

// --- roundtrip: fromLogFilter -> toLogFilter ---

func TestLogFilterRoundtrip(t *testing.T) {
	original := sdk.LogFilter{
		Severity:   sdk.LogSeverityError,
		ReplicaSet: "rs0",
		Event:      "backup",
	}
	r := fromLogFilter(original)
	roundtripped := r.toLogFilter()
	assert.Equal(t, original.Severity, roundtripped.Severity)
	assert.Equal(t, original.ReplicaSet, roundtripped.ReplicaSet)
	assert.Equal(t, original.Event, roundtripped.Event)
}

// --- logFilterTitle ---

func TestLogFilterTitle(t *testing.T) {
	tests := []struct {
		name   string
		filter sdk.LogFilter
		want   string
	}{
		{
			name:   "no filters",
			filter: sdk.LogFilter{},
			want:   "Logs",
		},
		{
			name:   "info severity is default",
			filter: sdk.LogFilter{Severity: sdk.LogSeverityInfo},
			want:   "Logs",
		},
		{
			name:   "warning severity",
			filter: sdk.LogFilter{Severity: sdk.LogSeverityWarning},
			want:   "Logs (W)",
		},
		{
			name:   "RS only",
			filter: sdk.LogFilter{ReplicaSet: "rs0"},
			want:   "Logs (rs0)",
		},
		{
			name:   "event only",
			filter: sdk.LogFilter{Event: "backup"},
			want:   "Logs (backup)",
		},
		{
			name:   "all filters",
			filter: sdk.LogFilter{Severity: sdk.LogSeverityDebug, ReplicaSet: "rs1", Event: "pitr"},
			want:   "Logs (D, rs1, pitr)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, logFilterTitle(tt.filter))
		})
	}
}

// --- uniqueReplicaSets ---

func TestUniqueReplicaSets(t *testing.T) {
	tests := []struct {
		name   string
		agents []sdk.Agent
		want   []string
	}{
		{
			name:   "empty",
			agents: nil,
			want:   nil,
		},
		{
			name: "single RS",
			agents: []sdk.Agent{
				{ReplicaSet: "rs0", Node: "host1:27017"},
				{ReplicaSet: "rs0", Node: "host2:27017"},
			},
			want: []string{"rs0"},
		},
		{
			name: "multiple RS preserves order",
			agents: []sdk.Agent{
				{ReplicaSet: "rs0"},
				{ReplicaSet: "rs1"},
				{ReplicaSet: "rs0"},
				{ReplicaSet: "cfg"},
			},
			want: []string{"rs0", "rs1", "cfg"},
		},
		{
			name: "empty RS skipped",
			agents: []sdk.Agent{
				{ReplicaSet: ""},
				{ReplicaSet: "rs0"},
			},
			want: []string{"rs0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uniqueReplicaSets(tt.agents)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- formatLogSource ---

func TestFormatLogSource(t *testing.T) {
	tests := []struct {
		name  string
		attrs map[string]any
		want  string
	}{
		{
			name:  "empty attrs",
			attrs: map[string]any{},
			want:  "",
		},
		{
			name:  "RS only",
			attrs: map[string]any{sdk.LogKeyReplicaSet: "rs0"},
			want:  "rs0",
		},
		{
			name:  "RS and node",
			attrs: map[string]any{sdk.LogKeyReplicaSet: "rs0", sdk.LogKeyNode: "host1:27017"},
			want:  "rs0/27017",
		},
		{
			name:  "node without RS",
			attrs: map[string]any{sdk.LogKeyNode: "host1:27017"},
			want:  "",
		},
		{
			name:  "node without port",
			attrs: map[string]any{sdk.LogKeyReplicaSet: "rs0", sdk.LogKeyNode: "host1"},
			want:  "rs0/host1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatLogSource(tt.attrs))
		})
	}
}
