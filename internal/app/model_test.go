package app

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/qf-studio/grot/internal/config"
	"github.com/qf-studio/grot/internal/datasource"
	"github.com/qf-studio/grot/pkg/tui/theme"
	"github.com/qf-studio/grot/pkg/tui/widget"
)

// fakeDS returns canned data and counts calls.
type fakeDS struct{ instant, rng int }

func (f *fakeDS) QueryInstant(context.Context, datasource.Instant) ([]widget.Series, error) {
	f.instant++
	return []widget.Series{{Legend: "v", Points: []widget.Point{{V: 42}}}}, nil
}

func (f *fakeDS) QueryRange(context.Context, datasource.Range) ([]widget.Series, error) {
	f.rng++
	return []widget.Series{{Legend: "s", Points: []widget.Point{{V: 1}, {V: 2}, {V: 3}}}}, nil
}

func runeKey(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

func drive(m Model, msg tea.Msg) Model {
	nm, _ := m.Update(msg)
	return nm.(Model)
}

func newModel(t *testing.T) (Model, *fakeDS) {
	t.Helper()
	ds := &fakeDS{}
	dash := &config.Dashboard{
		Title:   "test",
		Theme:   "pilot",
		Refresh: config.Duration(30 * time.Second),
		Range:   config.Duration(15 * time.Minute),
		Widgets: []config.WidgetSpec{
			{Type: config.TypeStat, Title: "a", Queries: []config.Query{{Expr: "x"}}},
			{Type: config.TypeStat, Title: "b", Queries: []config.Query{{Expr: "y"}}},
			{Type: config.TypeTimeSeries, Title: "c", Queries: []config.Query{{Expr: "z"}}},
		},
	}
	widgets := make([]widget.Widget, len(dash.Widgets))
	for i, spec := range dash.Widgets {
		w, err := BuildWidget(spec)
		if err != nil {
			t.Fatalf("BuildWidget: %v", err)
		}
		widgets[i] = w
	}
	return NewModel(dash, widgets, ds, theme.Pilot), ds
}

func TestModelReadyAndView(t *testing.T) {
	m, _ := newModel(t)
	if m.View() != "" {
		t.Error("View before sizing should be empty")
	}
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})
	if !m.ready {
		t.Fatal("WindowSizeMsg should mark model ready")
	}
	if len(m.rects) != 3 {
		t.Fatalf("expected 3 rects, got %d", len(m.rects))
	}
	if v := m.View(); !strings.Contains(v, "test") {
		t.Errorf("View missing title; got:\n%s", v)
	}
}

func TestModelFocusMovement(t *testing.T) {
	m, _ := newModel(t)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 40}) // 2 per row
	if m.focus != 0 {
		t.Fatalf("initial focus = %d, want 0", m.focus)
	}
	m = drive(m, runeKey('l'))
	if m.focus != 1 {
		t.Errorf("after 'l', focus = %d, want 1", m.focus)
	}
	m = drive(m, runeKey('h'))
	if m.focus != 0 {
		t.Errorf("after 'h', focus = %d, want 0", m.focus)
	}
	m = drive(m, runeKey('j'))
	if m.focus == 0 {
		t.Errorf("after 'j', focus should leave row 0, still %d", m.focus)
	}
}

func TestModelZoomToggle(t *testing.T) {
	m, _ := newModel(t)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})
	m = drive(m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.zoomed {
		t.Error("Enter should zoom")
	}
	m = drive(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.zoomed {
		t.Error("Enter again should unzoom")
	}
}

func TestModelRangePresets(t *testing.T) {
	m, _ := newModel(t)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})
	if m.rng != 15*time.Minute {
		t.Fatalf("initial range = %s, want 15m", m.rng)
	}
	m = drive(m, runeKey('+'))
	if m.rng != 30*time.Minute {
		t.Errorf("after '+', range = %s, want 30m", m.rng)
	}
	m = drive(m, runeKey('-'))
	m = drive(m, runeKey('-'))
	if m.rng != 5*time.Minute {
		t.Errorf("after '--', range = %s, want 5m", m.rng)
	}
	// Clamp at the low end.
	m = drive(m, runeKey('-'))
	if m.rng != 5*time.Minute {
		t.Errorf("range should clamp at 5m, got %s", m.rng)
	}
}

func TestModelThemeCycle(t *testing.T) {
	m, _ := newModel(t)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})
	start := m.th.Name
	m = drive(m, runeKey('t'))
	if m.th.Name == start {
		t.Errorf("'t' should change theme from %s", start)
	}
}

func TestModelQuit(t *testing.T) {
	m, _ := newModel(t)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})
	_, cmd := m.Update(runeKey('q'))
	if cmd == nil {
		t.Fatal("'q' should return a command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("'q' should return tea.Quit")
	}
}

func TestModelFetchCmdAndResult(t *testing.T) {
	m, ds := newModel(t)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})

	// Stat widget 0 → instant; timeseries widget 2 → range.
	if msg, ok := m.fetchCmd(0)().(fetchedMsg); !ok || msg.idx != 0 || msg.err != nil {
		t.Errorf("fetchCmd(0) bad msg: %#v", msg)
	}
	if msg, ok := m.fetchCmd(2)().(fetchedMsg); !ok || msg.idx != 2 || msg.err != nil {
		t.Errorf("fetchCmd(2) bad msg: %#v", msg)
	}
	if ds.instant == 0 || ds.rng == 0 {
		t.Errorf("expected both instant and range queries; got instant=%d range=%d", ds.instant, ds.rng)
	}

	// Delivering a result clears in-flight and stamps fetchedAt.
	m.inflight[0] = true
	m = drive(m, fetchedMsg{idx: 0, res: widget.QueryResult{FetchedAt: time.Now()}})
	if m.inflight[0] {
		t.Error("fetchedMsg should clear in-flight")
	}
	if m.fetchedAt[0].IsZero() {
		t.Error("fetchedMsg should stamp fetchedAt")
	}
}

func TestModelInFlightGuard(t *testing.T) {
	m, _ := newModel(t)
	// Becoming ready dispatches the initial fetch, marking every widget in flight.
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})
	for i, f := range m.inflight {
		if !f {
			t.Errorf("widget %d should be in-flight after initial dispatch", i)
		}
	}
	// A dispatch while all are busy is a no-op — slow queries can't stack.
	if cmd := m.fetchDueCmd(); cmd != nil {
		t.Error("fetchDueCmd should be a no-op while all in flight")
	}
	// Completing the fetches frees them...
	for i := range m.widgets {
		m = drive(m, fetchedMsg{idx: i, res: widget.QueryResult{FetchedAt: time.Now()}})
	}
	for i, f := range m.inflight {
		if f {
			t.Errorf("widget %d still in-flight after result", i)
		}
	}
	// ...so the next dispatch runs again.
	if cmd := m.fetchDueCmd(); cmd == nil {
		t.Fatal("fetchDueCmd should dispatch once widgets are idle")
	}
}

// TestSafeRenderTiny ensures a widget squeezed below its MinSize yields a blank
// box of the exact size instead of panicking on underflowed width math.
func TestSafeRenderTiny(t *testing.T) {
	w, err := BuildWidget(config.WidgetSpec{Type: config.TypeTimeSeries, Title: "c", Queries: []config.Query{{Expr: "z"}}})
	if err != nil {
		t.Fatalf("BuildWidget: %v", err)
	}
	for _, sz := range []struct{ w, h int }{{0, 0}, {1, 1}, {3, 2}, {10, 3}} {
		out := safeRender(w, sz.w, sz.h, theme.Pilot, false) // must not panic
		lines := 0
		if out != "" {
			lines = strings.Count(out, "\n") + 1
		}
		if sz.h > 0 && sz.w >= 0 && lines > sz.h {
			t.Errorf("safeRender(%dx%d) produced %d lines, want <= %d", sz.w, sz.h, lines, sz.h)
		}
	}
}
