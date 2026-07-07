package app

import (
	"testing"

	"github.com/qf-studio/grot/internal/config"
	"github.com/qf-studio/grot/pkg/tui/widget"
)

func fptr(v float64) *float64 { return &v }
func iptr(v int) *int         { return &v }

func TestBuildWidgetTypes(t *testing.T) {
	stat, err := BuildWidget(config.WidgetSpec{
		Type: config.TypeStat, Title: "s", Unit: "short", Decimals: iptr(1),
		Thresholds: []config.Threshold{{Color: "red"}, {Value: fptr(5), Color: "green"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s, ok := stat.(*widget.Stat)
	if !ok {
		t.Fatalf("stat type: %T", stat)
	}
	if len(s.Thresholds) != 2 || s.Thresholds[1].Value == nil || *s.Thresholds[1].Value != 5 {
		t.Errorf("stat thresholds: %+v", s.Thresholds)
	}
	if s.Decimals == nil || *s.Decimals != 1 {
		t.Errorf("stat decimals: %v", s.Decimals)
	}

	g, err := BuildWidget(config.WidgetSpec{Type: config.TypeGauge, Unit: "percentunit", Min: fptr(0), Max: fptr(1)})
	if err != nil {
		t.Fatal(err)
	}
	if gg := g.(*widget.Gauge); gg.Min != 0 || gg.Max != 1 {
		t.Errorf("gauge min/max: %v/%v", gg.Min, gg.Max)
	}

	b, err := BuildWidget(config.WidgetSpec{Type: config.TypeBarGauge, Max: fptr(100)})
	if err != nil {
		t.Fatal(err)
	}
	if bg := b.(*widget.BarGauge); bg.Max == nil || *bg.Max != 100 {
		t.Errorf("bargauge max: %v", bg.Max)
	}

	ts, err := BuildWidget(config.WidgetSpec{Type: config.TypeTimeSeries, Stacked: true})
	if err != nil {
		t.Fatal(err)
	}
	if !ts.(*widget.TimeSeries).Stacked {
		t.Errorf("timeseries stacked not set")
	}
}

func TestBuildWidgetUnknown(t *testing.T) {
	if _, err := BuildWidget(config.WidgetSpec{Type: "nope"}); err == nil {
		t.Error("expected error for unknown type")
	}
}
