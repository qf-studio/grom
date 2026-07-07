# grot

**btop-style terminal dashboards for Prometheus & Grafana.**

grot renders Prometheus metrics as polished terminal dashboards — braille
charts, gradient gauges, threshold-colored stats. Point it at your existing
Grafana dashboard JSON and get the same layout in your terminal.

> Status: pre-release, under active development.

## Quick start

```bash
# Widget gallery with fake data
grot demo --theme tokyo-night

# Coming soon:
# grot --prom http://localhost:9090 --config dashboard.yaml
# grot --prom http://localhost:9090 --grafana-json my-dashboard.json
```

## Why

- [grafterm](https://github.com/slok/grafterm) is abandoned (2019) and limited
  to termdash's fixed widgets.
- [btop](https://github.com/aristocratos/btop) proves terminal dashboards can
  be beautiful — but it only shows system metrics.
- Nothing renders *your* Grafana dashboards in the terminal. grot does.

## Features

- **Grafana import** — parse dashboard JSON (stat, gauge, bargauge,
  timeseries panels) into a terminal layout
- **Native YAML config** — simple dashboard definitions without Grafana
- **Themes** — pilot, tokyo-night, catppuccin-mocha
- **btop-like keys** — hjkl focus, zoom, live time-range switching
- Widget-local errors: a broken query never breaks the grid

## Development

```bash
make build   # build ./bin/grot
make demo    # render the widget gallery
make test    # go test -race
make lint    # golangci-lint
```

## License

MIT
