package config

import (
	"testing"
	"time"
)

func TestParseValid(t *testing.T) {
	src := `
title: pilot
theme: tokyo-night
refresh: 15s
range: 1h
widgets:
  - type: stat
    title: cost
    unit: currencyUSD
    decimals: 2
    queries:
      - expr: sum(pilot_execution_cost_usd_total)
  - type: gauge
    title: success rate
    unit: percentunit
    min: 0
    max: 1
    thresholds:
      - color: red
      - value: 0.8
        color: yellow
    queries:
      - expr: pilot_success_rate
        legend: "{{job}}"
`
	d, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Title != "pilot" || d.Theme != "tokyo-night" {
		t.Fatalf("meta: got %q/%q", d.Title, d.Theme)
	}
	if d.Refresh.Duration() != 15*time.Second {
		t.Errorf("refresh: got %v", d.Refresh.Duration())
	}
	if d.Range.Duration() != time.Hour {
		t.Errorf("range: got %v", d.Range.Duration())
	}
	if len(d.Widgets) != 2 {
		t.Fatalf("widgets: got %d", len(d.Widgets))
	}

	cost := d.Widgets[0]
	if cost.Type != TypeStat || cost.Decimals == nil || *cost.Decimals != 2 {
		t.Errorf("stat: %+v", cost)
	}

	g := d.Widgets[1]
	if g.Type != TypeGauge || g.Min == nil || *g.Min != 0 || g.Max == nil || *g.Max != 1 {
		t.Errorf("gauge min/max: %+v", g)
	}
	if len(g.Thresholds) != 2 || g.Thresholds[0].Value != nil || g.Thresholds[1].Value == nil {
		t.Errorf("thresholds: %+v", g.Thresholds)
	}
	if g.Queries[0].Legend != "{{job}}" {
		t.Errorf("legend: %q", g.Queries[0].Legend)
	}
}

func TestParseDefaults(t *testing.T) {
	d, err := Parse([]byte(`
widgets:
  - type: stat
    title: x
    queries:
      - expr: up
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Theme != "pilot" {
		t.Errorf("default theme: %q", d.Theme)
	}
	if d.Refresh.Duration() != 30*time.Second {
		t.Errorf("default refresh: %v", d.Refresh.Duration())
	}
	if d.Range.Duration() != 15*time.Minute {
		t.Errorf("default range: %v", d.Range.Duration())
	}
}

func TestParseErrors(t *testing.T) {
	cases := map[string]string{
		"no widgets":    `title: x`,
		"unknown type":  "widgets:\n  - type: piechart\n    queries:\n      - expr: up",
		"no queries":    "widgets:\n  - type: stat\n    title: x",
		"empty expr":    "widgets:\n  - type: stat\n    queries:\n      - expr: \"  \"",
		"unknown field": "widgets:\n  - type: stat\n    queries:\n      - expr: up\n    color: red",
		"bad duration":  "refresh: 5parsecs\nwidgets:\n  - type: stat\n    queries:\n      - expr: up",
	}
	for name, src := range cases {
		if _, err := Parse([]byte(src)); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}
