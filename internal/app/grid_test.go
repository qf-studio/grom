package app

import (
	"testing"

	"github.com/qf-studio/grot/internal/config"
)

func specs(types ...config.WidgetType) []config.WidgetSpec {
	out := make([]config.WidgetSpec, len(types))
	for i, t := range types {
		out[i] = config.WidgetSpec{Type: t}
	}
	return out
}

func TestGridLayoutBreakpoints(t *testing.T) {
	four := specs(config.TypeStat, config.TypeStat, config.TypeStat, config.TypeStat)

	tests := []struct {
		name       string
		termW      int
		wantPerRow int // widgets sharing the first row
	}{
		{"narrow stacks single column", 70, 1},
		{"medium packs two", 100, 2},
		{"wide packs four", 200, 4},
		{"width caps at ~34-col min", 80, 2}, // 80/34 = 2
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rects := GridLayout(four, tt.termW, 50)
			firstRowY := rects[0].Y
			got := 0
			for _, r := range rects {
				if r.Y == firstRowY {
					got++
				}
			}
			if got != tt.wantPerRow {
				t.Errorf("termW=%d: first row has %d widgets, want %d", tt.termW, got, tt.wantPerRow)
			}
		})
	}
}

func TestGridLayoutSpansFullWidth(t *testing.T) {
	// Every row must span exactly termW with no gaps or overhang.
	rects := GridLayout(specs(config.TypeStat, config.TypeStat, config.TypeStat), 141, 50)
	byRow := map[int][]Rect{}
	for _, r := range rects {
		byRow[r.Y] = append(byRow[r.Y], r)
	}
	for y, row := range byRow {
		right := 0
		x := 0
		for _, r := range row {
			if r.X != x {
				t.Errorf("row y=%d: gap — rect X=%d, expected %d", y, r.X, x)
			}
			x += r.W
			right = r.X + r.W
		}
		if right != 141 {
			t.Errorf("row y=%d spans %d cells, want 141", y, right)
		}
	}
}

func TestGridLayoutRowHeightIsTallest(t *testing.T) {
	// A stat (6) beside a timeseries (10) share the timeseries height.
	rects := GridLayout(specs(config.TypeStat, config.TypeTimeSeries), 200, 50)
	if rects[0].H != rects[1].H {
		t.Fatalf("cells in a row differ in height: %d vs %d", rects[0].H, rects[1].H)
	}
	if rects[0].H != typeHeight(config.TypeTimeSeries) {
		t.Errorf("row height = %d, want tallest %d", rects[0].H, typeHeight(config.TypeTimeSeries))
	}
}

func TestGridLayoutEmpty(t *testing.T) {
	if got := GridLayout(nil, 100, 50); len(got) != 0 {
		t.Errorf("empty specs → %d rects, want 0", len(got))
	}
}

func TestFocusMove(t *testing.T) {
	// 2×2 grid of equal cells:
	//   0 1
	//   2 3
	rects := []Rect{
		{X: 0, Y: 0, W: 10, H: 5},
		{X: 10, Y: 0, W: 10, H: 5},
		{X: 0, Y: 5, W: 10, H: 5},
		{X: 10, Y: 5, W: 10, H: 5},
	}
	tests := []struct {
		cur  int
		dir  byte
		want int
	}{
		{0, 'l', 1},
		{0, 'j', 2},
		{1, 'h', 0},
		{3, 'k', 1},
		{3, 'h', 2},
		{0, 'h', 0}, // nothing to the left → stays
		{0, 'k', 0}, // nothing above → stays
		{2, 'j', 2}, // nothing below → stays
	}
	for _, tt := range tests {
		if got := focusMove(rects, tt.cur, tt.dir); got != tt.want {
			t.Errorf("focusMove(cur=%d, dir=%c) = %d, want %d", tt.cur, tt.dir, got, tt.want)
		}
	}
}

func TestGridHeight(t *testing.T) {
	rects := GridLayout(specs(config.TypeStat, config.TypeStat, config.TypeTimeSeries), 70, 50)
	// Narrow → single column stack: 6 + 6 + 10 = 22.
	if got := GridHeight(rects); got != 22 {
		t.Errorf("GridHeight = %d, want 22", got)
	}
}
