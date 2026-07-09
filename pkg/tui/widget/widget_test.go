package widget

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/qf-studio/grom/pkg/tui/theme"
)

func result(vals ...float64) QueryResult {
	now := time.Now()
	pts := make([]Point, len(vals))
	for i, v := range vals {
		pts[i] = Point{T: now.Add(time.Duration(i) * time.Minute), V: v}
	}
	return QueryResult{Series: []Series{{Legend: "test", Points: pts}}, FetchedAt: now}
}

// Every widget must render exactly w×h cells in every state.
func TestWidgetsExactDimensions(t *testing.T) {
	th := theme.Pilot
	widgets := []Widget{
		NewStat("Stat", "short"),
		NewGauge("Gauge", "percent", 0, 100),
		NewBarGauge("Bars", "short"),
		NewTimeSeries("Chart", "s"),
	}
	states := []struct {
		name  string
		apply func(w Widget)
	}{
		{"no data", func(w Widget) { w.SetResult(QueryResult{}) }},
		{"single point", func(w Widget) { w.SetResult(result(42)) }},
		{"range", func(w Widget) { w.SetResult(result(1, 5, 3, 8, 2)) }},
		{"error", func(w Widget) {
			w.SetError(errors.New("connection refused: very long error message that should truncate"))
		}},
	}
	sizes := []struct{ w, h int }{{20, 4}, {30, 6}, {50, 12}, {80, 8}}

	for _, wg := range widgets {
		for _, st := range states {
			st.apply(wg)
			for _, sz := range sizes {
				for _, focused := range []bool{false, true} {
					out := wg.Render(sz.w, sz.h, th, focused)
					lines := strings.Split(out, "\n")
					if len(lines) != sz.h {
						t.Errorf("%s/%s %dx%d: %d lines", wg.Title(), st.name, sz.w, sz.h, len(lines))
						continue
					}
					for i, line := range lines {
						if got := lipgloss.Width(line); got != sz.w {
							t.Errorf("%s/%s %dx%d line %d: width %d", wg.Title(), st.name, sz.w, sz.h, i, got)
						}
					}
				}
			}
		}
	}
}

func TestThresholdColor(t *testing.T) {
	th := theme.Pilot
	fv := func(v float64) *float64 { return &v }
	thresholds := []Threshold{
		{Color: "red"}, {Value: fv(70), Color: "yellow"}, {Value: fv(90), Color: "green"},
	}
	tests := []struct {
		v    float64
		want string
	}{
		{50, th.Error}, {66, th.Error}, {75, th.Warning}, {95, th.Success},
	}
	for _, tt := range tests {
		if got := thresholdColor(tt.v, thresholds, th, th.Label); got != tt.want {
			t.Errorf("thresholdColor(%v) = %q, want %q", tt.v, got, tt.want)
		}
	}
}

// Threshold resolution must not depend on slice order — Grafana exports steps
// sorted ascending, hand-written YAML may not be.
func TestThresholdColorUnsorted(t *testing.T) {
	th := theme.Pilot
	fv := func(v float64) *float64 { return &v }
	// Same steps as TestThresholdColor, deliberately shuffled with base last.
	thresholds := []Threshold{
		{Value: fv(90), Color: "green"}, {Value: fv(70), Color: "yellow"}, {Color: "red"},
	}
	tests := []struct {
		v    float64
		want string
	}{
		{50, th.Error}, {66, th.Error}, {75, th.Warning}, {95, th.Success},
	}
	for _, tt := range tests {
		if got := thresholdColor(tt.v, thresholds, th, th.Label); got != tt.want {
			t.Errorf("thresholdColor(%v) = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		v    float64
		unit string
		want string
	}{
		{66, "percent", "66%"},
		{0.665, "percentunit", "66.5%"},
		{154.234, "currencyUSD", "$154.23"},
		{465.24, "s", "7.8m"},
		{0.5, "s", "500ms"},
		{0.000002, "s", "2µs"}, // non-ASCII unit: width math must be visual, not len()
		{57300, "short", "57.3K"},
		{3, "", "3"},
	}
	for _, tt := range tests {
		if got := FormatValue(tt.v, tt.unit, nil); got != tt.want {
			t.Errorf("FormatValue(%v, %q) = %q, want %q", tt.v, tt.unit, got, tt.want)
		}
	}
}

func multiResult() QueryResult {
	now := time.Now()
	mk := func(legend string, vals ...float64) Series {
		pts := make([]Point, len(vals))
		for i, v := range vals {
			pts[i] = Point{T: now.Add(time.Duration(i) * time.Minute), V: v}
		}
		return Series{Legend: legend, Points: pts}
	}
	return QueryResult{
		Series:    []Series{mk("a", 10, 20, 30), mk("b", 5, 15, 25)},
		FetchedAt: now,
	}
}

// Multi-series mode wiring: braille overlay (Stacked=false, the default) must
// render differently from stacked; solid mode has no overlay primitive, so
// both flag values render identically there. Overlay output must still be
// exactly w×h.
func TestTimeSeriesMultiSeriesModes(t *testing.T) {
	th := theme.Pilot
	renderTS := func(stacked, solid bool) string {
		w := NewTimeSeries("Chart", "short")
		w.Stacked = stacked
		w.Solid = solid
		w.SetResult(multiResult())
		return w.Render(50, 8, th, false)
	}

	if renderTS(false, false) == renderTS(true, false) {
		t.Error("braille overlay and stacked renders identical — Stacked flag not honored")
	}
	if renderTS(false, true) != renderTS(true, true) {
		t.Error("solid mode: overlay and stacked must render identically (no block overlay primitive)")
	}

	lines := strings.Split(renderTS(false, false), "\n")
	if len(lines) != 8 {
		t.Fatalf("overlay: %d lines, want 8", len(lines))
	}
	for i, line := range lines {
		if got := lipgloss.Width(line); got != 50 {
			t.Errorf("overlay line %d: width %d, want 50", i, got)
		}
	}
}
