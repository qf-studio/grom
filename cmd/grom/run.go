package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/qf-studio/grom/internal/app"
	"github.com/qf-studio/grom/internal/config"
	"github.com/qf-studio/grom/internal/datasource/prom"
	"github.com/qf-studio/grom/internal/grafana"
	"github.com/qf-studio/grom/pkg/tui/theme"
	"github.com/qf-studio/grom/pkg/tui/widget"
)

func runCmd() *cobra.Command {
	var (
		cfgPath     string
		grafanaPath string
		promAddr    string
		themeName   string
		ascii       bool
		once        bool
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Render a dashboard config against a live Prometheus",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dash, err := loadDashboard(cmd, cfgPath, grafanaPath)
			if err != nil {
				return err
			}
			ds, err := prom.New(promAddr)
			if err != nil {
				return err
			}
			th := theme.ByName(firstNonEmpty(themeName, dash.Theme))

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

			if once {
				w, h := termSize()
				rects := app.GridLayout(dash.Widgets, w, h)
				app.FetchAll(cmd.Context(), ds, dash, widgets, rects, fetchTimeout(dash))
				fmt.Println(app.StaticFrame(dash, widgets, th, w, h))
				return nil
			}
			return app.RunTUI(cmd.Context(), dash, widgets, ds, th)
		},
	}

	cmd.Flags().StringVarP(&cfgPath, "config", "c", "", "dashboard YAML config")
	cmd.Flags().StringVar(&grafanaPath, "grafana-json", "", "import a Grafana dashboard JSON instead of a YAML config")
	cmd.Flags().StringVar(&promAddr, "prom", "http://localhost:9090", "Prometheus base URL")
	cmd.Flags().StringVar(&themeName, "theme", "", "override theme (default: config theme)")
	cmd.Flags().BoolVar(&ascii, "ascii", false, "block-character charts instead of braille")
	cmd.Flags().BoolVar(&once, "once", false, "render a single static frame and exit (no TUI)")
	return cmd
}

// loadDashboard resolves the dashboard from exactly one of --config or
// --grafana-json. Import warnings are printed to stderr so they don't corrupt a
// piped --once frame on stdout.
func loadDashboard(cmd *cobra.Command, cfgPath, grafanaPath string) (*config.Dashboard, error) {
	switch {
	case cfgPath != "" && grafanaPath != "":
		return nil, fmt.Errorf("use --config or --grafana-json, not both")
	case grafanaPath != "":
		dash, warnings, err := grafana.Import(grafanaPath)
		if err != nil {
			return nil, err
		}
		for _, w := range warnings {
			cmd.PrintErrln("warning:", w)
		}
		return dash, nil
	case cfgPath != "":
		return config.Load(cfgPath)
	default:
		return nil, fmt.Errorf("one of --config or --grafana-json is required")
	}
}

// fetchTimeout bounds a single one-shot fetch: 10s, or the refresh interval if
// it is shorter.
func fetchTimeout(dash *config.Dashboard) time.Duration {
	t := 10 * time.Second
	if r := dash.Refresh.Duration(); r > 0 && r < t {
		t = r
	}
	return t
}

// termSize reports the terminal size, falling back to a sane default when
// stdout is not a TTY (e.g. piped output for --once).
func termSize() (int, int) {
	if w, h, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 40 {
		return w, h
	}
	return 100, 40
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
