// Package grafana imports a Grafana dashboard JSON export into grot's canonical
// config.Dashboard. It maps the panel subset grot renders (stat, gauge,
// bargauge, timeseries) 1:1, promotes row children, and represents everything
// else as a placeholder in its original grid slot, collecting warnings.
package grafana

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/qf-studio/grot/internal/config"
)

// Import reads and converts a Grafana dashboard JSON file.
func Import(path string) (*config.Dashboard, []string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read grafana json %q: %w", path, err)
	}
	return Convert(b)
}

// Convert converts Grafana dashboard JSON bytes into a normalized Dashboard,
// returning any non-fatal warnings (unsupported panels, dropped targets).
func Convert(b []byte) (*config.Dashboard, []string, error) {
	var rf rawFile
	if err := json.Unmarshal(b, &rf); err != nil {
		return nil, nil, fmt.Errorf("parse grafana json: %w", err)
	}
	g := rf.pick()

	dash := &config.Dashboard{Title: g.Title}
	if d, err := time.ParseDuration(g.Refresh); err == nil {
		dash.Refresh = config.Duration(d)
	}
	if rng, ok := parseNowRange(g.Time.From); ok {
		dash.Range = config.Duration(rng)
	}

	var warnings []string
	if g.Time.From != "" {
		if _, ok := parseNowRange(g.Time.From); !ok {
			warnings = append(warnings, fmt.Sprintf("time range %q not supported (only now-<dur>); using default", g.Time.From))
		}
	}

	for _, p := range flatten(g.Panels) {
		spec, warn := convertPanel(p)
		if warn != "" {
			warnings = append(warnings, warn)
		}
		if spec != nil {
			dash.Widgets = append(dash.Widgets, *spec)
		}
	}

	if err := dash.Normalize(); err != nil {
		return nil, warnings, err
	}
	return dash, warnings, nil
}

// flatten drops row panels but promotes their nested children (collapsed rows
// carry their panels inline), preserving document order.
func flatten(panels []rawPanel) []rawPanel {
	out := make([]rawPanel, 0, len(panels))
	for _, p := range panels {
		if p.Type == "row" {
			out = append(out, flatten(p.Panels)...)
			continue
		}
		out = append(out, p)
	}
	return out
}

func convertPanel(p rawPanel) (*config.WidgetSpec, string) {
	grid := config.GridPos{X: p.GridPos.X, Y: p.GridPos.Y, W: p.GridPos.W, H: p.GridPos.H}

	wt, ok := mapType(p.Type)
	if !ok {
		title := fmt.Sprintf("unsupported: %s", p.Type)
		return &config.WidgetSpec{Type: config.TypePlaceholder, Title: title, Grid: grid},
			fmt.Sprintf("panel %q: unsupported type %q → placeholder", p.Title, p.Type)
	}

	def := p.FieldConfig.Defaults
	spec := &config.WidgetSpec{
		Type:       wt,
		Title:      p.Title,
		Grid:       grid,
		Unit:       def.Unit,
		Decimals:   def.Decimals,
		Min:        def.Min,
		Max:        def.Max,
		Thresholds: convertThresholds(def.Thresholds.Steps),
		Stacked:    isStacked(def.Custom.Stacking.Mode),
		Reduce:     firstCalc(p.Options.ReduceOptions.Calcs),
		Queries:    convertTargets(p.Targets),
	}
	if len(spec.Queries) == 0 {
		return spec, fmt.Sprintf("panel %q: no usable targets", p.Title)
	}
	return spec, ""
}

// mapType maps a Grafana panel type to a grot widget type.
func mapType(t string) (config.WidgetType, bool) {
	switch t {
	case "stat":
		return config.TypeStat, true
	case "gauge":
		return config.TypeGauge, true
	case "bargauge":
		return config.TypeBarGauge, true
	case "timeseries":
		return config.TypeTimeSeries, true
	default:
		return "", false
	}
}

func convertTargets(targets []rawTarget) []config.Query {
	var qs []config.Query
	for _, t := range targets {
		if strings.TrimSpace(t.Expr) == "" {
			continue // template/expression rows with no PromQL
		}
		qs = append(qs, config.Query{Expr: t.Expr, Legend: t.LegendFormat, Instant: t.Instant})
	}
	return qs
}

func convertThresholds(steps []rawStep) []config.Threshold {
	if len(steps) == 0 {
		return nil
	}
	out := make([]config.Threshold, 0, len(steps))
	for _, s := range steps {
		out = append(out, config.Threshold{Value: s.Value, Color: s.Color})
	}
	return out
}

func isStacked(mode string) bool {
	return mode != "" && mode != "none"
}

func firstCalc(calcs []string) string {
	if len(calcs) == 0 {
		return ""
	}
	return calcs[0]
}

// parseNowRange parses a Grafana relative range origin ("now-6h", "now-15m")
// into the window duration. "now" (zero window) and absolute/other forms are
// unsupported.
func parseNowRange(from string) (time.Duration, bool) {
	const prefix = "now-"
	if !strings.HasPrefix(from, prefix) {
		return 0, false
	}
	d, err := time.ParseDuration(strings.TrimPrefix(from, prefix))
	if err != nil || d <= 0 {
		return 0, false
	}
	return d, true
}
