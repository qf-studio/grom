package render

import "math"

// SparkBlocks maps normalized levels (0-8) to Unicode block elements.
// Index 0 is a space, reserved for padding.
var SparkBlocks = []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// NormalizeSparkline scales values to 1-8 levels for sparkline rendering,
// right-aligned within width (left-padded with 0 = blank). More values than
// width keeps the most recent.
func NormalizeSparkline(values []float64, width int) []int {
	result := make([]int, width)
	if len(values) == 0 || width <= 0 {
		return result
	}

	offset := width - len(values)
	if offset < 0 {
		values = values[len(values)-width:]
		offset = 0
	}

	minVal, maxVal := values[0], values[0]
	for _, v := range values[1:] {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	span := maxVal - minVal
	if span == 0 {
		level := 1
		if values[0] > 0 {
			level = 4
		}
		for i := range values {
			result[offset+i] = level
		}
		return result
	}

	for i, v := range values {
		normalized := (v - minVal) / span * 7
		level := int(math.Round(normalized)) + 1
		if v == 0 {
			level = 1
		}
		if level < 1 {
			level = 1
		}
		if level > 8 {
			level = 8
		}
		result[offset+i] = level
	}
	return result
}

// Sparkline renders levels (0-8) as block characters, exactly width cells.
func Sparkline(levels []int, width int) string {
	out := make([]rune, width)
	start := 0
	if len(levels) > width {
		start = len(levels) - width
	}
	pad := width - (len(levels) - start)
	for i := 0; i < pad; i++ {
		out[i] = SparkBlocks[0]
	}
	for i := start; i < len(levels); i++ {
		idx := levels[i]
		if idx < 0 {
			idx = 0
		}
		if idx >= len(SparkBlocks) {
			idx = len(SparkBlocks) - 1
		}
		out[pad+i-start] = SparkBlocks[idx]
	}
	return string(out)
}

// BlockChart renders values as a multi-row block area chart of exactly
// width×rows cells, one string per row (top first). Each column is scaled to
// rows*8 sub-levels; partial tops use ▁▂▃▄▅▆▇, filled cells use █.
func BlockChart(values []float64, width, rows int) []string {
	if rows < 1 {
		rows = 1
	}
	levels := normalizeTo(values, width, rows*8)
	out := make([]string, rows)
	for r := 0; r < rows; r++ {
		rowRunes := make([]rune, width)
		// Sub-level threshold below this row (rows are top-first).
		base := (rows - 1 - r) * 8
		for c := 0; c < width; c++ {
			fill := levels[c] - base
			if fill < 0 {
				fill = 0
			}
			if fill > 8 {
				fill = 8
			}
			rowRunes[c] = SparkBlocks[fill]
		}
		out[r] = string(rowRunes)
	}
	return out
}

// normalizeTo scales values to 0..maxLevel, right-aligned within width.
func normalizeTo(values []float64, width, maxLevel int) []int {
	result := make([]int, width)
	if len(values) == 0 || width <= 0 {
		return result
	}
	offset := width - len(values)
	if offset < 0 {
		values = values[len(values)-width:]
		offset = 0
	}
	minVal, maxVal := values[0], values[0]
	for _, v := range values[1:] {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	span := maxVal - minVal
	for i, v := range values {
		var level int
		if span == 0 {
			if v > 0 {
				level = maxLevel / 2
			} else {
				level = 1
			}
		} else {
			level = int(math.Round((v-minVal)/span*float64(maxLevel-1))) + 1
			if v == 0 && minVal == 0 {
				level = 1
			}
		}
		if level < 0 {
			level = 0
		}
		if level > maxLevel {
			level = maxLevel
		}
		result[offset+i] = level
	}
	return result
}
