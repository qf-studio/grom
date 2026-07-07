package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/qf-studio/grot/pkg/tui/render"
	"github.com/qf-studio/grot/pkg/tui/theme"
)

// helpBindings lists the dashboard keybindings shown by the ? overlay.
var helpBindings = [][2]string{
	{"hjkl / arrows", "move focus"},
	{"enter", "zoom the focused panel"},
	{"esc", "close zoom (or this help)"},
	{"+ / -", "widen / narrow the time range"},
	{"r", "refresh now"},
	{"t", "cycle theme"},
	{"?", "toggle this help"},
	{"q / ctrl-c", "quit"},
}

// helpView renders the keybinding overlay as a quiet centered panel filling a
// w×h viewport. It replaces the grid while shown (a cell grid has no
// z-compositing); any key press dismisses it.
func helpView(th theme.Theme, w, h int) string {
	keyW, descW := 0, 0
	for _, b := range helpBindings {
		if kw := lipgloss.Width(b[0]); kw > keyW {
			keyW = kw
		}
		if dw := lipgloss.Width(b[1]); dw > descW {
			descW = dw
		}
	}

	lines := make([]string, 0, len(helpBindings)+2)
	lines = append(lines, "")
	for _, b := range helpBindings {
		key := th.LabelStyle().Render(render.PadOrTruncate(b[0], keyW))
		lines = append(lines, key+"   "+th.DimStyle().Render(b[1]))
	}
	lines = append(lines, "")

	pw := keyW + 3 + descW + 4 // columns + gap + panel chrome
	ph := len(lines) + 2
	if pw > w {
		pw = w
	}
	if ph > h {
		ph = h
	}
	ps := render.PanelStyle{Border: th.BorderStyle(), Title: th.TitleStyle()}
	panel := render.Panel("keys", strings.Join(lines, "\n"), pw, ph, ps)
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, panel)
}
