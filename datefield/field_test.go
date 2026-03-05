package datefield

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Helpers
// =============================================================================

// keyMsg constructs a tea.KeyMsg for a named key (e.g. "left", "up", "tab").
func keyMsg(key string) tea.KeyMsg {
	switch key {
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	default:
		// Rune key.
		r := []rune(key)
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: r}
	}
}

// execCmd runs a tea.Cmd and returns its Msg. Returns nil if cmd is nil.
func execCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

// sendKey calls Update with a key message and returns the updated picker and cmd msg.
func sendKey(d *DateTimePicker, key string) (*DateTimePicker, tea.Msg) {
	model, cmd := d.Update(keyMsg(key))
	d = model.(*DateTimePicker)
	return d, execCmd(cmd)
}

// =============================================================================
// Init
// =============================================================================

func TestInit(t *testing.T) {
	d := New(time.Now())
	cmd := d.Init()
	assert.Nil(t, cmd)
}

// =============================================================================
// Focus / Blur
// =============================================================================

func TestFocus(t *testing.T) {
	d := New(time.Now())
	assert.False(t, d.focused)
	cmd := d.Focus()
	assert.True(t, d.focused)
	assert.Nil(t, cmd)
}

func TestBlurWritesAccessor(t *testing.T) {
	var v time.Time
	now := time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC)
	d := New(now).Value(&v)
	d.Focus()

	cmd := d.Blur()
	assert.False(t, d.focused)
	assert.Nil(t, cmd)
	assert.Equal(t, now, v, "Blur should write current time to the bound pointer")
}

func TestBlurRunsValidation(t *testing.T) {
	sentinel := errors.New("bad time")
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	d := New(now).Validate(func(time.Time) error { return sentinel })
	d.Focus()
	d.Blur()
	assert.Equal(t, sentinel, d.err)
}

func TestBlurFlushesDigitBuf(t *testing.T) {
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	d := New(now)
	d.digitBuf = "12"
	d.Blur()
	assert.Empty(t, d.digitBuf)
}

// =============================================================================
// Update — non-key messages ignored
// =============================================================================

func TestUpdateIgnoresNonKeyMsg(t *testing.T) {
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	d := New(now)
	model, cmd := d.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	assert.Same(t, d, model.(*DateTimePicker))
	assert.Nil(t, cmd)
}

// =============================================================================
// Update — segment navigation
// =============================================================================

func TestUpdateLeft(t *testing.T) {
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	d := New(now)
	d.activeSeg = segMonth

	d, msg := sendKey(d, "left")
	assert.Equal(t, segYear, d.activeSeg)
	assert.Nil(t, msg)
}

func TestUpdateRight(t *testing.T) {
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	d := New(now)
	d.activeSeg = segYear

	d, msg := sendKey(d, "right")
	assert.Equal(t, segMonth, d.activeSeg)
	assert.Nil(t, msg)
}

// =============================================================================
// Update — up / down
// =============================================================================

func TestUpdateUpIncrementsActiveSeg(t *testing.T) {
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	d := New(now)
	d.activeSeg = segMonth

	d, msg := sendKey(d, "up")
	assert.Equal(t, 7, int(d.t.Month()))
	assert.Nil(t, msg)
}

func TestUpdateDownDecrementsActiveSeg(t *testing.T) {
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	d := New(now)
	d.activeSeg = segMonth

	d, msg := sendKey(d, "down")
	assert.Equal(t, 5, int(d.t.Month()))
	assert.Nil(t, msg)
}

func TestUpdateUpFlushesDigitBuf(t *testing.T) {
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	d := New(now)
	d.digitBuf = "1"

	d, _ = sendKey(d, "up")
	assert.Empty(t, d.digitBuf)
}

// =============================================================================
// Update — digit input
// =============================================================================

func TestUpdateDigitInput(t *testing.T) {
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	d := New(now)
	d.activeSeg = segDay

	d, _ = sendKey(d, "2")
	assert.Equal(t, "2", d.digitBuf)
	assert.Equal(t, segDay, d.activeSeg) // not yet advanced

	d, _ = sendKey(d, "8")
	assert.Empty(t, d.digitBuf)           // flushed after full width
	assert.Equal(t, segHour, d.activeSeg) // auto-advanced
	assert.Equal(t, 28, d.t.Day())
}

func TestUpdateNonDigitRune(t *testing.T) {
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	d := New(now)
	before := d.t

	d, _ = sendKey(d, "x")
	assert.Equal(t, before, d.t, "non-digit rune should not change time")
}

// =============================================================================
// Update — field navigation (tab / enter / shift+tab)
// =============================================================================

func TestUpdateTabEmitsNextField(t *testing.T) {
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	d := New(now)
	d.keymap.Next.SetEnabled(true)

	_, msg := sendKey(d, "tab")
	require.NotNil(t, msg, "tab should emit NextField cmd")
	// huh.NextField() returns a nextFieldMsg — we verify it matches
	// by checking it's the same type as huh.NextField().
	assert.IsType(t, huh.NextField(), msg)
}

func TestUpdateShiftTabEmitsPrevField(t *testing.T) {
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	d := New(now)
	d.keymap.Prev.SetEnabled(true)

	_, msg := sendKey(d, "shift+tab")
	require.NotNil(t, msg)
	assert.IsType(t, huh.PrevField(), msg)
}

func TestUpdateEnterEmitsNextField(t *testing.T) {
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	d := New(now)
	d.keymap.Submit.SetEnabled(true)

	_, msg := sendKey(d, "enter")
	require.NotNil(t, msg)
	assert.IsType(t, huh.NextField(), msg)
}

func TestUpdateTabWritesAccessor(t *testing.T) {
	var v time.Time
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	d := New(now).Value(&v)
	d.keymap.Next.SetEnabled(true)

	sendKey(d, "tab")
	assert.Equal(t, now, v, "tab should sync value to accessor")
}

func TestUpdateEnterValidationBlocks(t *testing.T) {
	sentinel := errors.New("too early")
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	d := New(now).Validate(func(time.Time) error { return sentinel })
	d.keymap.Submit.SetEnabled(true)

	_, msg := sendKey(d, "enter")
	assert.Nil(t, msg, "enter should not emit NextField when validation fails")
	assert.Equal(t, sentinel, d.err)
}

func TestUpdateEnterValidationPasses(t *testing.T) {
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	d := New(now).Validate(func(time.Time) error { return nil })
	d.keymap.Submit.SetEnabled(true)

	_, msg := sendKey(d, "enter")
	require.NotNil(t, msg)
	assert.IsType(t, huh.NextField(), msg)
}

func TestUpdateClearsErrOnKeyPress(t *testing.T) {
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	d := New(now)
	d.err = errors.New("stale error")

	d, _ = sendKey(d, "left")
	assert.Nil(t, d.err, "any key press should clear the error")
}

// =============================================================================
// Error / Skip / Zoom
// =============================================================================

func TestError(t *testing.T) {
	d := New(time.Now())
	assert.Nil(t, d.Error())
	d.err = errors.New("oops")
	assert.EqualError(t, d.Error(), "oops")
}

func TestSkipZoom(t *testing.T) {
	d := New(time.Now())
	assert.False(t, d.Skip())
	assert.False(t, d.Zoom())
}

// =============================================================================
// GetKey / GetValue / KeyBinds
// =============================================================================

func TestGetKey(t *testing.T) {
	d := New(time.Now()).Key("mykey")
	assert.Equal(t, "mykey", d.GetKey())
}

func TestGetValue(t *testing.T) {
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	d := New(now)
	assert.Equal(t, now, d.GetValue())
}

func TestKeyBinds(t *testing.T) {
	d := New(time.Now())
	binds := d.KeyBinds()
	assert.Len(t, binds, 7)
}

// =============================================================================
// With* setters
// =============================================================================

func TestWithWidth(t *testing.T) {
	d := New(time.Now())
	result := d.WithWidth(80)
	assert.Equal(t, 80, d.width)
	assert.Same(t, d, result.(*DateTimePicker))
}

func TestWithHeight(t *testing.T) {
	d := New(time.Now())
	result := d.WithHeight(10)
	assert.Equal(t, 10, d.height)
	assert.Same(t, d, result.(*DateTimePicker))
}

func TestWithTheme(t *testing.T) {
	d := New(time.Now())
	assert.Nil(t, d.theme)

	theme := huh.ThemeCharm()
	d.WithTheme(theme)
	assert.Equal(t, theme, d.theme)

	// Second call should be ignored (theme already set).
	other := huh.ThemeDracula()
	d.WithTheme(other)
	assert.Equal(t, theme, d.theme, "WithTheme should not overwrite an already-set theme")
}

func TestWithPosition(t *testing.T) {
	d := New(time.Now())

	// Middle position: Prev enabled, Next enabled, Submit disabled.
	d.WithPosition(huh.FieldPosition{Field: 1, Group: 0, FirstField: 0, LastField: 2})
	assert.True(t, d.keymap.Prev.Enabled())
	assert.True(t, d.keymap.Next.Enabled())
	assert.False(t, d.keymap.Submit.Enabled())
}

func TestWithPositionFirst(t *testing.T) {
	d := New(time.Now())
	// Field==FirstField and Group==FirstGroup → IsFirst() == true
	d.WithPosition(huh.FieldPosition{Field: 0, Group: 0, FirstField: 0, LastField: 2})
	assert.False(t, d.keymap.Prev.Enabled(), "Prev should be disabled at first field")
}

func TestWithPositionLast(t *testing.T) {
	d := New(time.Now())
	// Field==LastField and Group==LastGroup → IsLast() == true
	d.WithPosition(huh.FieldPosition{Field: 2, Group: 0, LastField: 2})
	assert.True(t, d.keymap.Submit.Enabled(), "Submit should be enabled at last field")
	assert.False(t, d.keymap.Next.Enabled(), "Next should be disabled at last field")
}

func TestWithKeyMap(t *testing.T) {
	d := New(time.Now())
	km := huh.NewDefaultKeyMap()
	d.WithKeyMap(km)
	// The keymap sets Next/Prev/Submit from huh's global keymap — just verify
	// it doesn't panic and returns the field.
	assert.NotNil(t, d)
}

// =============================================================================
// View
// =============================================================================

func TestViewContainsTitle(t *testing.T) {
	now := time.Date(2025, 6, 15, 14, 30, 45, 0, time.UTC)
	d := New(now).Title("Pick a time")
	d.WithWidth(40)
	view := d.View()
	assert.Contains(t, view, "Pick a time")
}

func TestViewContainsDescription(t *testing.T) {
	now := time.Date(2025, 6, 15, 14, 30, 45, 0, time.UTC)
	d := New(now).Description("Choose wisely")
	d.WithWidth(40)
	view := d.View()
	assert.Contains(t, view, "Choose wisely")
}

func TestViewContainsSegmentValues(t *testing.T) {
	now := time.Date(2025, 6, 15, 14, 30, 45, 0, time.UTC)
	d := New(now)
	d.WithWidth(40)
	view := d.View()
	// Strip ANSI so we can do plain string checks.
	plain := stripANSI(view)
	assert.Contains(t, plain, "2025")
	assert.Contains(t, plain, "06")
	assert.Contains(t, plain, "15")
	assert.Contains(t, plain, "14")
	assert.Contains(t, plain, "30")
	assert.Contains(t, plain, "45")
}

func TestViewContainsError(t *testing.T) {
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	d := New(now)
	d.err = errors.New("value out of range")
	d.WithWidth(40)
	view := d.View()
	assert.Contains(t, view, "value out of range")
}

func TestViewModeDateOmitsTime(t *testing.T) {
	now := time.Date(2025, 6, 15, 14, 30, 45, 0, time.UTC)
	d := New(now).Mode(ModeDate)
	d.WithWidth(40)
	plain := stripANSI(d.View())
	assert.Contains(t, plain, "2025")
	assert.NotContains(t, plain, "14", "ModeDate should not show hours")
}

func TestViewModeDateTimeOmitsSeconds(t *testing.T) {
	now := time.Date(2025, 6, 15, 14, 30, 45, 0, time.UTC)
	d := New(now).Mode(ModeDateTime)
	d.WithWidth(40)
	plain := stripANSI(d.View())
	assert.Contains(t, plain, "2025")
	assert.Contains(t, plain, "14", "ModeDateTime should show hours")
	assert.Contains(t, plain, "30", "ModeDateTime should show minutes")
	assert.NotContains(t, plain, "45", "ModeDateTime should not show seconds")
}

// stripANSI removes ANSI escape sequences for plain-text assertions.
func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			inEsc = true
		case inEsc && r == 'm':
			inEsc = false
		case !inEsc:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// =============================================================================
// RunAccessible
// =============================================================================

func TestRunAccessible(t *testing.T) {
	now := time.Date(2025, 6, 15, 14, 30, 45, 0, time.UTC)
	d := New(now).Mode(ModeDateTimeSec)

	var out strings.Builder
	in := strings.NewReader("2030-01-20 08:00:00\n")

	err := d.RunAccessible(&out, in)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2030, 1, 20, 8, 0, 0, 0, time.UTC), d.t)
}

func TestRunAccessibleInvalidInput(t *testing.T) {
	d := New(time.Now()).Mode(ModeDate)
	var out strings.Builder
	in := strings.NewReader("not-a-date\n")
	err := d.RunAccessible(&out, in)
	assert.Error(t, err)
}

func TestRunAccessibleValidationFails(t *testing.T) {
	sentinel := errors.New("too old")
	d := New(time.Now()).Mode(ModeDate).Validate(func(time.Time) error { return sentinel })
	var out strings.Builder
	in := strings.NewReader("2020-01-01\n")
	err := d.RunAccessible(&out, in)
	assert.Equal(t, sentinel, err)
}
