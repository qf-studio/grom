// Package prom implements a Prometheus-backed datasource.Datasource using the
// official client_golang API package for query decoding.
package prom

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"github.com/qf-studio/grot/internal/datasource"
	"github.com/qf-studio/grot/pkg/tui/widget"
)

// Client queries a Prometheus server.
type Client struct {
	api promv1.API
}

// New creates a client for the given base URL (e.g. http://localhost:9090).
func New(addr string) (*Client, error) {
	c, err := promapi.NewClient(promapi.Config{Address: addr})
	if err != nil {
		return nil, fmt.Errorf("prometheus client %q: %w", addr, err)
	}
	return &Client{api: promv1.NewAPI(c)}, nil
}

// Interface compliance.
var _ datasource.Datasource = (*Client)(nil)

// QueryInstant runs an instant vector query.
func (c *Client) QueryInstant(ctx context.Context, q datasource.Instant) ([]widget.Series, error) {
	ts := q.At
	if ts.IsZero() {
		ts = time.Now()
	}
	val, _, err := c.api.Query(ctx, q.Expr, ts)
	if err != nil {
		return nil, fmt.Errorf("instant query %q: %w", q.Expr, err)
	}
	vec, ok := val.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("instant query %q: expected vector, got %s", q.Expr, val.Type())
	}
	series := make([]widget.Series, 0, len(vec))
	for _, s := range vec {
		v := float64(s.Value)
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue // non-finite (e.g. x/0 → +Inf) is "no data", not a value
		}
		series = append(series, widget.Series{
			Legend: expandLegend(q.Legend, s.Metric),
			Points: []widget.Point{{T: s.Timestamp.Time(), V: v}},
		})
	}
	return series, nil
}

// QueryRange runs a range query.
func (c *Client) QueryRange(ctx context.Context, q datasource.Range) ([]widget.Series, error) {
	r := promv1.Range{Start: q.Start, End: q.End, Step: q.Step}
	val, _, err := c.api.QueryRange(ctx, q.Expr, r)
	if err != nil {
		return nil, fmt.Errorf("range query %q: %w", q.Expr, err)
	}
	mat, ok := val.(model.Matrix)
	if !ok {
		return nil, fmt.Errorf("range query %q: expected matrix, got %s", q.Expr, val.Type())
	}
	series := make([]widget.Series, 0, len(mat))
	for _, ss := range mat {
		// NaN samples are kept as-is: for a range they mark a gap at a real
		// timestamp, so the chart preserves the time axis (dropping them would
		// collapse sparse series onto one edge). Renderers skip NaN dots.
		// ±Inf (e.g. x/0) is normalized to NaN — a gap, not a chart-breaking value.
		pts := make([]widget.Point, 0, len(ss.Values))
		for _, p := range ss.Values {
			v := float64(p.Value)
			if math.IsInf(v, 0) {
				v = math.NaN()
			}
			pts = append(pts, widget.Point{T: p.Timestamp.Time(), V: v})
		}
		series = append(series, widget.Series{
			Legend: expandLegend(q.Legend, ss.Metric),
			Points: pts,
		})
	}
	return series, nil
}

// expandLegend fills a Grafana-style {{label}} template from a metric's labels.
// An empty template derives a default from the metric's identity.
func expandLegend(tmpl string, metric model.Metric) string {
	if strings.TrimSpace(tmpl) == "" {
		return defaultLegend(metric)
	}
	var b strings.Builder
	for {
		i := strings.Index(tmpl, "{{")
		if i < 0 {
			b.WriteString(tmpl)
			break
		}
		b.WriteString(tmpl[:i])
		rest := tmpl[i+2:]
		j := strings.Index(rest, "}}")
		if j < 0 {
			b.WriteString(tmpl[i:]) // unterminated placeholder — emit literally
			break
		}
		label := strings.TrimSpace(rest[:j])
		b.WriteString(string(metric[model.LabelName(label)]))
		tmpl = rest[j+2:]
	}
	return b.String()
}

// defaultLegend returns the metric name when it is the only distinguishing
// label, otherwise the full Prometheus label set (which is unique per series).
func defaultLegend(metric model.Metric) string {
	name := string(metric[model.MetricNameLabel])
	if len(metric) <= 1 {
		return name
	}
	return metric.String()
}
