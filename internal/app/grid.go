package app

import (
	"sort"

	"github.com/qf-studio/grot/internal/config"
)

// Rect is a widget's placement in terminal cells, top-left origin.
type Rect struct{ X, Y, W, H int }

// minColW is the narrowest a widget cell is allowed to get before the packer
// drops to fewer columns per row.
const minColW = 34

// typeHeight is the row height (in terminal cells) a widget of the given type
// wants. Charts are tallest; single-value panels are short.
func typeHeight(t config.WidgetType) int {
	switch t {
	case config.TypeStat:
		return 6
	case config.TypeGauge:
		return 7
	case config.TypeBarGauge:
		return 8
	case config.TypeTimeSeries:
		return 10
	}
	return 6
}

// maxPerRow caps how many widgets share a row at each width breakpoint:
// <80 → single column, 80–139 → up to two, ≥140 → up to four.
func maxPerRow(termW int) int {
	switch {
	case termW < 80:
		return 1
	case termW < 140:
		return 2
	default:
		return 4
	}
}

// GridLayout arranges widgets into absolute terminal-cell rectangles for a
// termW×termH viewport. Dashboards imported from Grafana carry 24-column
// gridPos, which is scaled to the terminal; native configs omit placement and
// are auto-flowed into rows. termH is advisory — vertical overflow is handled
// by the caller's scroll.
func GridLayout(specs []config.WidgetSpec, termW, termH int) []Rect {
	_ = termH
	if len(specs) == 0 {
		return nil
	}
	if hasGridPos(specs) {
		return gridPosLayout(specs, termW)
	}
	return autoFlow(specs, termW)
}

// hasGridPos reports whether any widget declares an explicit 24-column slot.
func hasGridPos(specs []config.WidgetSpec) bool {
	for _, s := range specs {
		if s.Grid.W > 0 {
			return true
		}
	}
	return false
}

// gridUnitRows converts Grafana grid-height units (~30px each) to terminal
// rows. Grafana stat panels are h=4, charts h=8; ×3/2 gives 6 and 12 — close to
// the auto-flow heights — with a floor so nothing renders below its chrome.
func gridUnitRows(h int) int {
	r := h * 3 / 2
	if r < 4 {
		return 4
	}
	return r
}

// gridPosLayout maps each widget's 24-column gridPos onto the terminal. Widgets
// sharing a Grafana row (same Y) map to the same terminal band and the same
// height (the row's tallest), so the row-based compositor reproduces the
// dashboard. Horizontal position/width scale from 24 columns to termW.
func gridPosLayout(specs []config.WidgetSpec, termW int) []Rect {
	rects := make([]Rect, len(specs))

	// Distinct Grafana rows, top to bottom.
	rowOf := map[int][]int{}
	var ys []int
	for i, s := range specs {
		if _, seen := rowOf[s.Grid.Y]; !seen {
			ys = append(ys, s.Grid.Y)
		}
		rowOf[s.Grid.Y] = append(rowOf[s.Grid.Y], i)
	}
	sort.Ints(ys)

	yCell := 0
	for _, gy := range ys {
		idxs := rowOf[gy]
		rowH := 0
		for _, i := range idxs {
			if h := gridUnitRows(specs[i].Grid.H); h > rowH {
				rowH = h
			}
		}
		for _, i := range idxs {
			g := specs[i].Grid
			x0 := g.X * termW / 24
			x1 := (g.X + g.W) * termW / 24
			if x1 > termW {
				x1 = termW
			}
			rects[i] = Rect{X: x0, Y: yCell, W: x1 - x0, H: rowH}
		}
		yCell += rowH
	}
	return rects
}

// autoFlow packs widgets left-to-right into rows, even-splitting each row's
// width. Used for native configs, which omit explicit placement.
func autoFlow(specs []config.WidgetSpec, termW int) []Rect {
	rects := make([]Rect, len(specs))
	per := maxPerRow(termW)
	if fit := termW / minColW; fit < per {
		per = fit
	}
	if per < 1 {
		per = 1
	}

	y := 0
	for start := 0; start < len(specs); start += per {
		end := start + per
		if end > len(specs) {
			end = len(specs)
		}
		n := end - start
		colW := termW / n
		rowH := 0
		for i := start; i < end; i++ {
			if h := typeHeight(specs[i].Type); h > rowH {
				rowH = h
			}
		}
		x := 0
		for i := start; i < end; i++ {
			w := colW
			if i == end-1 {
				w = termW - colW*(n-1) // last column absorbs the remainder
			}
			rects[i] = Rect{X: x, Y: y, W: w, H: rowH}
			x += colW
		}
		y += rowH
	}
	return rects
}

// GridHeight is the total cell height the laid-out grid occupies.
func GridHeight(rects []Rect) int {
	max := 0
	for _, r := range rects {
		if b := r.Y + r.H; b > max {
			max = b
		}
	}
	return max
}

// focusMove returns the index of the widget nearest to cur in direction dir
// ('h' left, 'j' down, 'k' up, 'l' right), or cur when nothing lies that way.
// Horizontal moves only consider widgets whose rows overlap the current one
// (and vertical moves, columns), then pick the nearest by edge distance — so
// focus tracks the visual row/column rather than drifting diagonally.
func focusMove(rects []Rect, cur int, dir byte) int {
	if cur < 0 || cur >= len(rects) {
		return cur
	}
	c := rects[cur]
	best, bestDist, found := cur, 0, false
	for i, r := range rects {
		if i == cur {
			continue
		}
		var dist int
		switch dir {
		case 'h':
			if r.X >= c.X || !overlaps(c.Y, c.H, r.Y, r.H) {
				continue
			}
			dist = c.X - r.X
		case 'l':
			if r.X <= c.X || !overlaps(c.Y, c.H, r.Y, r.H) {
				continue
			}
			dist = r.X - c.X
		case 'k':
			if r.Y >= c.Y || !overlaps(c.X, c.W, r.X, r.W) {
				continue
			}
			dist = c.Y - r.Y
		case 'j':
			if r.Y <= c.Y || !overlaps(c.X, c.W, r.X, r.W) {
				continue
			}
			dist = r.Y - c.Y
		default:
			return cur
		}
		if !found || dist < bestDist {
			bestDist, best, found = dist, i, true
		}
	}
	return best
}

// overlaps reports whether the spans [a, a+al) and [b, b+bl) intersect.
func overlaps(a, al, b, bl int) bool {
	return a < b+bl && b < a+al
}
