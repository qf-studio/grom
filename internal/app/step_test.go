package app

import (
	"testing"
	"time"
)

func TestStepFor(t *testing.T) {
	tests := []struct {
		name  string
		rng   time.Duration
		width int
		want  time.Duration
	}{
		{
			// 15m over 80 cells = 160 dots → 5.6s raw → floored to 10s.
			name: "short range floors at 10s", rng: 15 * time.Minute, width: 80,
			want: 10 * time.Second,
		},
		{
			// 6h over 80 cells = 160 dots → 135s raw → next rung 5m.
			name: "mid range rounds up to clean rung", rng: 6 * time.Hour, width: 80,
			want: 5 * time.Minute,
		},
		{
			// 24h over 100 cells = 200 dots → 432s raw → next rung 10m.
			name: "day range", rng: 24 * time.Hour, width: 100,
			want: 10 * time.Minute,
		},
		{
			// 1h over 30 cells = 60 dots → 60s raw → exact rung 1m (<= match).
			name: "raw lands exactly on a rung", rng: time.Hour, width: 30,
			want: 1 * time.Minute,
		},
		{
			// Very narrow chart still floors at 10s, never zero.
			name: "narrow chart floors at 10s", rng: time.Minute, width: 4,
			want: 10 * time.Second,
		},
		{
			// Above the ladder: 60h over 1 cell = 2 dots → 30h raw, already a
			// whole hour, so returned as-is (no spurious +1h).
			name: "beyond ladder keeps clean whole hour", rng: 60 * time.Hour, width: 1,
			want: 30 * time.Hour,
		},
		{
			// Above the ladder, fractional: 51h over 1 cell = 2 dots → 25.5h raw
			// → rounds up to 26h.
			name: "beyond ladder rounds fractional up", rng: 51 * time.Hour, width: 1,
			want: 26 * time.Hour,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StepFor(tt.rng, tt.width); got != tt.want {
				t.Errorf("StepFor(%s, %d) = %s, want %s", tt.rng, tt.width, got, tt.want)
			}
		})
	}
}

// StepFor must never return a step so small it exceeds the dot budget, nor a
// non-positive step, across a sweep of realistic ranges and widths.
func TestStepForInvariants(t *testing.T) {
	ranges := []time.Duration{
		time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour,
		6 * time.Hour, 24 * time.Hour, 7 * 24 * time.Hour,
	}
	widths := []int{0, 1, 4, 24, 80, 200, 400}
	for _, rng := range ranges {
		for _, w := range widths {
			step := StepFor(rng, w)
			if step < 10*time.Second {
				t.Errorf("StepFor(%s, %d) = %s below 10s floor", rng, w, step)
			}
			effW := w
			if effW < 1 {
				effW = 1
			}
			dots := effW * 2
			if samples := int(rng / step); samples > dots {
				t.Errorf("StepFor(%s, %d) = %s yields %d samples > %d dot budget",
					rng, w, step, samples, dots)
			}
		}
	}
}
