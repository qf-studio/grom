package render

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/lipgloss"
)

// LerpHex linearly interpolates two hex colors ("#rrggbb") at t in [0,1].
func LerpHex(a, b string, t float64) string {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	ar, ag, ab := parseHex(a)
	br, bg, bb := parseHex(b)
	r := int(float64(ar) + (float64(br)-float64(ar))*t)
	g := int(float64(ag) + (float64(bg)-float64(ag))*t)
	bl := int(float64(ab) + (float64(bb)-float64(ab))*t)
	return fmt.Sprintf("#%02x%02x%02x", r, g, bl)
}

// Gradient returns n colors sweeping through stops (≥1). With one stop the
// gradient runs from a dimmed variant of the stop to the stop itself —
// btop's single-color graph look.
func Gradient(stops []string, n int) []string {
	if n <= 0 {
		return nil
	}
	if len(stops) == 0 {
		stops = []string{"#ffffff"}
	}
	if len(stops) == 1 {
		stops = []string{Dim(stops[0], 0.55), stops[0]}
	}
	out := make([]string, n)
	if n == 1 {
		out[0] = stops[len(stops)-1]
		return out
	}
	segs := len(stops) - 1
	for i := 0; i < n; i++ {
		t := float64(i) / float64(n-1) * float64(segs)
		seg := int(t)
		if seg >= segs {
			seg = segs - 1
		}
		out[i] = LerpHex(stops[seg], stops[seg+1], t-float64(seg))
	}
	return out
}

// GradientStyles returns lipgloss foreground styles for a Gradient.
func GradientStyles(stops []string, n int) []lipgloss.Style {
	colors := Gradient(stops, n)
	styles := make([]lipgloss.Style, n)
	for i, c := range colors {
		styles[i] = lipgloss.NewStyle().Foreground(lipgloss.Color(c))
	}
	return styles
}

// Dim scales a hex color toward black by factor (0 = black, 1 = unchanged).
func Dim(hex string, factor float64) string {
	r, g, b := parseHex(hex)
	return fmt.Sprintf("#%02x%02x%02x",
		int(float64(r)*factor), int(float64(g)*factor), int(float64(b)*factor))
}

func parseHex(s string) (int, int, int) {
	if len(s) == 7 && s[0] == '#' {
		r, _ := strconv.ParseInt(s[1:3], 16, 32)
		g, _ := strconv.ParseInt(s[3:5], 16, 32)
		b, _ := strconv.ParseInt(s[5:7], 16, 32)
		return int(r), int(g), int(b)
	}
	return 128, 128, 128
}
