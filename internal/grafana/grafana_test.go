package grafana

import (
	"testing"
	"time"

	"github.com/qf-studio/grom/internal/config"
)

func TestImportPilotDashboard(t *testing.T) {
	dash, warnings, err := Import("testdata/pilot-dashboard.json")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for the Pilot dashboard, got %v", warnings)
	}
	if dash.Title != "Pilot Pipeline" {
		t.Errorf("title = %q, want %q", dash.Title, "Pilot Pipeline")
	}
	if got := dash.Range.Duration(); got != 6*time.Hour {
		t.Errorf("range = %s, want 6h (from now-6h)", got)
	}
	if got := dash.Refresh.Duration(); got != 30*time.Second {
		t.Errorf("refresh = %s, want 30s", got)
	}

	var stat, ts, stacked int
	for _, w := range dash.Widgets {
		switch w.Type {
		case config.TypeStat:
			stat++
			// Every Pilot stat panel sets graphMode "none" — no sparklines.
			if w.Sparkline {
				t.Errorf("stat %q: graphMode none should map to sparkline=false", w.Title)
			}
		case config.TypeTimeSeries:
			ts++
		}
		if w.Stacked {
			stacked++
		}
		if w.Grid.W == 0 {
			t.Errorf("widget %q lost its gridPos width", w.Title)
		}
	}
	if stat != 4 || ts != 12 {
		t.Errorf("panel counts: stat=%d ts=%d, want 4 and 12", stat, ts)
	}
	if stacked != 5 {
		t.Errorf("stacked timeseries = %d, want 5", stacked)
	}
}

func TestImportPilotStatDetails(t *testing.T) {
	dash, _, err := Import("testdata/pilot-dashboard.json")
	if err != nil {
		t.Fatal(err)
	}
	// First panel: "Success Rate (1h)" — percent, 1 decimal, 3 thresholds.
	w := dash.Widgets[0]
	if w.Title != "Success Rate (1h)" || w.Type != config.TypeStat {
		t.Fatalf("first widget = %q/%s", w.Title, w.Type)
	}
	if w.Unit != "percent" {
		t.Errorf("unit = %q, want percent", w.Unit)
	}
	if w.Decimals == nil || *w.Decimals != 1 {
		t.Errorf("decimals = %v, want 1", w.Decimals)
	}
	if len(w.Thresholds) != 3 {
		t.Fatalf("thresholds = %d, want 3", len(w.Thresholds))
	}
	if w.Thresholds[0].Value != nil {
		t.Errorf("base threshold should have nil value, got %v", *w.Thresholds[0].Value)
	}
	if w.Thresholds[2].Color != "green" || w.Thresholds[2].Value == nil || *w.Thresholds[2].Value != 80 {
		t.Errorf("top threshold = %+v, want green@80", w.Thresholds[2])
	}
	if len(w.Queries) == 0 || w.Queries[0].Legend != "Success Rate" {
		t.Errorf("query legend = %+v, want 'Success Rate'", w.Queries)
	}
}

func TestImportSynthetic(t *testing.T) {
	dash, warnings, err := Import("testdata/synthetic-dashboard.json")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	// gauge, bargauge, placeholder(heatmap), timeseries(promoted from row).
	if len(dash.Widgets) != 4 {
		t.Fatalf("widget count = %d, want 4; %+v", len(dash.Widgets), dash.Widgets)
	}
	if len(warnings) != 1 {
		t.Errorf("warnings = %v, want 1 (heatmap unsupported)", warnings)
	}

	gauge := dash.Widgets[0]
	if gauge.Type != config.TypeGauge || gauge.Min == nil || *gauge.Min != 0 || gauge.Max == nil || *gauge.Max != 100 {
		t.Errorf("gauge min/max = %v/%v, want 0/100", gauge.Min, gauge.Max)
	}
	bar := dash.Widgets[1]
	if bar.Type != config.TypeBarGauge || bar.Max == nil || *bar.Max != 10 {
		t.Errorf("bargauge max = %v, want 10", bar.Max)
	}
	ph := dash.Widgets[2]
	if ph.Type != config.TypePlaceholder || ph.Title != "unsupported: heatmap" {
		t.Errorf("placeholder = %q/%s", ph.Title, ph.Type)
	}
	if ph.Grid.X != 16 || ph.Grid.W != 8 {
		t.Errorf("placeholder lost grid slot: %+v", ph.Grid)
	}
	nested := dash.Widgets[3]
	if nested.Type != config.TypeTimeSeries || nested.Title != "Throughput" || !nested.Stacked {
		t.Errorf("promoted row child = %q/%s stacked=%v, want Throughput/timeseries/stacked", nested.Title, nested.Type, nested.Stacked)
	}
}

func TestParseNowRange(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
		ok   bool
	}{
		{"now-6h", 6 * time.Hour, true},
		{"now-15m", 15 * time.Minute, true},
		{"now-1h30m", 90 * time.Minute, true},
		{"now", 0, false},
		{"2024-01-01T00:00:00Z", 0, false},
		{"", 0, false},
		{"now-0h", 0, false},
	}
	for _, tt := range tests {
		got, ok := parseNowRange(tt.in)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("parseNowRange(%q) = %s,%v; want %s,%v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestConvertWrappedForm(t *testing.T) {
	// API exports wrap the dashboard under a "dashboard" key.
	wrapped := []byte(`{"dashboard":{"title":"Wrapped","refresh":"5s","time":{"from":"now-30m"},
		"panels":[{"type":"stat","title":"S","gridPos":{"x":0,"y":0,"w":24,"h":4},
		"targets":[{"expr":"up"}]}]}}`)
	dash, _, err := Convert(wrapped)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if dash.Title != "Wrapped" {
		t.Errorf("title = %q, want Wrapped", dash.Title)
	}
	if got := dash.Range.Duration(); got != 30*time.Minute {
		t.Errorf("range = %s, want 30m", got)
	}
	if len(dash.Widgets) != 1 || dash.Widgets[0].Type != config.TypeStat {
		t.Errorf("widgets = %+v, want one stat", dash.Widgets)
	}
	// graphMode absent → Grafana's default "area" → sparkline on.
	if !dash.Widgets[0].Sparkline {
		t.Error("stat without graphMode should default to sparkline=true")
	}
}
