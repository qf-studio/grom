// Package app wires the canonical config model to renderable widgets and (in
// later phases) the Bubble Tea application shell.
package app

import (
	"fmt"

	"github.com/qf-studio/grom/internal/config"
	"github.com/qf-studio/grom/pkg/tui/widget"
)

// BuildWidget constructs a renderable widget from a spec.
func BuildWidget(spec config.WidgetSpec) (widget.Widget, error) {
	thresholds := convertThresholds(spec.Thresholds)
	switch spec.Type {
	case config.TypeStat:
		w := widget.NewStat(spec.Title, spec.Unit)
		w.Decimals = spec.Decimals
		w.Thresholds = thresholds
		return w, nil
	case config.TypeGauge:
		var min, max float64
		if spec.Min != nil {
			min = *spec.Min
		}
		if spec.Max != nil {
			max = *spec.Max
		}
		w := widget.NewGauge(spec.Title, spec.Unit, min, max)
		w.Decimals = spec.Decimals
		w.Thresholds = thresholds
		return w, nil
	case config.TypeBarGauge:
		w := widget.NewBarGauge(spec.Title, spec.Unit)
		w.Decimals = spec.Decimals
		w.Max = spec.Max
		w.Thresholds = thresholds
		return w, nil
	case config.TypeTimeSeries:
		w := widget.NewTimeSeries(spec.Title, spec.Unit)
		w.Decimals = spec.Decimals
		w.Stacked = spec.Stacked
		return w, nil
	case config.TypePlaceholder:
		// spec.Title already carries "unsupported: <type>".
		return widget.NewPlaceholder(spec.Title, spec.Title), nil
	default:
		return nil, fmt.Errorf("unknown widget type %q", spec.Type)
	}
}

func convertThresholds(ts []config.Threshold) []widget.Threshold {
	if len(ts) == 0 {
		return nil
	}
	out := make([]widget.Threshold, len(ts))
	for i, t := range ts {
		out[i] = widget.Threshold{Value: t.Value, Color: t.Color}
	}
	return out
}
