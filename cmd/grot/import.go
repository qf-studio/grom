package main

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/qf-studio/grot/internal/config"
	"github.com/qf-studio/grot/internal/grafana"
)

func importCmd() *cobra.Command {
	var check bool

	cmd := &cobra.Command{
		Use:   "import <grafana-dashboard.json>",
		Short: "Convert a Grafana dashboard JSON and report what maps",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dash, warnings, err := grafana.Import(args[0])
			if err != nil {
				return err
			}

			if check {
				for _, w := range warnings {
					cmd.Println("warning:", w)
				}
				if len(warnings) > 0 {
					return fmt.Errorf("%d warning(s)", len(warnings))
				}
				cmd.Println("ok: no warnings")
				return nil
			}

			cmd.Printf("%s — %d widgets · theme %s · range %s · refresh %s\n",
				dash.Title, len(dash.Widgets), dash.Theme,
				dash.Range.Duration(), dash.Refresh.Duration())
			counts := countTypes(dash.Widgets)
			for _, t := range sortedKeys(counts) {
				cmd.Printf("  %-11s %d\n", t, counts[t])
			}
			for _, w := range warnings {
				cmd.Println("warning:", w)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "only validate and print warnings (non-zero exit on any)")
	return cmd
}

func countTypes(widgets []config.WidgetSpec) map[string]int {
	counts := map[string]int{}
	for _, w := range widgets {
		counts[string(w.Type)]++
	}
	return counts
}

// sortedKeys returns the map keys in lexical order for deterministic output.
func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
