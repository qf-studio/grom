package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/qf-studio/grot/pkg/tui/theme"
	"github.com/qf-studio/grot/pkg/tui/widget"
)

func demoCmd() *cobra.Command {
	var themeName string
	var width int

	cmd := &cobra.Command{
		Use:   "demo",
		Short: "Render a static widget gallery with fake data",
		Run: func(cmd *cobra.Command, args []string) {
			if width == 0 {
				if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 40 {
					width = w
				} else {
					width = 100
				}
			}
			th := theme.ByName(themeName)
			fmt.Println(renderDemo(th, width))
		},
	}
	cmd.Flags().StringVar(&themeName, "theme", "pilot", "theme: pilot | tokyo-night | catppuccin-mocha")
	cmd.Flags().IntVar(&width, "width", 0, "render width (default: terminal width)")
	return cmd
}

func renderDemo(th theme.Theme, width int) string {
	now := time.Now()
	instant := func(legend string, v float64) widget.QueryResult {
		return widget.QueryResult{
			Series:    []widget.Series{{Legend: legend, Points: []widget.Point{{T: now, V: v}}}},
			FetchedAt: now,
		}
	}
	rangeSeries := func(legend string, gen func(i int) float64, n int) widget.Series {
		pts := make([]widget.Point, n)
		for i := 0; i < n; i++ {
			pts[i] = widget.Point{T: now.Add(time.Duration(i-n) * time.Minute), V: gen(i)}
		}
		return widget.Series{Legend: legend, Points: pts}
	}

	fv := func(v float64) *float64 { return &v }

	// noisy plateau: stable base + jitter + occasional spikes — the shape
	// real service metrics actually have.
	noisy := func(base, jitter, spike float64) func(i int) float64 {
		return func(i int) float64 {
			v := base + jitter*math.Sin(float64(i)*1.7)*math.Cos(float64(i)*0.31)
			if i%17 == 0 {
				v += spike
			}
			if i%23 == 0 {
				v += spike * 0.6
			}
			if v < 0 {
				v = 0
			}
			return v
		}
	}

	// --- Row 1: four stats ---
	statW := width / 4
	lastW := width - statW*3 // absorb remainder
	statH := 6

	success := widget.NewStat("success rate", "percent")
	success.Thresholds = []widget.Threshold{
		{Color: "red"}, {Value: fv(70), Color: "yellow"}, {Value: fv(90), Color: "green"},
	}
	success.SetResult(instant("rate", 66.0))

	queue := widget.NewStat("queue depth", "short")
	queue.Thresholds = []widget.Threshold{
		{Color: "green"}, {Value: fv(5), Color: "yellow"}, {Value: fv(15), Color: "red"},
	}
	queueGen := noisy(2, 1.2, 4)
	queue.SetResult(widget.QueryResult{Series: []widget.Series{
		rangeSeries("queue", func(i int) float64 { return math.Round(queueGen(i)) }, 60),
	}, FetchedAt: now})

	prs := widget.NewStat("active prs", "short")
	prs.SetResult(instant("prs", 4))

	cost := widget.NewStat("cumulative cost", "currencyUSD")
	cost.SetResult(instant("cost", 154.23))

	row1 := lipgloss.JoinHorizontal(lipgloss.Top,
		success.Render(statW, statH, th, false),
		queue.Render(statW, statH, th, false),
		prs.Render(statW, statH, th, false),
		cost.Render(lastW, statH, th, true), // focused example
	)

	// --- Row 2: gauge + bargauge ---
	halfW := width / 2
	otherW := width - halfW
	row2H := 7

	gauge := widget.NewGauge("ci pass rate", "percent", 0, 100)
	gauge.Thresholds = []widget.Threshold{
		{Color: "red"}, {Value: fv(70), Color: "yellow"}, {Value: fv(90), Color: "green"},
	}
	gauge.SetResult(instant("ci", 87.5))

	bars := widget.NewBarGauge("tokens by model", "short")
	bars.SetResult(widget.QueryResult{Series: []widget.Series{
		{Legend: "opus/input", Points: []widget.Point{{T: now, V: 57_300}}},
		{Legend: "opus/output", Points: []widget.Point{{T: now, V: 31_000}}},
		{Legend: "haiku/input", Points: []widget.Point{{T: now, V: 12_400}}},
		{Legend: "haiku/output", Points: []widget.Point{{T: now, V: 4_100}}},
	}, FetchedAt: now})

	row2 := lipgloss.JoinHorizontal(lipgloss.Top,
		gauge.Render(halfW, row2H, th, false),
		bars.Render(otherW, row2H, th, false),
	)

	// --- Row 3: two timeseries (btop-short: airy, not ink mountains) ---
	row3H := 9

	p95 := widget.NewTimeSeries("execution duration p95", "s")
	p95.SetResult(widget.QueryResult{Series: []widget.Series{
		rangeSeries("p95", noisy(280, 40, 180), 160),
	}, FetchedAt: now})

	tokens := widget.NewTimeSeries("tokens / 5m", "short")
	tokens.Stacked = true
	tokens.SetResult(widget.QueryResult{Series: []widget.Series{
		rangeSeries("opus", noisy(1800, 500, 1400), 160),
		rangeSeries("haiku", noisy(700, 250, 500), 160),
	}, FetchedAt: now})

	row3 := lipgloss.JoinHorizontal(lipgloss.Top,
		p95.Render(halfW, row3H, th, false),
		tokens.Render(otherW, row3H, th, false),
	)

	// --- Error/no-data states ---
	row4H := 5
	errW := widget.NewStat("broken query", "short")
	errW.SetError(fmt.Errorf(`parse error: unexpected "}"`))
	empty := widget.NewTimeSeries("no data example", "short")
	empty.SetResult(widget.QueryResult{FetchedAt: now})

	row4 := lipgloss.JoinHorizontal(lipgloss.Top,
		errW.Render(halfW, row4H, th, false),
		empty.Render(otherW, row4H, th, false),
	)

	header := th.AccentStyle().Bold(true).Render(" grot") +
		th.DimStyle().Render(" · demo gallery · theme: "+th.Name)

	return header + "\n" + row1 + "\n" + row2 + "\n" + row3 + "\n" + row4
}
