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
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/cellbuf"
)

// Compile-time interface guard.
var _ huh.Field = (*DateTimePicker)(nil)

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

const defaultThemeIsDark = true

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
	keymap    KeyMap
	theme     huh.Theme
	hasDarkBg bool
	width     int
	height    int
}

// New creates a DateTimePicker with the given initial time.
// The time is normalized to UTC and truncated to second precision.
// Defaults to ModeDateTimeSec.
func New(initial time.Time) *DateTimePicker {
	t := initial.UTC().Truncate(time.Second)
	d := &DateTimePicker{
		accessor:  &huh.EmbeddedAccessor[time.Time]{},
		t:         t,
		mode:      ModeDateTimeSec,
		validate:  func(time.Time) error { return nil },
		keymap:    DefaultKeyMap(),
		hasDarkBg: defaultThemeIsDark,
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

// activeStyles returns focused or blurred FieldStyles from the active theme.
func (d *DateTimePicker) activeStyles() *huh.FieldStyles {
	styles := d.getTheme()
	if styles == nil {
		styles = huh.ThemeCharm(d.hasDarkBg)
	}
	if d.focused {
		return &styles.Focused
	}
	return &styles.Blurred
}

func (d *DateTimePicker) getTheme() *huh.Styles {
	if d.theme == nil {
		return nil
	}
	return d.theme.Theme(d.hasDarkBg)
}

// renderSegments renders the segmented date/time value string.
// The active segment (when focused) is highlighted using theme cursor styles.
func (d *DateTimePicker) renderSegments() string {
	styles := d.activeStyles()
	textStyle := styles.TextInput.Text
	activeStyle := lipgloss.NewStyle().
		Foreground(styles.FocusedButton.GetForeground()).
		Background(styles.FocusedButton.GetBackground()).
		Bold(true)
	sepStyle := styles.TextInput.Placeholder

	n := segCount(d.mode)
	var sb strings.Builder

	for i := 0; i < n; i++ {
		s := segment(i)
		val := d.segValue(s)

		// Show digit buffer preview for the active focused segment.
		var text string
		if d.focused && s == d.activeSeg && d.digitBuf != "" {
			w := segWidth(s)
			text = fmt.Sprintf("%0*s", w, d.digitBuf)
		} else {
			text = formatSegment(s, val)
		}

		if d.focused && s == d.activeSeg {
			sb.WriteString(activeStyle.Render(text))
		} else {
			sb.WriteString(textStyle.Render(text))
		}

		// Separators between segments.
		switch s {
		case segYear, segMonth:
			sb.WriteString(sepStyle.Render("-"))
		case segDay:
			if n > 3 {
				sb.WriteString(sepStyle.Render("  "))
			}
		case segHour:
			sb.WriteString(sepStyle.Render(":"))
		case segMinute:
			if n > 5 {
				sb.WriteString(sepStyle.Render(":"))
			}
		}
	}

	return sb.String()
}

// wrapText wraps s to limit display columns.
func wrapText(s string, limit int) string {
	return cellbuf.Wrap(s, limit, ",.-; ")
}

// =============================================================================
// huh.Field interface
// =============================================================================

// Init implements huh.Field.
func (d *DateTimePicker) Init() tea.Cmd { return nil }

// Update implements huh.Field.
func (d *DateTimePicker) Update(msg tea.Msg) (huh.Model, tea.Cmd) {
	if bgMsg, ok := msg.(tea.BackgroundColorMsg); ok {
		d.hasDarkBg = bgMsg.IsDark()
		return d, nil
	}

	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return d, nil
	}

	// Clear validation error on any key press.
	d.err = nil

	switch {
	case key.Matches(keyMsg, d.keymap.Prev):
		d.flushDigitBuf()
		d.accessor.Set(d.t)
		return d, huh.PrevField

	case key.Matches(keyMsg, d.keymap.Next):
		d.flushDigitBuf()
		d.accessor.Set(d.t)
		d.err = d.validate(d.t)
		if d.err != nil {
			return d, nil
		}
		return d, huh.NextField

	case key.Matches(keyMsg, d.keymap.Submit):
		d.flushDigitBuf()
		d.accessor.Set(d.t)
		d.err = d.validate(d.t)
		if d.err != nil {
			return d, nil
		}
		return d, huh.NextField

	case key.Matches(keyMsg, d.keymap.Left):
		d.prevSeg()

	case key.Matches(keyMsg, d.keymap.Right):
		d.nextSeg()

	case key.Matches(keyMsg, d.keymap.Up):
		d.flushDigitBuf()
		d.t = setSegValue(d.t, d.activeSeg, d.segValue(d.activeSeg)+1)

	case key.Matches(keyMsg, d.keymap.Down):
		d.flushDigitBuf()
		d.t = setSegValue(d.t, d.activeSeg, d.segValue(d.activeSeg)-1)

	default:
		// Digit input.
		if len(keyMsg.Text) == 1 {
			ch := keyMsg.Text[0]
			if ch >= '0' && ch <= '9' {
				d.addDigit(byte(ch))
			}
		}
	}

	return d, nil
}

// View implements huh.Field.
func (d *DateTimePicker) View() string {
	styles := d.activeStyles()
	maxWidth := d.width - styles.Base.GetHorizontalFrameSize()

	var sb strings.Builder

	if d.title != "" {
		sb.WriteString(styles.Title.Render(wrapText(d.title, maxWidth)))
		sb.WriteString("\n")
	}
	if d.desc != "" {
		sb.WriteString(styles.Description.Render(wrapText(d.desc, maxWidth)))
		sb.WriteString("\n")
	}

	sb.WriteString(d.renderSegments())

	if d.err != nil {
		sb.WriteString("\n")
		sb.WriteString(styles.ErrorMessage.Render(d.err.Error()))
	}

	return styles.Base.
		Width(d.width).
		Height(d.height).
		Render(sb.String())
}

// Focus implements huh.Field.
func (d *DateTimePicker) Focus() tea.Cmd {
	d.focused = true
	return nil
}

// Blur implements huh.Field.
func (d *DateTimePicker) Blur() tea.Cmd {
	d.focused = false
	d.flushDigitBuf()
	d.accessor.Set(d.t)
	d.err = d.validate(d.t)
	return nil
}

// Error implements huh.Field.
func (d *DateTimePicker) Error() error { return d.err }

// Skip implements huh.Field.
func (*DateTimePicker) Skip() bool { return false }

// Zoom implements huh.Field.
func (*DateTimePicker) Zoom() bool { return false }

// KeyBinds implements huh.Field.
func (d *DateTimePicker) KeyBinds() []key.Binding {
	return []key.Binding{
		d.keymap.Left,
		d.keymap.Right,
		d.keymap.Up,
		d.keymap.Down,
		d.keymap.Prev,
		d.keymap.Submit,
		d.keymap.Next,
	}
}

// Run implements huh.Field.
func (d *DateTimePicker) Run() error {
	return huh.Run(d) //nolint:wrapcheck
}

// RunAccessible implements huh.Field.
// Prompts the user to enter a date/time string in the format matching the mode.
func (d *DateTimePicker) RunAccessible(w io.Writer, r io.Reader) error {
	styles := d.activeStyles()

	formats := map[Mode]string{
		ModeDate:        "2006-01-02",
		ModeDateTime:    "2006-01-02 15:04",
		ModeDateTimeSec: "2006-01-02 15:04:05",
	}
	layout := formats[d.mode]

	prompt := styles.Title.PaddingRight(1).Render(d.title)
	if d.title == "" {
		prompt = "Date/time: "
	}
	_, _ = fmt.Fprintf(w, "%s (format: %s): ", prompt, layout)

	scanner := bufio.NewScanner(r)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())

	t, err := time.ParseInLocation(layout, input, time.UTC)
	if err != nil {
		return fmt.Errorf("invalid date/time %q (expected %s): %w", input, layout, err)
	}
	if err := d.validate(t); err != nil {
		return err
	}
	d.t = t
	d.accessor.Set(t)
	return nil
}

// WithTheme implements huh.Field.
func (d *DateTimePicker) WithTheme(theme huh.Theme) huh.Field {
	if d.theme != nil || theme == nil {
		return d
	}
	d.theme = theme
	return d
}

// WithAccessible implements huh.Field.
//
// Deprecated: call [DateTimePicker.RunAccessible] directly.
func (d *DateTimePicker) WithAccessible(_ bool) huh.Field { return d }

// WithKeyMap implements huh.Field.
// Maps Next, Prev, and Submit from the huh form keymap.
// Left, Right, Up, and Down retain their [DefaultKeyMap] bindings since
// [huh.KeyMap] has no corresponding navigation fields for them.
func (d *DateTimePicker) WithKeyMap(k *huh.KeyMap) huh.Field {
	d.keymap.Next = k.Input.Next
	d.keymap.Prev = k.Input.Prev
	d.keymap.Submit = k.Input.Submit
	return d
}

// WithWidth implements huh.Field.
func (d *DateTimePicker) WithWidth(width int) huh.Field {
	d.width = width
	return d
}

// WithHeight implements huh.Field.
func (d *DateTimePicker) WithHeight(height int) huh.Field {
	d.height = height
	return d
}

// WithPosition implements huh.Field.
func (d *DateTimePicker) WithPosition(p huh.FieldPosition) huh.Field {
	d.keymap.Prev.SetEnabled(!p.IsFirst())
	d.keymap.Next.SetEnabled(!p.IsLast())
	d.keymap.Submit.SetEnabled(p.IsLast())
	return d
}

// GetKey implements huh.Field.
func (d *DateTimePicker) GetKey() string { return d.fieldKey }

// GetValue implements huh.Field.
func (d *DateTimePicker) GetValue() any { return d.accessor.Get() }
