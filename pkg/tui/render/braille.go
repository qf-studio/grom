package render

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Braille dot bit layout within one cell (2 columns × 4 rows), per Unicode
// U+2800 + mask:
//
//	(0,0)=0x01  (1,0)=0x08
//	(0,1)=0x02  (1,1)=0x10
//	(0,2)=0x04  (1,2)=0x20
//	(0,3)=0x40  (1,3)=0x80
var brailleBits = [4][2]int{
	{0x01, 0x08},
	{0x02, 0x10},
	{0x04, 0x20},
	{0x40, 0x80},
}

// BrailleArea renders values as a filled braille area chart of exactly
// width×rows cells (top row first), colored with a vertical gradient:
// row 0 (top) gets gradient[last], bottom row gradient[0] — btop style.
//
// Resolution: width*2 data columns × rows*4 dot rows. Values are min/max
// scaled; each dot column fills from the bottom up to its value height.
func BrailleArea(values []float64, width, rows int, gradient []lipgloss.Style) []string {
	if width <= 0 || rows <= 0 {
		return nil
	}
	dotW := width * 2
	dotH := rows * 4

	heights := scaleToDots(values, dotW, dotH)

	// Build cell masks.
	masks := make([][]int, rows)
	for r := range masks {
		masks[r] = make([]int, width)
	}
	for dc := 0; dc < dotW; dc++ {
		h := heights[dc]
		for dr := 0; dr < h; dr++ {
			// dr counts from the bottom; convert to top-first dot row.
			y := dotH - 1 - dr
			masks[y/4][dc/2] |= brailleBits[y%4][dc%2]
		}
	}

	out := make([]string, rows)
	for r := 0; r < rows; r++ {
		var b strings.Builder
		st := rowStyle(gradient, r, rows)
		var run strings.Builder
		for c := 0; c < width; c++ {
			if masks[r][c] == 0 {
				run.WriteRune(' ')
			} else {
				run.WriteRune(rune(0x2800 | masks[r][c]))
			}
		}
		b.WriteString(st.Render(run.String()))
		out[r] = b.String()
	}
	return out
}

// BrailleLine renders values as a braille line chart (curve only, not
// filled), one dot per column at the value height, with line interpolation
// between adjacent points. Colored per row via gradient like BrailleArea.
func BrailleLine(values []float64, width, rows int, gradient []lipgloss.Style) []string {
	if width <= 0 || rows <= 0 {
		return nil
	}
	dotW := width * 2
	dotH := rows * 4

	heights := scaleToDots(values, dotW, dotH)

	masks := make([][]int, rows)
	for r := range masks {
		masks[r] = make([]int, width)
	}
	set := func(dc, hFromBottom int) {
		if hFromBottom < 1 {
			hFromBottom = 1
		}
		if hFromBottom > dotH {
			hFromBottom = dotH
		}
		y := dotH - hFromBottom
		masks[y/4][dc/2] |= brailleBits[y%4][dc%2]
	}
	prev := -1
	for dc := 0; dc < dotW; dc++ {
		h := heights[dc]
		if h == 0 {
			prev = -1
			continue
		}
		set(dc, h)
		// Vertical interpolation toward the previous column for continuity.
		if prev > 0 {
			lo, hi := prev, h
			if lo > hi {
				lo, hi = hi, lo
			}
			for v := lo + 1; v < hi; v++ {
				set(dc, v)
			}
		}
		prev = h
	}

	out := make([]string, rows)
	for r := 0; r < rows; r++ {
		st := rowStyle(gradient, r, rows)
		var run strings.Builder
		for c := 0; c < width; c++ {
			if masks[r][c] == 0 {
				run.WriteRune(' ')
			} else {
				run.WriteRune(rune(0x2800 | masks[r][c]))
			}
		}
		out[r] = st.Render(run.String())
	}
	return out
}

// BrailleMulti renders multiple series into one braille chart of exactly
// width×rows cells with a SHARED y-scale. Series 0 draws as a filled area
// with a vertical gradient of its color; the rest draw as lines in flat
// series colors (later series draw over earlier ones).
func BrailleMulti(series [][]float64, width, rows int, colors []string) []string {
	if width <= 0 || rows <= 0 || len(series) == 0 {
		return nil
	}
	dotW := width * 2
	dotH := rows * 4

	// Shared min/max across all series.
	first := true
	var minV, maxV float64
	for _, vals := range series {
		for _, v := range vals {
			if first {
				minV, maxV = v, v
				first = false
				continue
			}
			if v < minV {
				minV = v
			}
			if v > maxV {
				maxV = v
			}
		}
	}
	if first {
		return blankRows(width, rows)
	}

	masks := make([][]int, rows)
	owner := make([][]int, rows) // series index +1; 0 = unset
	for r := range masks {
		masks[r] = make([]int, width)
		owner[r] = make([]int, width)
	}
	set := func(dc, hFromBottom, si int) {
		if hFromBottom < 1 || hFromBottom > dotH {
			return
		}
		y := dotH - hFromBottom
		masks[y/4][dc/2] |= brailleBits[y%4][dc%2]
		owner[y/4][dc/2] = si + 1
	}

	for si, vals := range series {
		heights := scaleWithRange(vals, dotW, dotH, minV, maxV)
		if si == 0 {
			// Filled area.
			for dc := 0; dc < dotW; dc++ {
				for h := 1; h <= heights[dc]; h++ {
					set(dc, h, si)
				}
			}
			continue
		}
		// Line with vertical interpolation.
		prev := -1
		for dc := 0; dc < dotW; dc++ {
			h := heights[dc]
			if h == 0 {
				prev = -1
				continue
			}
			set(dc, h, si)
			if prev > 0 {
				lo, hi := prev, h
				if lo > hi {
					lo, hi = hi, lo
				}
				for v := lo + 1; v < hi; v++ {
					set(dc, v, si)
				}
			}
			prev = h
		}
	}

	// Styles: series 0 = vertical gradient; others flat.
	areaGradient := GradientStyles([]string{colors[0]}, rows)
	flat := make([]lipgloss.Style, len(series))
	for i := range series {
		c := colors[i%len(colors)]
		flat[i] = lipgloss.NewStyle().Foreground(lipgloss.Color(c))
	}

	out := make([]string, rows)
	for r := 0; r < rows; r++ {
		var b strings.Builder
		gradSt := rowStyle(areaGradient, r, rows)
		// Group consecutive same-owner cells into runs to limit escapes.
		c := 0
		for c < width {
			own := owner[r][c]
			start := c
			for c < width && owner[r][c] == own {
				c++
			}
			var run strings.Builder
			for i := start; i < c; i++ {
				if masks[r][i] == 0 {
					run.WriteRune(' ')
				} else {
					run.WriteRune(rune(0x2800 | masks[r][i]))
				}
			}
			switch own {
			case 0:
				b.WriteString(run.String())
			case 1:
				b.WriteString(gradSt.Render(run.String()))
			default:
				b.WriteString(flat[own-1].Render(run.String()))
			}
		}
		out[r] = b.String()
	}
	return out
}

func blankRows(width, rows int) []string {
	out := make([]string, rows)
	blank := strings.Repeat(" ", width)
	for i := range out {
		out[i] = blank
	}
	return out
}

// scaleWithRange resamples values onto dotW columns with an explicit y-range.
func scaleWithRange(values []float64, dotW, dotH int, minV, maxV float64) []int {
	heights := make([]int, dotW)
	if len(values) == 0 {
		return heights
	}
	span := maxV - minV
	n := len(values)
	cols := dotW
	if n < cols {
		cols = n
	}
	offset := dotW - cols
	for i := 0; i < cols; i++ {
		var idx int
		if cols == 1 {
			idx = n - 1
		} else {
			idx = int(math.Round(float64(i) / float64(cols-1) * float64(n-1)))
		}
		v := values[idx]
		var h int
		if span == 0 {
			if v > 0 {
				h = dotH / 2
			} else {
				h = 1
			}
		} else {
			h = int(math.Round((v-minV)/span*float64(dotH-1))) + 1
		}
		if h < 1 {
			h = 1
		}
		if h > dotH {
			h = dotH
		}
		heights[offset+i] = h
	}
	return heights
}

// scaleToDots resamples values onto dotW columns and scales each to a dot
// height in 1..dotH (0 = no data). Right-aligned: the most recent value maps
// to the rightmost column.
func scaleToDots(values []float64, dotW, dotH int) []int {
	heights := make([]int, dotW)
	if len(values) == 0 {
		return heights
	}

	minV, maxV := values[0], values[0]
	for _, v := range values[1:] {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	span := maxV - minV

	// Resample: map each dot column to a value index (nearest).
	n := len(values)
	cols := dotW
	if n < cols {
		cols = n
	}
	offset := dotW - cols
	for i := 0; i < cols; i++ {
		var idx int
		if cols == 1 {
			idx = n - 1
		} else {
			idx = int(math.Round(float64(i) / float64(cols-1) * float64(n-1)))
		}
		v := values[idx]
		var h int
		if span == 0 {
			if v > 0 {
				h = dotH / 2
			} else {
				h = 1
			}
		} else {
			h = int(math.Round((v-minV)/span*float64(dotH-1))) + 1
		}
		if h < 1 {
			h = 1
		}
		if h > dotH {
			h = dotH
		}
		heights[offset+i] = h
	}
	return heights
}

func rowStyle(gradient []lipgloss.Style, row, rows int) lipgloss.Style {
	if len(gradient) == 0 {
		return lipgloss.NewStyle()
	}
	// Top row = brightest (last gradient entry).
	idx := (rows - 1 - row) * len(gradient) / rows
	if idx < 0 {
		idx = 0
	}
	if idx >= len(gradient) {
		idx = len(gradient) - 1
	}
	return gradient[idx]
}
