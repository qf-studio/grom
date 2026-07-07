package widget

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/qf-studio/grot/pkg/tui/render"
	"github.com/qf-studio/grot/pkg/tui/theme"
)

// TimeSeries renders series as a block area chart with y-axis labels and a
// legend row. (Braille rendering lands in Phase 3; the block chart is the
// --ascii fallback and the Phase 1/2 default.)
type TimeSeries struct {
	data
	title    string
	Unit     string
	Decimals *int
	Stacked  bool
}

// NewTimeSeries creates a time-series chart widget.
func NewTimeSeries(title, unit string) *TimeSeries {
	return &TimeSeries{title: title, Unit: unit}
}

func (t *TimeSeries) Title() string       { return t.title }
func (t *TimeSeries) MinSize() (int, int) { return 24, 6 }

func (t *TimeSeries) Render(w, h int, th theme.Theme, focused bool) string {
	iw, ih := render.InnerSize(w, h)
	ps := t.panelStyle(th, focused)

	var body string
	switch {
	case t.err != nil:
		body = errorBody(t.err, iw, ih, th)
	case len(t.res.Series) == 0 || len(t.res.Series[0].Points) == 0:
		body = noDataBody(iw, ih, th)
	default:
		body = t.body(iw, ih, th)
	}
	return render.Panel(t.title, body, w, h, ps)
}

func (t *TimeSeries) body(iw, ih int, th theme.Theme) string {
	// Layout: chart rows + legend row (if ≥2 series or room permits).
	legendRows := 0
	if ih >= 4 {
		legendRows = 1
	}
	chartRows := ih - legendRows
	if chartRows < 1 {
		chartRows = 1
		legendRows = 0
	}

	// Primary series drives the chart (multi-series overlay lands with
	// braille in Phase 3; block chart shows series[0], legend shows all).
	primary := t.res.Series[0]
	vals := make([]float64, len(primary.Points))
	minV, maxV := primary.Points[0].V, primary.Points[0].V
	for i, p := range primary.Points {
		vals[i] = p.V
		if p.V < minV {
			minV = p.V
		}
		if p.V > maxV {
			maxV = p.V
		}
	}

	// Y-axis label gutter.
	hiLabel := FormatValue(maxV, t.Unit, t.Decimals)
	loLabel := FormatValue(minV, t.Unit, t.Decimals)
	gutter := len(hiLabel)
	if len(loLabel) > gutter {
		gutter = len(loLabel)
	}
	if gutter > iw/3 {
		gutter = iw / 3
	}
	chartW := iw - gutter - 1
	if chartW < 4 {
		chartW = iw
		gutter = 0
	}

	rows := render.BlockChart(vals, chartW, chartRows)
	seriesStyle := th.SeriesStyle(0)

	lines := make([]string, 0, ih)
	for i, row := range rows {
		label := strings.Repeat(" ", gutter)
		if gutter > 0 {
			switch i {
			case 0:
				label = render.PadOrTruncate(hiLabel, gutter)
			case len(rows) - 1:
				label = render.PadOrTruncate(loLabel, gutter)
			}
		}
		sep := ""
		if gutter > 0 {
			sep = " "
		}
		lines = append(lines, th.DimStyle().Render(label)+sep+seriesStyle.Render(row))
	}

	if legendRows > 0 {
		lines = append(lines, t.legend(iw, th))
	}

	return strings.Join(lines, "\n")
}

// legend renders "● legend1  ● legend2 ..." with per-series colors.
func (t *TimeSeries) legend(iw int, th theme.Theme) string {
	parts := make([]string, 0, len(t.res.Series))
	for i, s := range t.res.Series {
		name := s.Legend
		if name == "" {
			name = "series"
		}
		dot := th.SeriesStyle(i).Render("●")
		parts = append(parts, dot+" "+th.DimStyle().Render(name))
	}
	line := strings.Join(parts, "  ")
	if lipgloss.Width(line) > iw {
		line = render.TruncateVisual(line, iw)
	}
	return line
}
