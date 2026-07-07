package app

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/qf-studio/grot/internal/config"
	"github.com/qf-studio/grot/internal/datasource"
	"github.com/qf-studio/grot/pkg/tui/theme"
	"github.com/qf-studio/grot/pkg/tui/widget"
)

// rangePresets are the windows the +/- keys step through.
var rangePresets = []time.Duration{
	5 * time.Minute, 15 * time.Minute, 30 * time.Minute,
	1 * time.Hour, 3 * time.Hour, 6 * time.Hour, 12 * time.Hour, 24 * time.Hour,
}

// tickMsg fires on the refresh interval; fetchedMsg delivers one widget's data.
type (
	tickMsg    time.Time
	fetchedMsg struct {
		idx int
		res widget.QueryResult
		err error
	}
)

// Model is grot's Bubble Tea application: a focusable, zoomable grid of widgets
// backed by a polling datasource. Fetches run as concurrent Cmds guarded so a
// slow query never blocks Update or stacks up behind itself.
type Model struct {
	dash    *config.Dashboard
	specs   []config.WidgetSpec
	widgets []widget.Widget
	ds      datasource.Datasource

	th       theme.Theme
	themeIdx int

	width, height int
	rects         []Rect
	focus         int
	zoomed        bool
	scroll        int
	ready         bool

	rng      time.Duration
	rangeIdx int
	refresh  time.Duration

	inflight  []bool
	fetchedAt []time.Time
}

// NewModel builds the application model. Widgets must already be constructed
// (see BuildWidget); data is fetched by the running program.
func NewModel(dash *config.Dashboard, widgets []widget.Widget, ds datasource.Datasource, th theme.Theme) Model {
	return Model{
		dash:      dash,
		specs:     dash.Widgets,
		widgets:   widgets,
		ds:        ds,
		th:        th,
		themeIdx:  themeIndex(th.Name),
		rng:       dash.Range.Duration(),
		rangeIdx:  nearestRange(dash.Range.Duration()),
		refresh:   dash.Refresh.Duration(),
		inflight:  make([]bool, len(widgets)),
		fetchedAt: make([]time.Time, len(widgets)),
	}
}

// RunTUI runs the interactive dashboard until quit, Ctrl-C, or ctx cancel.
func RunTUI(ctx context.Context, dash *config.Dashboard, widgets []widget.Widget, ds datasource.Datasource, th theme.Theme) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	p := tea.NewProgram(
		NewModel(dash, widgets, ds, th),
		tea.WithAltScreen(),
		tea.WithContext(ctx),
	)
	_, err := p.Run()
	return err
}

func (m Model) Init() tea.Cmd { return m.tickCmd() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.rects = GridLayout(m.specs, m.width, m.contentHeight())
		m.clampFocus()
		m.ensureVisible()
		if !m.ready {
			m.ready = true
			return m, m.fetchDueCmd() // first paint has dimensions → fetch now
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tickMsg:
		return m, tea.Batch(m.fetchDueCmd(), m.tickCmd())

	case fetchedMsg:
		if msg.idx >= 0 && msg.idx < len(m.widgets) {
			m.inflight[msg.idx] = false
			m.fetchedAt[msg.idx] = time.Now()
			if msg.err != nil {
				m.widgets[msg.idx].SetError(msg.err)
			} else {
				m.widgets[msg.idx].SetResult(msg.res)
			}
		}
		return m, nil
	}
	return m, nil
}

func (m Model) View() string {
	if !m.ready {
		return ""
	}
	header := headerLine(coalesce(m.dash.Title, "grot"), m.th, formatRange(m.rng), m.zoomed, m.staleNames())
	if m.zoomed && len(m.widgets) > 0 {
		r := m.fetchRect(m.focus)
		return header + "\n" + safeRender(m.widgets[m.focus], r.W, r.H, m.th, true)
	}
	grid := composeGrid(m.widgets, m.rects, m.th, m.focus)
	return header + "\n" + cropVertical(grid, m.scroll, m.contentHeight())
}

func (m Model) handleKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "h", "left":
		m.focus = focusMove(m.rects, m.focus, 'h')
		m.ensureVisible()
	case "j", "down":
		m.focus = focusMove(m.rects, m.focus, 'j')
		m.ensureVisible()
	case "k", "up":
		m.focus = focusMove(m.rects, m.focus, 'k')
		m.ensureVisible()
	case "l", "right":
		m.focus = focusMove(m.rects, m.focus, 'l')
		m.ensureVisible()
	case "enter":
		m.zoomed = !m.zoomed
	case "+", "=":
		if m.rangeIdx < len(rangePresets)-1 {
			m.rangeIdx++
			m.rng = rangePresets[m.rangeIdx]
			return m, m.fetchDueCmd()
		}
	case "-", "_":
		if m.rangeIdx > 0 {
			m.rangeIdx--
			m.rng = rangePresets[m.rangeIdx]
			return m, m.fetchDueCmd()
		}
	case "r":
		return m, m.fetchDueCmd()
	case "t":
		m.themeIdx = (m.themeIdx + 1) % len(theme.All)
		m.th = theme.All[m.themeIdx]
	}
	return m, nil
}

// fetchDueCmd dispatches a fetch for every widget not already in flight,
// marking them so they don't stack. Returns nil when all are busy.
func (m *Model) fetchDueCmd() tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.widgets {
		if m.inflight[i] {
			continue
		}
		m.inflight[i] = true
		cmds = append(cmds, m.fetchCmd(i))
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// fetchCmd captures the current step/range/width for widget i and fetches off
// the Update goroutine.
func (m Model) fetchCmd(i int) tea.Cmd {
	spec := m.specs[i]
	cw := chartWidth(m.fetchRect(i))
	rng := m.rng
	ds := m.ds
	timeout := m.timeout()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		res, err := FetchWidget(ctx, ds, spec, cw, rng)
		return fetchedMsg{idx: i, res: res, err: err}
	}
}

// fetchRect is the rectangle a widget currently occupies — the full content
// area when it is the zoomed widget, otherwise its grid cell.
func (m Model) fetchRect(i int) Rect {
	if m.zoomed && i == m.focus {
		return Rect{W: m.width, H: m.contentHeight()}
	}
	if i >= 0 && i < len(m.rects) {
		return m.rects[i]
	}
	return Rect{}
}

// staleNames lists widgets whose freshest data is older than twice the refresh
// interval — surfaced as an amber hint in the header.
func (m Model) staleNames() []string {
	if m.refresh <= 0 {
		return nil
	}
	var names []string
	for i, at := range m.fetchedAt {
		if at.IsZero() {
			continue
		}
		if time.Since(at) > 2*m.refresh {
			names = append(names, m.widgets[i].Title())
		}
	}
	return names
}

func (m *Model) ensureVisible() {
	if m.zoomed || m.focus < 0 || m.focus >= len(m.rects) {
		return
	}
	r := m.rects[m.focus]
	ch := m.contentHeight()
	if r.Y < m.scroll {
		m.scroll = r.Y
	}
	if r.Y+r.H > m.scroll+ch {
		m.scroll = r.Y + r.H - ch
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
}

func (m *Model) clampFocus() {
	if m.focus >= len(m.widgets) {
		m.focus = len(m.widgets) - 1
	}
	if m.focus < 0 {
		m.focus = 0
	}
}

func (m Model) contentHeight() int {
	h := m.height - 1 // header row
	if h < 1 {
		return 1
	}
	return h
}

func (m Model) timeout() time.Duration {
	t := 10 * time.Second
	if m.refresh > 0 && m.refresh < t {
		t = m.refresh
	}
	return t
}

func (m Model) tickCmd() tea.Cmd {
	d := m.refresh
	if d <= 0 {
		d = 30 * time.Second
	}
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// cropVertical returns the height lines of s starting at offset, for scroll.
func cropVertical(s string, offset, height int) string {
	lines := strings.Split(s, "\n")
	if offset < 0 {
		offset = 0
	}
	if offset > len(lines) {
		offset = len(lines)
	}
	end := offset + height
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[offset:end], "\n")
}

func nearestRange(d time.Duration) int {
	best, bestDiff := 0, absDur(rangePresets[0]-d)
	for i, p := range rangePresets {
		if diff := absDur(p - d); diff < bestDiff {
			best, bestDiff = i, diff
		}
	}
	return best
}

func absDur(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func themeIndex(name string) int {
	for i, t := range theme.All {
		if t.Name == name {
			return i
		}
	}
	return 0
}
