// Package config defines grot's canonical dashboard model and a YAML loader.
// Both native YAML configs and (later) Grafana JSON import produce a Dashboard.
package config

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// WidgetType enumerates the renderable widget kinds.
type WidgetType string

const (
	TypeStat       WidgetType = "stat"
	TypeGauge      WidgetType = "gauge"
	TypeBarGauge   WidgetType = "bargauge"
	TypeTimeSeries WidgetType = "timeseries"
)

// Dashboard is grot's canonical dashboard model.
type Dashboard struct {
	Title   string       `yaml:"title"`
	Theme   string       `yaml:"theme"`
	Refresh Duration     `yaml:"refresh"`
	Range   Duration     `yaml:"range"`
	Widgets []WidgetSpec `yaml:"widgets"`
}

// WidgetSpec describes one panel: its type, placement, queries, and formatting.
type WidgetSpec struct {
	Type       WidgetType  `yaml:"type"`
	Title      string      `yaml:"title"`
	Grid       GridPos     `yaml:"grid"`
	Queries    []Query     `yaml:"queries"`
	Unit       string      `yaml:"unit"`
	Decimals   *int        `yaml:"decimals"`
	Min        *float64    `yaml:"min"`
	Max        *float64    `yaml:"max"`
	Thresholds []Threshold `yaml:"thresholds"`
	Stacked    bool        `yaml:"stacked"`
	Reduce     string      `yaml:"reduce"`
}

// Query is a single PromQL expression with an optional {{label}} legend.
// Instant selects an instant query; otherwise a range query is used.
type Query struct {
	Expr    string `yaml:"expr"`
	Legend  string `yaml:"legend"`
	Instant bool   `yaml:"instant"`
}

// GridPos is a Grafana-compatible 24-column placement.
type GridPos struct {
	X int `yaml:"x"`
	Y int `yaml:"y"`
	W int `yaml:"w"`
	H int `yaml:"h"`
}

// Threshold colors a value at or above Value. A nil Value is the base color.
type Threshold struct {
	Value *float64 `yaml:"value"`
	Color string   `yaml:"color"`
}

// Duration is a time.Duration that unmarshals from a YAML string like "30s".
type Duration time.Duration

// UnmarshalYAML parses a Go duration string ("30s", "5m", "1h").
func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return fmt.Errorf("duration: %w", err)
	}
	if s == "" {
		return nil
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(v)
	return nil
}

// Duration returns the underlying time.Duration.
func (d Duration) Duration() time.Duration { return time.Duration(d) }

// Load reads and validates a dashboard config from a YAML file.
func Load(path string) (*Dashboard, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	return Parse(b)
}

// Parse decodes and validates a dashboard config from YAML bytes. Unknown
// fields are rejected so typos surface immediately.
func Parse(b []byte) (*Dashboard, error) {
	var d Dashboard
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&d); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := d.validate(); err != nil {
		return nil, err
	}
	d.applyDefaults()
	return &d, nil
}

func (d *Dashboard) validate() error {
	if len(d.Widgets) == 0 {
		return fmt.Errorf("config has no widgets")
	}
	for i, w := range d.Widgets {
		label := w.Title
		if label == "" {
			label = fmt.Sprintf("#%d", i)
		}
		switch w.Type {
		case TypeStat, TypeGauge, TypeBarGauge, TypeTimeSeries:
		case "":
			return fmt.Errorf("widget %s: missing type", label)
		default:
			return fmt.Errorf("widget %s: unknown type %q", label, w.Type)
		}
		if len(w.Queries) == 0 {
			return fmt.Errorf("widget %s: no queries", label)
		}
		for j, q := range w.Queries {
			if strings.TrimSpace(q.Expr) == "" {
				return fmt.Errorf("widget %s: query %d has empty expr", label, j)
			}
		}
	}
	return nil
}

func (d *Dashboard) applyDefaults() {
	if d.Theme == "" {
		d.Theme = "pilot"
	}
	if d.Refresh == 0 {
		d.Refresh = Duration(30 * time.Second)
	}
	if d.Range == 0 {
		d.Range = Duration(15 * time.Minute)
	}
}
