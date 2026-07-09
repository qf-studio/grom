// grom — a btop-style terminal dashboard for Prometheus & Grafana.
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
		Use:   "grom",
		Short: "grom — btop-style terminal dashboards for Prometheus & Grafana",
		Long: `grom renders Prometheus metrics as polished terminal dashboards.
Import your existing Grafana dashboard JSON or write a simple YAML config.`,
		SilenceUsage: true,
	}

	root.AddCommand(demoCmd())
	root.AddCommand(runCmd())
	root.AddCommand(importCmd())
	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("grom %s (built %s)\n", version, buildTime)
		},
	})

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
