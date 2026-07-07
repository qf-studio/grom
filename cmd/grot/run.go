package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/qf-studio/grot/internal/app"
	"github.com/qf-studio/grot/internal/config"
	"github.com/qf-studio/grot/internal/datasource"
	"github.com/qf-studio/grot/internal/datasource/prom"
	"github.com/qf-studio/grot/pkg/tui/render"
	"github.com/qf-studio/grot/pkg/tui/theme"
	"github.com/qf-studio/grot/pkg/tui/widget"
)

func runCmd() *cobra.Command {
	var (
		cfgPath   string
		promAddr  string
		themeName string
		watch     bool
		ascii     bool
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Render a dashboard config against a live Prometheus",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dash, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			ds, err := prom.New(promAddr)
			if err != nil {
				return err
			}
			th := theme.ByName(coalesce(themeName, dash.Theme))

			widgets := make([]widget.Widget, len(dash.Widgets))
			for i, spec := range dash.Widgets {
				w, err := app.BuildWidget(spec)
				if err != nil {
					return fmt.Errorf("widget %d: %w", i, err)
				}
				if ascii {
					if ts, ok := w.(*widget.TimeSeries); ok {
						ts.Solid = true
					}
				}
				widgets[i] = w
			}

			refresh := dash.Refresh.Duration()
			renderFrame := func(ctx context.Context) string {
				// Lay out once per tick so fetch (range-query step sizing) and
				// draw share the same cell widths, and both track a resize.
				lay := planLayout(dash.Widgets, termWidth())
				fetchAll(ctx, ds, dash, widgets, lay, refresh)
				return frame(dash, widgets, th, lay)
			}

			if !watch {
				fmt.Println(renderFrame(cmd.Context()))
				return nil
			}
			return watchLoop(cmd.Context(), refresh, renderFrame)
		},
	}

	cmd.Flags().StringVarP(&cfgPath, "config", "c", "", "dashboard YAML config (required)")
	cmd.Flags().StringVar(&promAddr, "prom", "http://localhost:9090", "Prometheus base URL")
	cmd.Flags().StringVar(&themeName, "theme", "", "override theme (default: config theme)")
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "refresh on the config interval until Ctrl-C")
	cmd.Flags().BoolVar(&ascii, "ascii", false, "block-character charts instead of braille")
	_ = cmd.MarkFlagRequired("config")
	return cmd
}

// fetchAll runs each widget's queries and delivers results (or errors). A slow
// or failing query only affects its own widget — the grid never breaks. The
// layout supplies each widget's cell size so range queries can size their step
// to the chart width.
func fetchAll(ctx context.Context, ds datasource.Datasource, dash *config.Dashboard, widgets []widget.Widget, lay layout, refresh time.Duration) {
	timeout := 10 * time.Second
	if refresh > 0 && refresh < timeout {
		timeout = refresh
	}
	rng := dash.Range.Duration()
	for i, spec := range dash.Widgets {
		chartW, _ := render.InnerSize(lay.cells[i].W, lay.cells[i].H)
		cctx, cancel := context.WithTimeout(ctx, timeout)
		res, err := fetchWidget(cctx, ds, spec, chartW, rng)
		cancel()
		if err != nil {
			widgets[i].SetError(err)
			continue
		}
		widgets[i].SetResult(res)
	}
}

func fetchWidget(ctx context.Context, ds datasource.Datasource, spec config.WidgetSpec, chartW int, rng time.Duration) (widget.QueryResult, error) {
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
				Step:   app.StepFor(rng, chartW),
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

// useRange decides whether a query runs as query_range. Time-series panels need
// a range to draw a line; single-value panels (stat/gauge/bargauge) reduce to
// one point and stay instant. A query may opt back into instant via Instant.
func useRange(spec config.WidgetSpec, q config.Query) bool {
	return spec.Type == config.TypeTimeSeries && !q.Instant
}

// frame renders the header plus the packed grid described by lay.
func frame(dash *config.Dashboard, widgets []widget.Widget, th theme.Theme, lay layout) string {
	header := th.AccentStyle().Bold(true).Render(" "+coalesce(dash.Title, "grot")) +
		th.DimStyle().Render("  ·  "+time.Now().Format("15:04:05")+"  ·  "+th.Name)
	return header + "\n" + packRows(widgets, th, lay)
}

// cellPlan is one widget's outer cell size within the packed grid.
type cellPlan struct{ W, H int }

// layout is the resolved placement for a frame: per-widget cell sizes plus the
// grouping of widget indices into rows. Computed once per tick so fetch and
// draw agree on widths.
type layout struct {
	cells []cellPlan
	rows  [][]int
}

// planLayout packs widgets left-to-right into rows, even-splitting each row's
// width. Phase 3 uses this simple split; the 24-column grid engine arrives in
// Phase 4.
func planLayout(specs []config.WidgetSpec, termW int) layout {
	perRow := termW / 34
	if perRow < 1 {
		perRow = 1
	}
	if perRow > 4 {
		perRow = 4
	}

	lay := layout{cells: make([]cellPlan, len(specs))}
	for start := 0; start < len(specs); start += perRow {
		end := start + perRow
		if end > len(specs) {
			end = len(specs)
		}
		n := end - start
		colW := termW / n
		rowH := 0
		for i := start; i < end; i++ {
			if h := typeHeight(specs[i].Type); h > rowH {
				rowH = h
			}
		}
		row := make([]int, 0, n)
		for i := start; i < end; i++ {
			cellW := colW
			if i == end-1 {
				cellW = termW - colW*(n-1) // last column absorbs the remainder
			}
			lay.cells[i] = cellPlan{W: cellW, H: rowH}
			row = append(row, i)
		}
		lay.rows = append(lay.rows, row)
	}
	return lay
}

// packRows draws the widgets at the sizes fixed by lay.
func packRows(widgets []widget.Widget, th theme.Theme, lay layout) string {
	rows := make([]string, 0, len(lay.rows))
	for _, idxs := range lay.rows {
		cols := make([]string, 0, len(idxs))
		for _, i := range idxs {
			c := lay.cells[i]
			cols = append(cols, widgets[i].Render(c.W, c.H, th, false))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cols...))
	}
	return strings.Join(rows, "\n")
}

func typeHeight(t config.WidgetType) int {
	switch t {
	case config.TypeStat:
		return 6
	case config.TypeGauge:
		return 7
	case config.TypeBarGauge:
		return 8
	case config.TypeTimeSeries:
		return 10
	}
	return 6
}

// watchLoop clears the screen and re-renders on the refresh interval until the
// context is cancelled or an interrupt arrives.
func watchLoop(ctx context.Context, refresh time.Duration, render func(context.Context) string) error {
	if refresh <= 0 {
		refresh = 30 * time.Second
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	draw := func() {
		fmt.Print("\033[H\033[2J") // cursor home + clear screen
		fmt.Println(render(ctx))
	}
	draw()

	t := time.NewTicker(refresh)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			draw()
		}
	}
}

func termWidth() int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 40 {
		return w
	}
	return 100
}

func coalesce(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
