// Package theme defines grom's color themes. A Theme is a small set of
// semantic color tokens; render primitives derive lipgloss styles from it.
package theme

import "github.com/charmbracelet/lipgloss"

// Theme is a semantic color palette. All values are hex colors.
type Theme struct {
	Name    string
	Accent  string // titles, focused borders, primary data
	Success string // good values, success thresholds
	Error   string // errors, critical thresholds
	Warning string // warnings, amber thresholds
	Border  string // panel borders
	Label   string // widget titles, legends
	Dim     string // secondary text, axis labels
	DimMore string // faintest text (padding glyphs, hints)
	Series  []string
}

// Style helpers — precomputed on demand; lipgloss styles are cheap values.

func (t Theme) TitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(t.Label))
}

func (t Theme) BorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(t.Border))
}

func (t Theme) FocusBorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(t.Accent))
}

func (t Theme) AccentStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(t.Accent))
}

func (t Theme) SuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success))
}

func (t Theme) ErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error))
}

func (t Theme) WarningStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(t.Warning))
}

func (t Theme) LabelStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(t.Label))
}

func (t Theme) DimStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(t.Dim))
}

func (t Theme) DimMoreStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(t.DimMore))
}

// SeriesStyle returns the style for the i-th chart series, cycling the palette.
func (t Theme) SeriesStyle(i int) lipgloss.Style {
	c := t.Series[i%len(t.Series)]
	return lipgloss.NewStyle().Foreground(lipgloss.Color(c))
}

// SeriesColor returns the hex color for the i-th series, cycling the palette.
func (t Theme) SeriesColor(i int) string {
	return t.Series[i%len(t.Series)]
}

// ResolveColor maps a semantic or Grafana color name to a theme hex color.
// Unknown names pass through unchanged (assumed to be hex already).
func (t Theme) ResolveColor(name string) string {
	switch name {
	case "green", "success", "ok":
		return t.Success
	case "red", "critical", "error":
		return t.Error
	case "yellow", "orange", "amber", "warning":
		return t.Warning
	case "blue", "accent":
		return t.Accent
	case "text":
		return t.Label
	case "gray", "grey", "dim":
		return t.Dim
	}
	return name
}

// Built-in themes.
var (
	// Pilot is the muted terminal aesthetic from the Pilot dashboard.
	Pilot = Theme{
		Name:    "pilot",
		Accent:  "#7eb8da", // steel blue
		Success: "#7ec699", // sage green
		Error:   "#d48a8a", // dusty rose
		Warning: "#d4a054", // amber
		Border:  "#3d4450", // slate
		Label:   "#c9d1d9",
		Dim:     "#8b949e",
		DimMore: "#6e7681",
		Series:  []string{"#7eb8da", "#7ec699", "#d4a054", "#bb9af7", "#d48a8a", "#7dcfff"},
	}

	// TokyoNight matches the popular tokyonight palette.
	TokyoNight = Theme{
		Name:    "tokyo-night",
		Accent:  "#7aa2f7",
		Success: "#9ece6a",
		Error:   "#f7768e",
		Warning: "#e0af68",
		Border:  "#3b4261",
		Label:   "#c0caf5",
		Dim:     "#565f89",
		DimMore: "#414868",
		Series:  []string{"#7aa2f7", "#9ece6a", "#e0af68", "#bb9af7", "#f7768e", "#7dcfff"},
	}

	// CatppuccinMocha matches catppuccin-mocha.
	CatppuccinMocha = Theme{
		Name:    "catppuccin-mocha",
		Accent:  "#89b4fa",
		Success: "#a6e3a1",
		Error:   "#f38ba8",
		Warning: "#f9e2af",
		Border:  "#45475a",
		Label:   "#cdd6f4",
		Dim:     "#6c7086",
		DimMore: "#585b70",
		Series:  []string{"#89b4fa", "#a6e3a1", "#f9e2af", "#cba6f7", "#f38ba8", "#94e2d5"},
	}
)

// All lists the built-in themes.
var All = []Theme{Pilot, TokyoNight, CatppuccinMocha}

// ByName returns the named theme, falling back to Pilot.
func ByName(name string) Theme {
	for _, t := range All {
		if t.Name == name {
			return t
		}
	}
	return Pilot
}
