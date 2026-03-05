// Package datefield provides a date/time picker field for huh forms.
// It implements the [huh.Field] interface, making it a drop-in addition
// alongside standard huh fields (Input, Select, Confirm, etc.).
//
// The picker displays a segmented date/time value where each component
// (year, month, day, hour, minute, second) is individually editable:
//
//	2025-03-05  14:30:00
//	          ^^
//
// Navigation uses arrow keys; values are changed with up/down or digit input.
// Three modes are available controlling which segments are shown:
//
//	datefield.New(time.Now()).
//	    Title("Restore point").
//	    Mode(datefield.ModeDateTimeSec).
//	    Value(&myTime)
package datefield

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
)

// =============================================================================
// Mode
// =============================================================================

// Mode controls which segments are shown by the picker.
type Mode int

const (
	// ModeDate shows year, month, and day only: 2025-03-05
	ModeDate Mode = iota
	// ModeDateTime shows date plus hour and minute: 2025-03-05 14:30
	ModeDateTime
	// ModeDateTimeSec shows full date and time with seconds: 2025-03-05 14:30:00
	ModeDateTimeSec
)

// =============================================================================
// Segment
// =============================================================================

// segment identifies a single unit within the date/time value.
type segment int

const (
	segYear segment = iota
	segMonth
	segDay
	segHour
	segMinute
	segSecond
)

// segCount returns the number of active segments for a given mode.
func segCount(m Mode) int {
	switch m {
	case ModeDate:
		return 3 // year, month, day
	case ModeDateTime:
		return 5 // year, month, day, hour, minute
	default:
		return 6 // year, month, day, hour, minute, second
	}
}

// segWidth returns the number of digit characters for a segment.
func segWidth(s segment) int {
	if s == segYear {
		return 4
	}
	return 2
}

// =============================================================================
// KeyMap
// =============================================================================

// KeyMap holds the key bindings for the DateTimePicker field.
type KeyMap struct {
	Next   key.Binding
	Prev   key.Binding
	Submit key.Binding
	Left   key.Binding
	Right  key.Binding
	Up     key.Binding
	Down   key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Next:   key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field")),
		Prev:   key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev field")),
		Submit: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
		Left:   key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "prev segment")),
		Right:  key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "next segment")),
		Up:     key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "increment")),
		Down:   key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "decrement")),
	}
}

// =============================================================================
// DateTimePicker
// =============================================================================

// DateTimePicker is a huh form field for selecting a date and/or time value.
// It renders a segmented display where each component is individually editable.
//
// Create with [New] and configure with the fluent builder methods.
type DateTimePicker struct {
	// Value accessor — the external *time.Time the field reads/writes.
	accessor huh.Accessor[time.Time]

	// Working copy of the current time, kept in sync with the accessor.
	t time.Time

	// Configuration
	fieldKey string
	title    string
	desc     string
	mode     Mode
	validate func(time.Time) error
	err      error

	// Interaction state
	focused   bool
	activeSeg segment // which segment currently has cursor focus
	digitBuf  string  // accumulated digit input for the current segment

	// huh integration
	keymap   KeyMap
	theme    *huh.Theme
	position huh.FieldPosition
	width    int
	height   int
}

// New creates a DateTimePicker with the given initial time.
// Defaults to ModeDateTimeSec and UTC time.
func New(initial time.Time) *DateTimePicker {
	t := initial.UTC()
	d := &DateTimePicker{
		accessor: &huh.EmbeddedAccessor[time.Time]{},
		t:        t,
		mode:     ModeDateTimeSec,
		validate: func(time.Time) error { return nil },
		keymap:   DefaultKeyMap(),
	}
	d.accessor.Set(t)
	return d
}

// =============================================================================
// Builder methods
// =============================================================================

// Title sets the field title displayed above the picker.
func (d *DateTimePicker) Title(title string) *DateTimePicker {
	d.title = title
	return d
}

// Description sets the optional description displayed below the title.
func (d *DateTimePicker) Description(desc string) *DateTimePicker {
	d.desc = desc
	return d
}

// Mode sets the display mode (date only, date+time, date+time+seconds).
func (d *DateTimePicker) Mode(m Mode) *DateTimePicker {
	d.mode = m
	// Clamp active segment to valid range for the new mode.
	if int(d.activeSeg) >= segCount(m) {
		d.activeSeg = segment(segCount(m) - 1)
	}
	return d
}

// Value sets the pointer that the field reads its initial value from and writes
// the result to on submit/blur.
func (d *DateTimePicker) Value(v *time.Time) *DateTimePicker {
	d.accessor = huh.NewPointerAccessor(v)
	if !v.IsZero() {
		d.t = v.UTC()
	}
	return d
}

// Key sets the field key used for form value lookup.
func (d *DateTimePicker) Key(k string) *DateTimePicker {
	d.fieldKey = k
	return d
}

// Validate sets the validation function called on blur and submit.
func (d *DateTimePicker) Validate(fn func(time.Time) error) *DateTimePicker {
	d.validate = fn
	return d
}

// =============================================================================
// Segment value helpers
// =============================================================================

// segValue returns the current integer value of a segment.
func (d *DateTimePicker) segValue(s segment) int {
	switch s {
	case segYear:
		return d.t.Year()
	case segMonth:
		return int(d.t.Month())
	case segDay:
		return d.t.Day()
	case segHour:
		return d.t.Hour()
	case segMinute:
		return d.t.Minute()
	default:
		return d.t.Second()
	}
}

// setSegValue returns a new time.Time with the given segment set to v.
// Cyclic segments (month, day, hour, minute, second) wrap at their boundaries.
// Year is unbounded — it increments and decrements freely; domain constraints
// belong in the caller's Validate function.
func setSegValue(t time.Time, s segment, v int) time.Time {
	yr, mo, dy := t.Date()
	hr, mn, sc := t.Clock()

	switch s {
	case segYear:
		yr = v
	case segMonth:
		mo = time.Month(wrapInt(v, 1, 12))
	case segDay:
		dy = wrapInt(v, 1, daysInMonth(yr, mo))
	case segHour:
		hr = wrapInt(v, 0, 23)
	case segMinute:
		mn = wrapInt(v, 0, 59)
	case segSecond:
		sc = wrapInt(v, 0, 59)
	}

	// After changing month or year the day may exceed the new month's max.
	if s == segMonth || s == segYear {
		maxDay := daysInMonth(yr, mo)
		if dy > maxDay {
			dy = maxDay
		}
	}

	return time.Date(yr, mo, dy, hr, mn, sc, 0, time.UTC)
}

// daysInMonth returns the number of days in the given month/year.
func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// wrapInt wraps v within [min, max] inclusive.
func wrapInt(v, min, max int) int {
	r := max - min + 1
	v = (v-min)%r + min
	if v < min {
		v += r
	}
	return v
}

// =============================================================================
// Segment navigation
// =============================================================================

// prevSeg moves focus to the previous segment. Flushes any pending digit input.
// No-op when already at the first segment.
func (d *DateTimePicker) prevSeg() {
	d.flushDigitBuf()
	if d.activeSeg > segYear {
		d.activeSeg--
	}
}

// nextSeg moves focus to the next segment. Flushes any pending digit input.
// No-op when already at the last segment for the current mode.
func (d *DateTimePicker) nextSeg() {
	d.flushDigitBuf()
	last := segment(segCount(d.mode) - 1)
	if d.activeSeg < last {
		d.activeSeg++
	}
}

// =============================================================================
// Digit input
// =============================================================================

// addDigit appends a typed digit to the buffer and applies it to the active
// segment. When the buffer fills the segment's full width, focus auto-advances
// to the next segment.
func (d *DateTimePicker) addDigit(ch byte) {
	maxLen := segWidth(d.activeSeg)
	d.digitBuf += string(ch)

	var v int
	if _, err := fmt.Sscanf(d.digitBuf, "%d", &v); err == nil {
		d.t = setSegValue(d.t, d.activeSeg, v)
	}

	if len(d.digitBuf) >= maxLen {
		d.digitBuf = ""
		d.nextSeg()
	}
}

// flushDigitBuf clears any pending digit buffer without applying further changes.
func (d *DateTimePicker) flushDigitBuf() {
	d.digitBuf = ""
}

// =============================================================================
// Rendering helpers
// =============================================================================

// formatSegment returns the zero-padded string for a segment value.
func formatSegment(s segment, v int) string {
	if s == segYear {
		return fmt.Sprintf("%04d", v)
	}
	return fmt.Sprintf("%02d", v)
}
