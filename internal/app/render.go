package app

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/qf-studio/grot/internal/config"
	"github.com/qf-studio/grot/pkg/tui/theme"
	"github.com/qf-studio/grot/pkg/tui/widget"
)

// StaticFrame renders a one-shot header + grid at the given viewport, with no
// focus. Widgets must already hold their fetched results.
func StaticFrame(dash *config.Dashboard, widgets []widget.Widget, th theme.Theme, termW, termH int) string {
	rects := GridLayout(dash.Widgets, termW, termH)
	title := coalesce(dash.Title, "grot")
	header := headerLine(title, th, formatRange(dash.Range.Duration()), false, nil)
	return header + "\n" + composeGrid(widgets, rects, th, -1)
}

// headerLine is the quiet top strip: bold title, then dim clock · theme · range,
// plus an amber stale hint when any widget's data has aged out.
func headerLine(title string, th theme.Theme, rangeLabel string, zoomed bool, stale []string) string {
	b := new(strings.Builder)
	b.WriteString(th.AccentStyle().Bold(true).Render(" " + title))
	meta := "  ·  " + time.Now().Format("15:04:05") + "  ·  " + th.Name + "  ·  " + rangeLabel
	if zoomed {
		meta += "  ·  zoom"
	}
	b.WriteString(th.DimStyle().Render(meta))
	if len(stale) > 0 {
		b.WriteString(th.WarningStyle().Render("  ·  stale: " + strings.Join(stale, ", ")))
	}
	return b.String()
}

// composeGrid draws widgets into their rects. Rects sharing a Y form one row,
// laid left-to-right in X order with blank spacers filling any horizontal gaps;
// rows stack top to bottom. focus is the index drawn focused (-1 for none).
func composeGrid(widgets []widget.Widget, rects []Rect, th theme.Theme, focus int) string {
	if len(widgets) == 0 {
		return ""
	}
	rowOf := map[int][]int{}
	var ys []int
	for i, r := range rects {
		if _, seen := rowOf[r.Y]; !seen {
			ys = append(ys, r.Y)
		}
		rowOf[r.Y] = append(rowOf[r.Y], i)
	}
	sort.Ints(ys)

	var rows []string
	for _, y := range ys {
		idxs := rowOf[y]
		sort.Slice(idxs, func(a, b int) bool { return rects[idxs[a]].X < rects[idxs[b]].X })
		rowH := 0
		for _, i := range idxs {
			if rects[i].H > rowH {
				rowH = rects[i].H
			}
		}
		var cols []string
		cursor := 0
		for _, i := range idxs {
			if rects[i].X > cursor {
				cols = append(cols, blankCell(rects[i].X-cursor, rowH))
			}
			cols = append(cols, safeRender(widgets[i], rects[i].W, rects[i].H, th, i == focus))
			cursor = rects[i].X + rects[i].W
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cols...))
	}
	return strings.Join(rows, "\n")
}

// safeRender renders a widget, but never below its declared MinSize — a cell
// squeezed smaller than that (e.g. mid-resize, or in a very narrow terminal)
// gets a blank box of the exact requested size instead. Widgets assume at least
// MinSize; handing them less underflows their internal width math and panics.
func safeRender(wd widget.Widget, width, height int, th theme.Theme, focused bool) string {
	mw, mh := wd.MinSize()
	if width < mw || height < mh {
		return blankCell(width, height)
	}
	return wd.Render(width, height, th, focused)
}

// blankCell is a width×height block of spaces (dimensions clamped to zero).
func blankCell(width, height int) string {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	if height == 0 {
		return ""
	}
	line := strings.Repeat(" ", width)
	lines := make([]string, height)
	for i := range lines {
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

// formatRange renders a window duration compactly for the header (90s → "90s",
// 15m → "15m", 3h → "3h").
func formatRange(d time.Duration) string {
	switch {
	case d <= 0:
		return "—"
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		if d%time.Hour == 0 {
			return fmt.Sprintf("%dh", int(d.Hours()))
		}
		return fmt.Sprintf("%.1fh", d.Hours())
	}
}

// coalesce returns the first non-empty string.
func coalesce(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
