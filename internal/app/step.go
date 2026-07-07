package app

import "time"

// stepLadder is the set of "clean" query_range steps StepFor rounds up to, so
// axis ticks land on human-friendly intervals rather than arbitrary seconds.
var stepLadder = []time.Duration{
	10 * time.Second, 15 * time.Second, 30 * time.Second,
	1 * time.Minute, 2 * time.Minute, 5 * time.Minute,
	10 * time.Minute, 15 * time.Minute, 30 * time.Minute,
	1 * time.Hour, 2 * time.Hour, 3 * time.Hour,
	6 * time.Hour, 12 * time.Hour, 24 * time.Hour,
}

// StepFor picks a query_range step for a chart chartWidth cells wide. Braille
// packs two horizontal dots per cell, so the target sample count is chartWidth×2
// — roughly one sample per plotted dot. The raw step (range ÷ dots) is rounded
// UP to the next clean ladder value; rounding up keeps the returned sample count
// at or below the dot budget (never more points than we can draw) and floors the
// step at 10s so we don't hammer Prometheus on short ranges.
func StepFor(rng time.Duration, chartWidth int) time.Duration {
	if chartWidth < 1 {
		chartWidth = 1
	}
	raw := rng / time.Duration(chartWidth*2)
	return roundStep(raw)
}

// roundStep returns the smallest ladder value >= d. Anything at or below the
// floor (10s) snaps to the floor; anything above the ladder rounds up to whole
// hours.
func roundStep(d time.Duration) time.Duration {
	for _, s := range stepLadder {
		if d <= s {
			return s
		}
	}
	if d%time.Hour == 0 {
		return d
	}
	return d.Truncate(time.Hour) + time.Hour
}
