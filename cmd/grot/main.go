// grot — a btop-style terminal dashboard for Prometheus & Grafana.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	root := &cobra.Command{
		Use:   "grot",
		Short: "grot — btop-style terminal dashboards for Prometheus & Grafana",
		Long: `grot renders Prometheus metrics as polished terminal dashboards.
Import your existing Grafana dashboard JSON or write a simple YAML config.`,
		SilenceUsage: true,
	}

	root.AddCommand(demoCmd())
	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("grot %s (built %s)\n", version, buildTime)
		},
	})

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
