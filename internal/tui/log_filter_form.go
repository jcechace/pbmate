package tui

import (
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// logFilterAll is the sentinel value for "All" in filter selectors.
const logFilterAll = ""

// logFilterFormResult holds the user's selections from the log filter form.
type logFilterFormResult struct {
	severity   string // "D", "I", "W", "E", "F"
	replicaSet string // logFilterAll or a specific RS name
	event      string // logFilterAll or a specific event type
	confirmed  bool   // true = apply, false = reset
}

// toLogFilter converts the form result into an SDK LogFilter.
func (r *logFilterFormResult) toLogFilter() sdk.LogFilter {
	var f sdk.LogFilter

	switch r.severity {
	case "D":
		f.Severity = sdk.LogSeverityDebug
	case "W":
		f.Severity = sdk.LogSeverityWarning
	case "E":
		f.Severity = sdk.LogSeverityError
	case "F":
		f.Severity = sdk.LogSeverityFatal
	default:
		// "I" or anything else: zero value defaults to Info in the SDK.
	}

	f.ReplicaSet = r.replicaSet
	f.Event = r.event
	return f
}

// logFilterTitle returns a title string for the log panel that includes
// active filter indicators. Returns "Logs" when no filters are active
// (severity=Info, no RS, no event).
func logFilterTitle(f sdk.LogFilter) string {
	var parts []string

	// Severity: only show if not the default (Info / zero).
	if !f.Severity.IsZero() && !f.Severity.Equal(sdk.LogSeverityInfo) {
		parts = append(parts, f.Severity.String()[:1])
	}

	if f.ReplicaSet != "" {
		parts = append(parts, f.ReplicaSet)
	}

	if f.Event != "" {
		parts = append(parts, f.Event)
	}

	if len(parts) == 0 {
		return "Logs"
	}
	return "Logs (" + strings.Join(parts, ", ") + ")"
}

// fromLogFilter converts an SDK LogFilter into form result values for
// pre-populating the form when re-opening it.
func fromLogFilter(f sdk.LogFilter) *logFilterFormResult {
	r := &logFilterFormResult{
		severity:   "I",
		replicaSet: logFilterAll,
		event:      logFilterAll,
		confirmed:  true,
	}

	switch {
	case f.Severity.Equal(sdk.LogSeverityDebug):
		r.severity = "D"
	case f.Severity.Equal(sdk.LogSeverityWarning):
		r.severity = "W"
	case f.Severity.Equal(sdk.LogSeverityError):
		r.severity = "E"
	case f.Severity.Equal(sdk.LogSeverityFatal):
		r.severity = "F"
	}

	r.replicaSet = f.ReplicaSet
	r.event = f.Event
	return r
}

// uniqueReplicaSets extracts sorted unique RS names from a list of agents.
func uniqueReplicaSets(agents []sdk.Agent) []string {
	seen := make(map[string]bool)
	var result []string
	for _, a := range agents {
		if a.ReplicaSet != "" && !seen[a.ReplicaSet] {
			seen[a.ReplicaSet] = true
			result = append(result, a.ReplicaSet)
		}
	}
	return result
}

// newLogFilterForm creates a form for configuring log filters.
// The agents list is used to populate the replica set selector.
// initial carries the current filter state for pre-population (nil for defaults).
func newLogFilterForm(formTheme *huh.Theme, agents []sdk.Agent, initial *logFilterFormResult) (*huh.Form, *logFilterFormResult) {
	result := &logFilterFormResult{
		severity:   "I",
		replicaSet: logFilterAll,
		event:      logFilterAll,
		confirmed:  true,
	}
	if initial != nil {
		result.severity = initial.severity
		result.replicaSet = initial.replicaSet
		result.event = initial.event
	}

	// RS options: All + each unique RS from agents.
	rsOpts := []huh.Option[string]{
		huh.NewOption("All", logFilterAll),
	}
	for _, rs := range uniqueReplicaSets(agents) {
		rsOpts = append(rsOpts, huh.NewOption(rs, rs))
	}

	groups := []*huh.Group{
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Severity").
				Options(
					huh.NewOption("Debug", "D"),
					huh.NewOption("Info", "I"),
					huh.NewOption("Warning", "W"),
					huh.NewOption("Error", "E"),
					huh.NewOption("Fatal", "F"),
				).
				Inline(true).
				Value(&result.severity),

			huh.NewSelect[string]().
				Title("Replica set").
				Options(rsOpts...).
				Inline(true).
				Value(&result.replicaSet),

			huh.NewSelect[string]().
				Title("Event").
				Options(
					huh.NewOption("All", logFilterAll),
					huh.NewOption("backup", "backup"),
					huh.NewOption("restore", "restore"),
					huh.NewOption("cancelBackup", "cancelBackup"),
					huh.NewOption("resync", "resync"),
					huh.NewOption("pitr", "pitr"),
					huh.NewOption("delete", "delete"),
				).
				Inline(true).
				Value(&result.event),
		),
	}

	// Confirm: Apply or Reset.
	groups = append(groups, huh.NewGroup(
		huh.NewConfirm().
			Title("Apply log filter?").
			WithButtonAlignment(lipgloss.Left).
			Affirmative("Apply").
			Negative("Reset").
			Value(&result.confirmed),
	))

	form := newStandardForm(groups, formTheme)
	return form, result
}
