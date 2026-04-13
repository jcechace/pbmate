package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThemeResolveDefaultTheme(t *testing.T) {
	dark := LookupTheme("default", true)
	light := LookupTheme("default", false)

	assert.NotEqual(t, dark.Subtle, light.Subtle)
	assert.NotEqual(t, dark.Highlight, light.Highlight)
	assert.NotEqual(t, dark.OK, light.OK)
	assert.Equal(t, "swapoff", dark.ChromaStyle)
	assert.Equal(t, "swapoff", light.ChromaStyle)
	assert.True(t, dark.isDark)
	assert.False(t, light.isDark)
}

func TestThemeResolveCatppuccinTheme(t *testing.T) {
	resolvedDark := LookupTheme("latte", true)
	resolvedLight := LookupTheme("latte", false)

	assert.Equal(t, resolvedDark.Primary, resolvedLight.Primary)
	assert.Equal(t, resolvedDark.Subtle, resolvedLight.Subtle)
	assert.Equal(t, resolvedDark.ChromaStyle, resolvedLight.ChromaStyle)
	assert.False(t, resolvedDark.isDark)
	assert.False(t, resolvedLight.isDark)
}

func TestDefaultThemeVariants(t *testing.T) {
	dark := defaultDarkTheme()
	light := defaultLightTheme()

	assert.True(t, dark.isDark)
	assert.False(t, light.isDark)
	assert.NotEqual(t, dark.Subtle, light.Subtle)
	assert.NotEqual(t, dark.Highlight, light.Highlight)
	assert.NotEqual(t, dark.OK, light.OK)
}
