package app

import (
	"testing"

	"github.com/qf-studio/grot/internal/config"
)

func TestUseRange(t *testing.T) {
	q := config.Query{}
	qi := config.Query{Instant: true}
	tests := []struct {
		name string
		spec config.WidgetSpec
		q    config.Query
		want bool
	}{
		{"timeseries", config.WidgetSpec{Type: config.TypeTimeSeries}, q, true},
		{"timeseries instant opt-out", config.WidgetSpec{Type: config.TypeTimeSeries}, qi, false},
		{"plain stat", config.WidgetSpec{Type: config.TypeStat}, q, false},
		{"sparkline stat", config.WidgetSpec{Type: config.TypeStat, Sparkline: true}, q, true},
		{"sparkline stat instant opt-out", config.WidgetSpec{Type: config.TypeStat, Sparkline: true}, qi, false},
		{"gauge ignores sparkline", config.WidgetSpec{Type: config.TypeGauge, Sparkline: true}, q, false},
		{"bargauge", config.WidgetSpec{Type: config.TypeBarGauge}, q, false},
	}
	for _, tt := range tests {
		if got := useRange(tt.spec, tt.q); got != tt.want {
			t.Errorf("%s: useRange = %v, want %v", tt.name, got, tt.want)
		}
	}
}
