package tui

import (
	"strings"

	catppuccingo "github.com/catppuccin/go"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Theme holds the color palette used throughout the TUI.
type Theme struct {
	Primary   lipgloss.TerminalColor
	Subtle    lipgloss.TerminalColor
	Highlight lipgloss.TerminalColor
	OK        lipgloss.TerminalColor
	Error     lipgloss.TerminalColor
	Warning   lipgloss.TerminalColor
	Muted     lipgloss.TerminalColor

	// ChromaStyle is the Chroma syntax highlighting style name for YAML
	// rendering. Must match a registered Chroma style (e.g. "catppuccin-mocha").
	ChromaStyle string

	// flavor is the Catppuccin flavor for this theme. Nil for the default
	// adaptive theme. Used to build a matching huh form theme.
	flavor *catppuccingo.Flavor
}

// defaultChromaStyle is used when the theme does not specify a Chroma style.
const defaultChromaStyle = "swapoff"

// DefaultTheme returns an adaptive theme that works on both light and dark
// terminals using standard ANSI colors.
func DefaultTheme() Theme {
	return Theme{
		Primary:     lipgloss.AdaptiveColor{Light: "62", Dark: "62"},
		Subtle:      lipgloss.AdaptiveColor{Light: "245", Dark: "241"},
		Highlight:   lipgloss.AdaptiveColor{Light: "236", Dark: "252"},
		OK:          lipgloss.AdaptiveColor{Light: "34", Dark: "42"},
		Error:       lipgloss.AdaptiveColor{Light: "160", Dark: "196"},
		Warning:     lipgloss.AdaptiveColor{Light: "172", Dark: "214"},
		Muted:       lipgloss.AdaptiveColor{Light: "245", Dark: "241"},
		ChromaStyle: defaultChromaStyle,
	}
}

// catppuccinTheme builds a Theme from a catppuccin flavor.
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
		flavor:      &f,
	}
}

// CatppuccinMocha returns the Catppuccin Mocha (dark) theme.
func CatppuccinMocha() Theme { return catppuccinTheme(catppuccingo.Mocha, "catppuccin-mocha") }

// CatppuccinLatte returns the Catppuccin Latte (light) theme.
func CatppuccinLatte() Theme { return catppuccinTheme(catppuccingo.Latte, "catppuccin-latte") }

// CatppuccinFrappe returns the Catppuccin Frappe (dark, muted) theme.
func CatppuccinFrappe() Theme { return catppuccinTheme(catppuccingo.Frappe, "catppuccin-frappe") }

// CatppuccinMacchiato returns the Catppuccin Macchiato (dark, warm) theme.
func CatppuccinMacchiato() Theme {
	return catppuccinTheme(catppuccingo.Macchiato, "catppuccin-macchiato")
}

// ThemeByName returns a theme by name. Unrecognised names fall back to the
// default adaptive theme.
func ThemeByName(name string) Theme {
	switch strings.ToLower(name) {
	case "mocha", "catppuccin-mocha":
		return CatppuccinMocha()
	case "latte", "catppuccin-latte":
		return CatppuccinLatte()
	case "frappe", "catppuccin-frappe":
		return CatppuccinFrappe()
	case "macchiato", "catppuccin-macchiato":
		return CatppuccinMacchiato()
	default:
		return DefaultTheme()
	}
}

// HuhTheme returns a huh form theme matching this theme. For Catppuccin
// themes, the form colors use the same flavor. For the default adaptive
// theme, huh's built-in adaptive Catppuccin theme is used.
func (t Theme) HuhTheme() *huh.Theme {
	if t.flavor == nil {
		return huh.ThemeCatppuccin()
	}
	return huhCatppuccinTheme(*t.flavor)
}

// huhCatppuccinTheme builds a huh form theme from a specific Catppuccin
// flavor, using hardcoded colors instead of adaptive ones. This mirrors
// huh.ThemeCatppuccin() but pins to the chosen flavor.
func huhCatppuccinTheme(f catppuccingo.Flavor) *huh.Theme {
	t := huh.ThemeBase()

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
