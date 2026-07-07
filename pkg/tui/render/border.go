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
	if w < 6 || h < 2 {
		return ""
	}
	lines := make([]string, 0, h)
	lines = append(lines, topBorder(title, w, ps))

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

// topBorder creates: ╭─ TITLE ─────...─────╮ with exact w width.
func topBorder(title string, w int, ps PanelStyle) string {
	if title == "" {
		return ps.Border.Render("╭" + strings.Repeat("─", w-2) + "╮")
	}
	titleUpper := strings.ToUpper(title)
	prefix := "╭─ "
	// Truncate over-long titles so the frame never breaks.
	maxTitle := w - lipgloss.Width(prefix) - 3 // trailing " ─╮" minimum
	if lipgloss.Width(titleUpper) > maxTitle {
		titleUpper = TruncateVisual(titleUpper, maxTitle)
	}
	prefixWidth := lipgloss.Width(prefix + titleUpper + " ")
	dashCount := w - prefixWidth - 1
	if dashCount < 0 {
		dashCount = 0
	}
	return ps.Border.Render(prefix) + ps.Title.Render(titleUpper) +
		ps.Border.Render(" "+strings.Repeat("─", dashCount)+"╮")
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
