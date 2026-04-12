package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThemeResolveDefaultTheme(t *testing.T) {
	theme := DefaultTheme()

	dark := theme.Resolve(true)
	light := theme.Resolve(false)

	assert.NotEqual(t, dark.Subtle, light.Subtle)
	assert.NotEqual(t, dark.Highlight, light.Highlight)
	assert.NotEqual(t, dark.OK, light.OK)
	assert.Equal(t, defaultChromaStyle, dark.ChromaStyle)
	assert.Equal(t, defaultChromaStyle, light.ChromaStyle)
}

func TestThemeResolveCatppuccinTheme(t *testing.T) {
	theme := CatppuccinLatte()

	assert.Equal(t, theme, theme.Resolve(true))
	assert.Equal(t, theme, theme.Resolve(false))
}
