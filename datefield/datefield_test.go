package datefield

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// segCount
// =============================================================================

func TestSegCount(t *testing.T) {
	cases := []struct {
		mode Mode
		want int
	}{
		{ModeDate, 3},
		{ModeDateTime, 5},
		{ModeDateTimeSec, 6},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, segCount(tc.mode), "mode %v", tc.mode)
	}
}

// =============================================================================
// segWidth
// =============================================================================

func TestSegWidth(t *testing.T) {
	assert.Equal(t, 4, segWidth(segYear))
	assert.Equal(t, 2, segWidth(segMonth))
	assert.Equal(t, 2, segWidth(segDay))
	assert.Equal(t, 2, segWidth(segHour))
	assert.Equal(t, 2, segWidth(segMinute))
	assert.Equal(t, 2, segWidth(segSecond))
}

// =============================================================================
// daysInMonth
// =============================================================================

func TestDaysInMonth(t *testing.T) {
	cases := []struct {
		year  int
		month time.Month
		want  int
	}{
		{2024, time.January, 31},
		{2024, time.February, 29}, // leap year
		{2023, time.February, 28}, // non-leap year
		{2024, time.April, 30},
		{2024, time.December, 31},
	}
	for _, tc := range cases {
		got := daysInMonth(tc.year, tc.month)
		assert.Equal(t, tc.want, got, "%d-%02d", tc.year, tc.month)
	}
}

// =============================================================================
// wrapInt
// =============================================================================

func TestWrapInt(t *testing.T) {
	cases := []struct {
		v, min, max, want int
		desc              string
	}{
		{6, 1, 5, 1, "wrap above max"},
		{0, 1, 5, 5, "wrap below min"},
		{3, 1, 5, 3, "in range"},
		{1, 1, 5, 1, "at min"},
		{5, 1, 5, 5, "at max"},
		{13, 1, 12, 1, "month wrap over"},
		{0, 1, 12, 12, "month wrap under"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, wrapInt(tc.v, tc.min, tc.max), tc.desc)
	}
}

// =============================================================================
// setSegValue
// =============================================================================

func TestSetSegValue(t *testing.T) {
	base := time.Date(2025, 6, 15, 12, 30, 45, 0, time.UTC)

	cases := []struct {
		seg  segment
		v    int
		want int
		desc string
	}{
		{segYear, 2030, 2030, "set year"},
		{segYear, 1066, 1066, "year below 1970 is allowed"},
		{segYear, 2124, 2124, "year above 2099 is allowed"},
		{segMonth, 12, 12, "set month"},
		{segMonth, 13, 1, "wrap month 13 → 1"},
		{segMonth, 0, 12, "wrap month 0 → 12"},
		{segDay, 28, 28, "set day"},
		{segHour, 23, 23, "set hour"},
		{segHour, 24, 0, "wrap hour 24 → 0"},
		{segMinute, 59, 59, "set minute"},
		{segMinute, 60, 0, "wrap minute 60 → 0"},
		{segSecond, 0, 0, "set second"},
		{segSecond, 61, 1, "wrap second 61 → 1"},
	}

	getters := map[segment]func(time.Time) int{
		segYear:   func(t time.Time) int { return t.Year() },
		segMonth:  func(t time.Time) int { return int(t.Month()) },
		segDay:    func(t time.Time) int { return t.Day() },
		segHour:   func(t time.Time) int { return t.Hour() },
		segMinute: func(t time.Time) int { return t.Minute() },
		segSecond: func(t time.Time) int { return t.Second() },
	}

	for _, tc := range cases {
		got := setSegValue(base, tc.seg, tc.v)
		assert.Equal(t, tc.want, getters[tc.seg](got), tc.desc)
	}
}

// setSegValue should clamp day when changing month would exceed new month's days.
func TestSetSegValueDayClampOnMonthChange(t *testing.T) {
	// Jan 31 → change month to Feb (28 days in 2025)
	base := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	got := setSegValue(base, segMonth, 2)
	assert.Equal(t, 28, got.Day(), "day should clamp to 28 for Feb 2025")
	assert.Equal(t, time.February, got.Month())
}

// setSegValue should clamp day when changing year makes it a non-leap year.
func TestSetSegValueDayClampOnYearChange(t *testing.T) {
	// Feb 29, 2024 (leap) → change year to 2025 (non-leap)
	base := time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)
	got := setSegValue(base, segYear, 2025)
	assert.Equal(t, 28, got.Day(), "day should clamp to 28 for Feb 2025")
}

// =============================================================================
// formatSegment
// =============================================================================

func TestFormatSegment(t *testing.T) {
	cases := []struct {
		seg  segment
		v    int
		want string
	}{
		{segYear, 2025, "2025"},
		{segYear, 300, "0300"},
		{segMonth, 3, "03"},
		{segDay, 5, "05"},
		{segHour, 0, "00"},
		{segMinute, 59, "59"},
		{segSecond, 9, "09"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, formatSegment(tc.seg, tc.v))
	}
}

// =============================================================================
// New / builder methods
// =============================================================================

func TestNew(t *testing.T) {
	now := time.Date(2025, 3, 5, 14, 30, 0, 0, time.UTC)
	d := New(now)
	require.NotNil(t, d)
	assert.Equal(t, ModeDateTimeSec, d.mode)
	assert.Equal(t, now, d.t)
	assert.Equal(t, segYear, d.activeSeg)
	assert.False(t, d.focused)
}

func TestNewConvertsToUTC(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	local := time.Date(2025, 3, 5, 9, 0, 0, 0, loc) // 09:00 EST = 14:00 UTC
	d := New(local)
	assert.Equal(t, time.UTC, d.t.Location())
	assert.Equal(t, 14, d.t.Hour())
}

func TestBuilderMethods(t *testing.T) {
	now := time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC)
	d := New(now).
		Title("Pick date").
		Description("Choose wisely").
		Mode(ModeDate).
		Key("mykey")

	assert.Equal(t, "Pick date", d.title)
	assert.Equal(t, "Choose wisely", d.desc)
	assert.Equal(t, ModeDate, d.mode)
	assert.Equal(t, "mykey", d.fieldKey)
}

func TestModeClampsSeg(t *testing.T) {
	now := time.Date(2025, 3, 5, 14, 30, 45, 0, time.UTC)
	d := New(now)
	d.activeSeg = segSecond // 5 — valid for ModeDateTimeSec

	// Switching to ModeDate (3 segs) should clamp activeSeg to 2 (segDay).
	d.Mode(ModeDate)
	assert.Equal(t, segment(2), d.activeSeg)
}

func TestValueSetsPointer(t *testing.T) {
	var v time.Time
	initial := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	d := New(initial).Value(&v)
	// Value with zero pointer should keep the initial time (v was zero).
	assert.Equal(t, initial, d.t)

	// Value with a non-zero pointer should use it.
	preset := time.Date(2030, 1, 15, 0, 0, 0, 0, time.UTC)
	d2 := New(initial).Value(&preset)
	assert.Equal(t, preset.UTC(), d2.t)
}

func TestValidateFn(t *testing.T) {
	now := time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC)
	called := false
	d := New(now).Validate(func(t time.Time) error {
		called = true
		return nil
	})
	err := d.validate(now)
	assert.NoError(t, err)
	assert.True(t, called)
}

// =============================================================================
// segValue
// =============================================================================

func TestSegValue(t *testing.T) {
	base := time.Date(2025, 3, 5, 14, 30, 45, 0, time.UTC)
	d := New(base)

	cases := []struct {
		seg  segment
		want int
	}{
		{segYear, 2025},
		{segMonth, 3},
		{segDay, 5},
		{segHour, 14},
		{segMinute, 30},
		{segSecond, 45},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, d.segValue(tc.seg), "seg %v", tc.seg)
	}
}

// =============================================================================
// prevSeg / nextSeg
// =============================================================================

func TestPrevNextSeg(t *testing.T) {
	now := time.Date(2025, 3, 5, 14, 30, 45, 0, time.UTC)
	d := New(now) // ModeDateTimeSec, 6 segs

	// prevSeg at first segment is a no-op.
	d.activeSeg = segYear
	d.prevSeg()
	assert.Equal(t, segYear, d.activeSeg)

	// nextSeg advances one step at a time.
	d.activeSeg = segYear
	d.nextSeg()
	assert.Equal(t, segMonth, d.activeSeg)

	// nextSeg at last segment is a no-op.
	d.activeSeg = segSecond // 5
	d.nextSeg()
	assert.Equal(t, segSecond, d.activeSeg)

	// prevSeg retreats.
	d.activeSeg = segSecond
	d.prevSeg()
	assert.Equal(t, segMinute, d.activeSeg)
}

func TestPrevNextSegFlushesDigitBuf(t *testing.T) {
	now := time.Date(2025, 3, 5, 14, 30, 45, 0, time.UTC)
	d := New(now)
	d.activeSeg = segMonth
	d.digitBuf = "1"

	d.nextSeg()
	assert.Empty(t, d.digitBuf)

	d.digitBuf = "2"
	d.prevSeg()
	assert.Empty(t, d.digitBuf)
}

// =============================================================================
// addDigit / flushDigitBuf
// =============================================================================

func TestAddDigitYearAutoAdvance(t *testing.T) {
	now := time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC)
	d := New(now)
	d.activeSeg = segYear

	// Year needs 4 digits — should not auto-advance until full.
	d.addDigit('2')
	assert.Equal(t, segYear, d.activeSeg)
	d.addDigit('0')
	assert.Equal(t, segYear, d.activeSeg)
	d.addDigit('3')
	assert.Equal(t, segYear, d.activeSeg)
	d.addDigit('0')
	// Now full — should advance to segMonth.
	assert.Equal(t, segMonth, d.activeSeg)
	assert.Equal(t, 2030, d.t.Year())
	assert.Empty(t, d.digitBuf)
}

func TestAddDigitTwoDigitSeg(t *testing.T) {
	now := time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC)
	d := New(now)
	d.activeSeg = segMonth

	d.addDigit('1') // partial — not enough digits yet
	assert.Equal(t, segMonth, d.activeSeg)
	assert.Equal(t, "1", d.digitBuf)

	d.addDigit('2') // full — should auto-advance to segDay
	assert.Equal(t, segDay, d.activeSeg)
	assert.Equal(t, time.December, d.t.Month())
	assert.Empty(t, d.digitBuf)
}

func TestFlushDigitBuf(t *testing.T) {
	now := time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC)
	d := New(now)
	d.digitBuf = "12"
	d.flushDigitBuf()
	assert.Empty(t, d.digitBuf)
}

func TestAddDigitLastSegmentStaysAndClears(t *testing.T) {
	now := time.Date(2025, 3, 5, 14, 30, 0, 0, time.UTC)
	d := New(now) // ModeDateTimeSec — last segment is segSecond (index 5)
	d.activeSeg = segSecond

	d.addDigit('4')
	// One digit in — not yet full, stays on segSecond.
	assert.Equal(t, segSecond, d.activeSeg)
	assert.Equal(t, "4", d.digitBuf)

	d.addDigit('5')
	// Two digits in — full. nextSeg is a no-op at the last segment.
	assert.Equal(t, segSecond, d.activeSeg)
	assert.Equal(t, 45, d.t.Second())
	assert.Empty(t, d.digitBuf)
}

// =============================================================================
// Mode-specific segment navigation bounds
// =============================================================================

func TestNextSegBoundsModeDate(t *testing.T) {
	now := time.Date(2025, 3, 5, 14, 30, 45, 0, time.UTC)
	d := New(now).Mode(ModeDate) // 3 segments: year, month, day

	d.activeSeg = segDay
	d.nextSeg()
	assert.Equal(t, segDay, d.activeSeg, "nextSeg at segDay should be a no-op in ModeDate")
}

func TestNextSegBoundsModeDateTime(t *testing.T) {
	now := time.Date(2025, 3, 5, 14, 30, 45, 0, time.UTC)
	d := New(now).Mode(ModeDateTime) // 5 segments: year, month, day, hour, minute

	d.activeSeg = segMinute
	d.nextSeg()
	assert.Equal(t, segMinute, d.activeSeg, "nextSeg at segMinute should be a no-op in ModeDateTime")
}

// =============================================================================
// setSegValue preserves untouched segments
// =============================================================================

func TestSetSegValuePreservesOtherSegments(t *testing.T) {
	original := time.Date(2025, 3, 15, 14, 30, 45, 0, time.UTC)

	tests := []struct {
		name string
		seg  segment
		v    int
	}{
		{"change year", segYear, 2030},
		{"change month", segMonth, 6},
		{"change day", segDay, 20},
		{"change hour", segHour, 8},
		{"change minute", segMinute, 0},
		{"change second", segSecond, 59},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := setSegValue(original, tt.seg, tt.v)
			// The changed segment should have the new value.
			d := New(result)
			assert.Equal(t, tt.v, d.segValue(tt.seg))
			// All other segments should be unchanged.
			for _, s := range []segment{segYear, segMonth, segDay, segHour, segMinute, segSecond} {
				if s == tt.seg {
					continue
				}
				assert.Equal(t, New(original).segValue(s), d.segValue(s), "segment %v should be unchanged", s)
			}
		})
	}
}
