// Package datasource defines the boundary grom's widgets query against. The
// interface lives here (the consumer side); concrete backends such as
// datasource/prom implement it.
package datasource

import (
	"context"
	"time"

	"github.com/qf-studio/grom/pkg/tui/widget"
)

// Instant requests a single-value (vector) query evaluated at At. A zero At
// means "server now".
type Instant struct {
	Expr   string
	Legend string // Grafana-style {{label}} template; empty → derived from labels
	At     time.Time
}

// Range requests a query_range over [Start, End] at the given Step.
type Range struct {
	Expr   string
	Legend string
	Start  time.Time
	End    time.Time
	Step   time.Duration
}

// Datasource fetches series for widgets. Each returned Series carries an
// expanded legend and the points for one result element.
type Datasource interface {
	// QueryInstant evaluates an instant vector query, returning one Series
	// (with a single Point) per result element.
	QueryInstant(ctx context.Context, q Instant) ([]widget.Series, error)
	// QueryRange evaluates a range query, returning one Series per element.
	QueryRange(ctx context.Context, q Range) ([]widget.Series, error)
}
