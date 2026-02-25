package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHorizontalSplit(t *testing.T) {
	tests := []struct {
		name                                string
		totalW                              int
		wantPanelLeftW, wantPanelRightW     int
		wantContentLeftW, wantContentRightW int
		expectLeftPlusBorderEqualsLeftW     bool
	}{
		{
			name:   "standard 80 column terminal",
			totalW: 80,
			// 80 * 30% = 24, but min is 28, so leftW = 28, rightW = 52
			wantPanelLeftW:    26, // 28 - panelBorderH(2)
			wantPanelRightW:   50, // 52 - panelBorderH(2)
			wantContentLeftW:  24, // 28 - panelBorderH(2) - panelPaddingH(2)
			wantContentRightW: 48, // 52 - panelBorderH(2) - panelPaddingH(2)
		},
		{
			name:   "narrow terminal hits minimum left width",
			totalW: 60,
			// 60 * 30% = 18, but min is 28, so leftW = 28, rightW = 32
			wantPanelLeftW:    26, // 28 - 2
			wantPanelRightW:   30, // 32 - 2
			wantContentLeftW:  24, // 28 - 2 - 2
			wantContentRightW: 28, // 32 - 2 - 2
		},
		{
			name:   "wide terminal 120 columns",
			totalW: 120,
			// 120 * 30% = 36, right = 84
			wantPanelLeftW:    34,
			wantPanelRightW:   82,
			wantContentLeftW:  32,
			wantContentRightW: 80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panelLeftW, panelRightW, contentLeftW, contentRightW := horizontalSplit(tt.totalW)
			assert.Equal(t, tt.wantPanelLeftW, panelLeftW, "panelLeftW")
			assert.Equal(t, tt.wantPanelRightW, panelRightW, "panelRightW")
			assert.Equal(t, tt.wantContentLeftW, contentLeftW, "contentLeftW")
			assert.Equal(t, tt.wantContentRightW, contentRightW, "contentRightW")
		})
	}
}

func TestHorizontalSplitPanelWidthsSumToTotal(t *testing.T) {
	// For any reasonable terminal width, left + right panel widths (plus
	// borders) should sum to the total terminal width.
	for totalW := 40; totalW <= 200; totalW++ {
		panelLeftW, panelRightW, _, _ := horizontalSplit(totalW)
		// Panel width is inside border, so outer = panelW + panelBorderH.
		outerLeft := panelLeftW + panelBorderH
		outerRight := panelRightW + panelBorderH
		require.Equal(t, totalW, outerLeft+outerRight,
			"total width %d: outer left (%d) + outer right (%d) != total", totalW, outerLeft, outerRight)
	}
}

func TestInnerHeight(t *testing.T) {
	tests := []struct {
		panelH   int
		expected int
	}{
		{10, 6}, // 10 - panelBorderV(2) - panelPaddingV(2) = 6
		{4, 0},  // 4 - 4 = 0 (minimum)
		{3, 0},  // clamped to 0
		{0, 0},  // clamped to 0
		{100, 96},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, innerHeight(tt.panelH), "innerHeight(%d)", tt.panelH)
	}
}
