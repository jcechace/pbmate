// Package tui implements the BubbleTea terminal UI for PBMate.
//
// # Architecture
//
// The UI follows the Elm Architecture (Model-Update-View) via BubbleTea.
// A single root [Model] in app.go owns three tab sub-models (overview,
// backups, config) and routes messages to the active tab. Sub-models are
// plain structs (not tea.Model) with update, view, resize, and setData
// methods.
//
// # Data Flow
//
// The polling loop is a self-healing timer chain:
//
//	Init -> connectCmd -> connectMsg (client ready)
//	                          |
//	                     fetchCmd -> dataMsg (fresh data)
//	                          |          |
//	                     setData()   tickCmd (schedule next poll)
//	                                     |
//	                                fetchCmd -> ...
//
// Each fetch is a single-shot command. There are no persistent goroutine
// tickers. If a fetch fails, the tick still fires and retries on the next
// cycle. Action results (backup, restore, resync) trigger an immediate
// fetch to refresh data.
//
// # Overlays
//
// Modal form overlays implement the [formOverlay] interface and capture
// all input while active. Overlays can chain: the set-config form
// transitions to a file picker, which may transition to a confirm dialog.
// Each overlay type lives in its own file (*_overlay.go).
//
// # File Organization
//
//   - app.go         — root Model, Init, Update routing, View
//   - data.go        — fetch commands, action commands, message types
//   - keys.go        — keybinding definitions
//   - layout.go      — panel geometry, scroll helper
//   - styles.go      — lipgloss styles and theme
//   - render.go      — shared rendering helpers (cursor list, help overlay)
//   - overlay.go     — formOverlay interface
//   - *_overlay.go   — concrete overlay implementations
//   - *_form.go      — huh form constructors and result types
//   - overview.go    — Overview tab sub-model
//   - backups.go     — Backups tab sub-model
//   - config.go      — Config tab sub-model
package tui
