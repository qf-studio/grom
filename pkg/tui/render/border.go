package render

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PanelStyle carries the styles used to draw a panel frame.
type PanelStyle struct {
	Border lipgloss.Style // frame characters
	Title  lipgloss.Style // title text in the top border
}

// Panel draws a bordered panel of exactly w×h cells:
//
//	╭─ TITLE ────────╮
//	│ content        │
//	╰────────────────╯
//
// Content lines beyond h-2 are dropped; missing lines are padded blank.
// Each content line is padded/truncated to the inner width (w-4).
func Panel(title, content string, w, h int, ps PanelStyle) string {
	return PanelInfo(title, "", content, w, h, ps)
}

// PanelInfo draws a Panel with extra info embedded in the right side of the
// top border, btop-style:
//
//	╭─ TITLE ─────┤ info ├─╮
//
// info may contain styled (ANSI) text; it is truncated if too wide.
func PanelInfo(title, info, content string, w, h int, ps PanelStyle) string {
	if w < 6 || h < 2 {
		return ""
	}
	lines := make([]string, 0, h)
	lines = append(lines, topBorderInfo(title, info, w, ps))

	contentLines := strings.Split(content, "\n")
	innerRows := h - 2
	for i := 0; i < innerRows; i++ {
		if i < len(contentLines) {
			lines = append(lines, contentLine(contentLines[i], w, ps))
		} else {
			lines = append(lines, emptyLine(w, ps))
		}
	}

	lines = append(lines, bottomBorder(w, ps))
	return strings.Join(lines, "\n")
}

// InnerSize returns the content area dimensions for a Panel of w×h.
func InnerSize(w, h int) (int, int) {
	return w - 4, h - 2
}

// topBorderInfo creates: ╭─ TITLE ────┤ info ├─╮ with exact w width.
// info is optional; when absent the border is plain dashes to ╮.
func topBorderInfo(title, info string, w int, ps PanelStyle) string {
	var left string
	leftWidth := 1 // ╭
	if title != "" {
		titleUpper := strings.ToUpper(title)
		prefix := "╭─ "
		maxTitle := w - lipgloss.Width(prefix) - 3 // trailing " ─╮" minimum
		if lipgloss.Width(titleUpper) > maxTitle {
			titleUpper = TruncateVisual(titleUpper, maxTitle)
		}
		left = ps.Border.Render(prefix) + ps.Title.Render(titleUpper) + ps.Border.Render(" ")
		leftWidth = lipgloss.Width(prefix + titleUpper + " ")
	} else {
		left = ps.Border.Render("╭")
	}

	// Right segment: ┤ info ├─╮ (only when info fits).
	var right string
	rightWidth := 1 // ╮
	if info != "" {
		maxInfo := w - leftWidth - 8 // room for caps + some dashes
		if maxInfo > 0 {
			if lipgloss.Width(info) > maxInfo {
				info = TruncateVisual(info, maxInfo)
			}
			right = ps.Border.Render("┤ ") + info + ps.Border.Render(" ├─╮")
			rightWidth = lipgloss.Width("┤ ") + lipgloss.Width(info) + lipgloss.Width(" ├─╮")
		}
	}
	if right == "" {
		right = ps.Border.Render("╮")
	}

	dashCount := w - leftWidth - rightWidth
	if dashCount < 0 {
		dashCount = 0
	}
	return left + ps.Border.Render(strings.Repeat("─", dashCount)) + right
}

func bottomBorder(w int, ps PanelStyle) string {
	return ps.Border.Render("╰" + strings.Repeat("─", w-2) + "╯")
}

func emptyLine(w int, ps PanelStyle) string {
	b := ps.Border.Render("│")
	return b + strings.Repeat(" ", w-2) + b
}

func contentLine(content string, w int, ps PanelStyle) string {
	adjusted := PadOrTruncate(content, w-4)
	b := ps.Border.Render("│")
	return b + " " + adjusted + " " + b
}
