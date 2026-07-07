package render

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestPanelExactDimensions(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		content string
		w, h    int
	}{
		{"basic", "Title", "hello", 30, 5},
		{"empty content", "T", "", 20, 4},
		{"overflow content", "Long", strings.Repeat("line\n", 20), 25, 6},
		{"long title", strings.Repeat("VERYLONGTITLE", 5), "x", 20, 4},
		{"styled content", "S", lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Render("red text"), 30, 4},
	}
	ps := PanelStyle{Border: lipgloss.NewStyle(), Title: lipgloss.NewStyle()}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := Panel(tt.title, tt.content, tt.w, tt.h, ps)
			lines := strings.Split(out, "\n")
			if len(lines) != tt.h {
				t.Fatalf("got %d lines, want %d", len(lines), tt.h)
			}
			for i, line := range lines {
				if got := lipgloss.Width(line); got != tt.w {
					t.Errorf("line %d: width %d, want %d: %q", i, got, tt.w, line)
				}
			}
		})
	}
}

func TestPadOrTruncate(t *testing.T) {
	tests := []struct {
		in    string
		width int
	}{
		{"short", 20},
		{"exactly-ten", 11},
		{"this is a much longer string than allowed", 15},
		{"", 5},
	}
	for _, tt := range tests {
		out := PadOrTruncate(tt.in, tt.width)
		if got := lipgloss.Width(out); got != tt.width {
			t.Errorf("PadOrTruncate(%q, %d): width %d", tt.in, tt.width, got)
		}
	}
}

func TestTruncateVisualStyled(t *testing.T) {
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("#7eb8da")).Render("styled long content here")
	out := TruncateVisual(styled, 12)
	if got := lipgloss.Width(out); got != 12 {
		t.Errorf("styled truncate: width %d, want 12", got)
	}
}

func TestNormalizeSparkline(t *testing.T) {
	// Levels are 1-8, right-aligned, zeros pad left.
	levels := NormalizeSparkline([]float64{0, 5, 10}, 5)
	if len(levels) != 5 {
		t.Fatalf("len %d", len(levels))
	}
	if levels[0] != 0 || levels[1] != 0 {
		t.Errorf("left padding not zero: %v", levels)
	}
	if levels[2] != 1 {
		t.Errorf("zero value should be baseline 1: %v", levels)
	}
	if levels[4] != 8 {
		t.Errorf("max value should be 8: %v", levels)
	}
}

func TestSparklineWidth(t *testing.T) {
	for _, w := range []int{5, 20, 60} {
		out := Sparkline(NormalizeSparkline([]float64{1, 2, 3}, w), w)
		if got := lipgloss.Width(out); got != w {
			t.Errorf("width %d: got %d", w, got)
		}
	}
}

func TestBlockChartDimensions(t *testing.T) {
	vals := []float64{1, 5, 3, 8, 2, 9, 4}
	rows := BlockChart(vals, 20, 4)
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4", len(rows))
	}
	for i, r := range rows {
		if got := lipgloss.Width(r); got != 20 {
			t.Errorf("row %d: width %d, want 20", i, got)
		}
	}
	// Max value column must reach the top row with a filled block.
	if !strings.ContainsAny(rows[0], "▁▂▃▄▅▆▇█") {
		t.Errorf("top row empty for max value: %q", rows[0])
	}
}

func TestMeterWidth(t *testing.T) {
	fill := lipgloss.NewStyle()
	empty := lipgloss.NewStyle()
	for _, frac := range []float64{-0.5, 0, 0.33, 0.5, 0.875, 1, 1.5} {
		out := Meter(frac, 16, fill, empty)
		if got := lipgloss.Width(out); got != 16 {
			t.Errorf("frac %v: width %d, want 16", frac, got)
		}
	}
}

func TestFormatCompact(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{0, "0"}, {999, "999"}, {1000, "1.0K"}, {57300, "57.3K"}, {1_200_000, "1.2M"},
	}
	for _, tt := range tests {
		if got := FormatCompact(tt.in); got != tt.want {
			t.Errorf("FormatCompact(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestTextHelpersNonPositiveWidth guards against the negative-Repeat panic that
// surfaced when a widget was laid out smaller than its chrome during a resize.
func TestTextHelpersNonPositiveWidth(t *testing.T) {
	for _, w := range []int{0, -1, -4, -100} {
		if got := TruncateVisual("no data", w); got != "" {
			t.Errorf("TruncateVisual(_, %d) = %q, want empty", w, got)
		}
		if got := Center("no data", w); got != "" {
			t.Errorf("Center(_, %d) = %q, want empty", w, got)
		}
		if got := PadOrTruncate("no data", w); got != "" {
			t.Errorf("PadOrTruncate(_, %d) = %q, want empty", w, got)
		}
	}
}
