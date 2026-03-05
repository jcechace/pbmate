// Package datefield provides a date/time picker field for huh forms.
// It implements the [huh.Field] interface, making it a drop-in addition
// alongside standard huh fields (Input, Select, Confirm, etc.).
//
// The picker displays a segmented date/time value where each component
// (year, month, day, hour, minute, second) is individually editable.
// Navigation uses arrow keys; values are changed with up/down or digit input.
// Sub-second precision is not supported — all times are truncated to seconds.
//
// # Time zones
//
// All times are normalized to UTC on input. The value written to the bound
// pointer on blur or submit is always in UTC. Domain constraints such as
// valid date ranges belong in the caller's [DateTimePicker.Validate] function.
//
// # Modes
//
// Three modes control which segments are shown: [ModeDate], [ModeDateTime],
// and [ModeDateTimeSec].
package datefield
