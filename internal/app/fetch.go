package app

import (
	"context"
	"time"

	"github.com/qf-studio/grom/internal/config"
	"github.com/qf-studio/grom/internal/datasource"
	"github.com/qf-studio/grom/pkg/tui/render"
	"github.com/qf-studio/grom/pkg/tui/widget"
)

// FetchWidget runs every query in spec and returns the combined result.
// Time-series queries go through query_range (windowed [now-rng, now] at a step
// sized to chartW); everything else is an instant query. chartW is the widget's
// inner chart width in cells — used only to size the range step.
func FetchWidget(ctx context.Context, ds datasource.Datasource, spec config.WidgetSpec, chartW int, rng time.Duration) (widget.QueryResult, error) {
	var all []widget.Series
	for _, q := range spec.Queries {
		var (
			s   []widget.Series
			err error
		)
		if useRange(spec, q) {
			end := time.Now()
			s, err = ds.QueryRange(ctx, datasource.Range{
				Expr:   q.Expr,
				Legend: q.Legend,
				Start:  end.Add(-rng),
				End:    end,
				Step:   StepFor(rng, chartW),
			})
		} else {
			s, err = ds.QueryInstant(ctx, datasource.Instant{Expr: q.Expr, Legend: q.Legend})
		}
		if err != nil {
			return widget.QueryResult{}, err
		}
		all = append(all, s...)
	}
	return widget.QueryResult{Series: all, FetchedAt: time.Now()}, nil
}

// FetchAll fetches every widget synchronously, assigning each its result or
// error. A failing query only marks its own widget — the grid stays intact.
// Used by the one-shot (--once) path; the TUI fetches concurrently instead.
func FetchAll(ctx context.Context, ds datasource.Datasource, dash *config.Dashboard, widgets []widget.Widget, rects []Rect, timeout time.Duration) {
	rng := dash.Range.Duration()
	for i, spec := range dash.Widgets {
		cctx, cancel := context.WithTimeout(ctx, timeout)
		res, err := FetchWidget(cctx, ds, spec, chartWidth(rects[i]), rng)
		cancel()
		if err != nil {
			widgets[i].SetError(err)
			continue
		}
		widgets[i].SetResult(res)
	}
}

// useRange decides whether a query runs as query_range. Time-series panels need
// a range to draw a line; stat panels opt in via Sparkline (value + trend band).
// Other single-value panels reduce to one point and stay instant. A query may
// opt back into instant via Instant.
func useRange(spec config.WidgetSpec, q config.Query) bool {
	if q.Instant {
		return false
	}
	switch spec.Type {
	case config.TypeTimeSeries:
		return true
	case config.TypeStat:
		return spec.Sparkline
	}
	return false
}

// chartWidth returns the inner chart width for a cell rect — the value
// FetchWidget wants for range-step sizing.
func chartWidth(r Rect) int {
	w, _ := render.InnerSize(r.W, r.H)
	return w
}
