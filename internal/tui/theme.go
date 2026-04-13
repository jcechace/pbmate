package tui

import (
	"image/color"
	"strings"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	catppuccingo "github.com/catppuccin/go"
)

// Theme is the concrete palette used by the TUI at render time.
type Theme struct {
	Primary   color.Color
	Subtle    color.Color
	Highlight color.Color
	OK        color.Color
	Error     color.Color
	Warning   color.Color
	Muted     color.Color

	// ChromaStyle is the Chroma syntax highlighting style name for YAML
	// rendering. Must match a registered Chroma style.
	ChromaStyle string

	// isDark records whether the resolved palette is dark. Huh's base theme
	// still needs this after the concrete colors have been chosen.
	isDark bool

	// flavor is set only for fixed Catppuccin themes so the Huh theme can reuse
	// the exact same flavor-specific palette.
	flavor *catppuccingo.Flavor
}

func defaultLightTheme() Theme {
	return Theme{
		Primary:     lipgloss.Color("62"),
		Subtle:      lipgloss.Color("245"),
		Highlight:   lipgloss.Color("236"),
		OK:          lipgloss.Color("34"),
		Error:       lipgloss.Color("160"),
		Warning:     lipgloss.Color("172"),
		Muted:       lipgloss.Color("245"),
		ChromaStyle: "swapoff",
		isDark:      false,
	}
}

func defaultDarkTheme() Theme {
	return Theme{
		Primary:     lipgloss.Color("62"),
		Subtle:      lipgloss.Color("241"),
		Highlight:   lipgloss.Color("252"),
		OK:          lipgloss.Color("42"),
		Error:       lipgloss.Color("196"),
		Warning:     lipgloss.Color("214"),
		Muted:       lipgloss.Color("241"),
		ChromaStyle: "swapoff",
		isDark:      true,
	}
}

func catppuccinTheme(f catppuccingo.Flavor, chromaStyle string) Theme {
	return Theme{
		Primary:     lipgloss.Color(f.Blue().Hex),
		Subtle:      lipgloss.Color(f.Overlay1().Hex),
		Highlight:   lipgloss.Color(f.Text().Hex),
		OK:          lipgloss.Color(f.Green().Hex),
		Error:       lipgloss.Color(f.Red().Hex),
		Warning:     lipgloss.Color(f.Peach().Hex),
		Muted:       lipgloss.Color(f.Overlay0().Hex),
		ChromaStyle: chromaStyle,
		isDark:      f != catppuccingo.Latte,
		flavor:      &f,
	}
}

// Theme registry by canonical or accepted config name. The default adaptive
// theme is handled in LookupTheme and uses the dedicated default-light and
// default-dark entries.
var themeRegistry = map[string]Theme{
	"default-light":        defaultLightTheme(),
	"default-dark":         defaultDarkTheme(),
	"mocha":                catppuccinTheme(catppuccingo.Mocha, "catppuccin-mocha"),
	"catppuccin-mocha":     catppuccinTheme(catppuccingo.Mocha, "catppuccin-mocha"),
	"latte":                catppuccinTheme(catppuccingo.Latte, "catppuccin-latte"),
	"catppuccin-latte":     catppuccinTheme(catppuccingo.Latte, "catppuccin-latte"),
	"frappe":               catppuccinTheme(catppuccingo.Frappe, "catppuccin-frappe"),
	"catppuccin-frappe":    catppuccinTheme(catppuccingo.Frappe, "catppuccin-frappe"),
	"macchiato":            catppuccinTheme(catppuccingo.Macchiato, "catppuccin-macchiato"),
	"catppuccin-macchiato": catppuccinTheme(catppuccingo.Macchiato, "catppuccin-macchiato"),
}

// LookupTheme resolves a configured theme name to a concrete runtime palette.
// The adaptive default theme chooses between default-light and default-dark
// based on the terminal background. Unknown names fall back to the adaptive
// default theme.
func LookupTheme(name string, isDark bool) Theme {
	name = strings.ToLower(name)
	if name == "" || name == "default" {
		if isDark {
			return themeRegistry["default-dark"]
		}
		return themeRegistry["default-light"]
	}
	if theme, ok := themeRegistry[name]; ok {
		return theme
	}
	if isDark {
		return themeRegistry["default-dark"]
	}
	return themeRegistry["default-light"]
}

// HuhTheme returns a huh form theme matching this resolved palette.
func (t Theme) HuhTheme() huh.Theme {
	if t.flavor != nil {
		return huh.ThemeFunc(func(bool) *huh.Styles {
			return huhCatppuccinTheme(*t.flavor)
		})
	}
	return huh.ThemeFunc(func(bool) *huh.Styles {
		return huhThemeFromTheme(t)
	})
}

// BorderlessHuhTheme hides Huh's group border so the surrounding overlay chrome
// remains the only visible frame in compact forms.
func BorderlessHuhTheme(theme huh.Theme) huh.Theme {
	if theme == nil {
		theme = huh.ThemeFunc(huh.ThemeCharm)
	}
	return huh.ThemeFunc(func(isDark bool) *huh.Styles {
		styles := theme.Theme(isDark)
		clone := *styles
		clone.Focused.Base = clone.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
		return &clone
	})
}

func huhThemeFromTheme(t Theme) *huh.Styles {
	styles := huh.ThemeBase(t.isDark)

	styles.Focused.Base = styles.Focused.Base.BorderForeground(t.Primary)
	styles.Focused.Card = styles.Focused.Base
	styles.Focused.Title = styles.Focused.Title.Foreground(t.Primary)
	styles.Focused.NoteTitle = styles.Focused.NoteTitle.Foreground(t.Primary)
	styles.Focused.Directory = styles.Focused.Directory.Foreground(t.Primary)
	styles.Focused.Description = styles.Focused.Description.Foreground(t.Muted)
	styles.Focused.ErrorIndicator = styles.Focused.ErrorIndicator.Foreground(t.Error)
	styles.Focused.ErrorMessage = styles.Focused.ErrorMessage.Foreground(t.Error)
	styles.Focused.SelectSelector = styles.Focused.SelectSelector.Foreground(t.Primary)
	styles.Focused.NextIndicator = styles.Focused.NextIndicator.Foreground(t.Primary)
	styles.Focused.PrevIndicator = styles.Focused.PrevIndicator.Foreground(t.Primary)
	styles.Focused.Option = styles.Focused.Option.Foreground(t.Highlight)
	styles.Focused.MultiSelectSelector = styles.Focused.MultiSelectSelector.Foreground(t.Primary)
	styles.Focused.SelectedOption = styles.Focused.SelectedOption.Foreground(t.OK)
	styles.Focused.SelectedPrefix = styles.Focused.SelectedPrefix.Foreground(t.OK)
	styles.Focused.UnselectedPrefix = styles.Focused.UnselectedPrefix.Foreground(t.Highlight)
	styles.Focused.UnselectedOption = styles.Focused.UnselectedOption.Foreground(t.Highlight)
	styles.Focused.FocusedButton = styles.Focused.FocusedButton.Foreground(t.Highlight).Background(t.Primary)
	styles.Focused.BlurredButton = styles.Focused.BlurredButton.Foreground(t.Highlight)

	styles.Focused.TextInput.Cursor = styles.Focused.TextInput.Cursor.Foreground(t.Primary)
	styles.Focused.TextInput.Placeholder = styles.Focused.TextInput.Placeholder.Foreground(t.Muted)
	styles.Focused.TextInput.Prompt = styles.Focused.TextInput.Prompt.Foreground(t.Primary)

	styles.Blurred = styles.Focused
	styles.Blurred.Base = styles.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	styles.Blurred.Card = styles.Blurred.Base

	styles.Help.Ellipsis = styles.Help.Ellipsis.Foreground(t.Muted)
	styles.Help.ShortKey = styles.Help.ShortKey.Foreground(t.Muted)
	styles.Help.ShortDesc = styles.Help.ShortDesc.Foreground(t.Subtle)
	styles.Help.ShortSeparator = styles.Help.ShortSeparator.Foreground(t.Muted)
	styles.Help.FullKey = styles.Help.FullKey.Foreground(t.Muted)
	styles.Help.FullDesc = styles.Help.FullDesc.Foreground(t.Subtle)
	styles.Help.FullSeparator = styles.Help.FullSeparator.Foreground(t.Muted)

	styles.Group.Title = styles.Focused.Title
	styles.Group.Description = styles.Focused.Description

	return styles
}

// huhCatppuccinTheme builds a huh form theme from a specific Catppuccin
// flavor, using hardcoded colors instead of adaptive ones. This mirrors
// huh.ThemeCatppuccin() but pins to the chosen flavor.
func huhCatppuccinTheme(f catppuccingo.Flavor) *huh.Styles {
	t := huh.ThemeBase(f != catppuccingo.Latte)

	base := lipgloss.Color(f.Base().Hex)
	text := lipgloss.Color(f.Text().Hex)
	subtext1 := lipgloss.Color(f.Subtext1().Hex)
	subtext0 := lipgloss.Color(f.Subtext0().Hex)
	overlay1 := lipgloss.Color(f.Overlay1().Hex)
	overlay0 := lipgloss.Color(f.Overlay0().Hex)
	green := lipgloss.Color(f.Green().Hex)
	red := lipgloss.Color(f.Red().Hex)
	pink := lipgloss.Color(f.Pink().Hex)
	mauve := lipgloss.Color(f.Mauve().Hex)
	cursor := lipgloss.Color(f.Rosewater().Hex)

	t.Focused.Base = t.Focused.Base.BorderForeground(subtext1)
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = t.Focused.Title.Foreground(mauve)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(mauve)
	t.Focused.Directory = t.Focused.Directory.Foreground(mauve)
	t.Focused.Description = t.Focused.Description.Foreground(subtext0)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(red)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(red)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(pink)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(pink)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(pink)
	t.Focused.Option = t.Focused.Option.Foreground(text)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(pink)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(green)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(green)
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(text)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(text)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(base).Background(pink)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(text).Background(base)

	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(cursor)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(overlay0)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(pink)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base

	t.Help.Ellipsis = t.Help.Ellipsis.Foreground(subtext0)
	t.Help.ShortKey = t.Help.ShortKey.Foreground(subtext0)
	t.Help.ShortDesc = t.Help.ShortDesc.Foreground(overlay1)
	t.Help.ShortSeparator = t.Help.ShortSeparator.Foreground(subtext0)
	t.Help.FullKey = t.Help.FullKey.Foreground(subtext0)
	t.Help.FullDesc = t.Help.FullDesc.Foreground(overlay1)
	t.Help.FullSeparator = t.Help.FullSeparator.Foreground(subtext0)

	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description

	return t
}
