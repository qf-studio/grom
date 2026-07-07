package render

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Meter renders a horizontal gauge of exactly width cells:
//
//	████████▓░░░░░░░
//
// frac is clamped to [0,1]. fill styles the solid portion, empty the rest.
// A ▓ shade cell marks the boundary for a gradient feel (only when partially
// filled).
func Meter(frac float64, width int, fill, empty lipgloss.Style) string {
	if width <= 0 {
		return ""
	}
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	filled := int(frac * float64(width))
	if filled > width {
		filled = width
	}

	var b strings.Builder
	if filled > 0 {
		solid := filled
		edge := false
		if filled < width && filled > 1 {
			solid = filled - 1
			edge = true
		}
		b.WriteString(fill.Render(strings.Repeat("█", solid)))
		if edge {
			b.WriteString(fill.Render("▓"))
		}
	}
	if width-filled > 0 {
		b.WriteString(empty.Render(strings.Repeat("░", width-filled)))
	}
	return b.String()
}

// GradientMeter renders a horizontal gauge whose filled cells sweep through
// a color gradient left→right, btop-style:
//
//	████████████▓░░░░░░
//
// stops are hex colors (e.g. success→warning→error); the empty track uses
// the empty style.
func GradientMeter(frac float64, width int, stops []string, empty lipgloss.Style) string {
	if width <= 0 {
		return ""
	}
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	filled := int(frac * float64(width))
	if filled > width {
		filled = width
	}
	styles := GradientStyles(stops, width)

	var b strings.Builder
	for i := 0; i < filled; i++ {
		ch := "█"
		if i == filled-1 && filled < width {
			ch = "▓"
		}
		b.WriteString(styles[i].Render(ch))
	}
	if width-filled > 0 {
		b.WriteString(empty.Render(strings.Repeat("░", width-filled)))
	}
	return b.String()
}

// HBar renders a horizontal bar (no empty track) of up to width cells with a
// half-block cap for sub-cell resolution.
func HBar(frac float64, width int, fill lipgloss.Style) string {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	cells := frac * float64(width)
	full := int(cells)
	half := cells-float64(full) >= 0.5 && full < width
	s := strings.Repeat("█", full)
	if half {
		s += "▌"
	}
	return fill.Render(s)
}
