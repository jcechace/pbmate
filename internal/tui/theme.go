package tui

import (
	"strings"

	catppuccingo "github.com/catppuccin/go"
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
