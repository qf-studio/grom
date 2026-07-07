package render

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PadOrTruncate ensures s is exactly targetWidth visual cells, ANSI-aware.
func PadOrTruncate(s string, targetWidth int) string {
	visualWidth := lipgloss.Width(s)
	if visualWidth == targetWidth {
		return s
	}
	if visualWidth > targetWidth {
		return TruncateVisual(s, targetWidth)
	}
	return s + strings.Repeat(" ", targetWidth-visualWidth)
}

// TruncateVisual truncates s to targetWidth visual cells, appending "..." when
// truncation occurs. ANSI escape sequences (lipgloss color codes) are copied
// through with zero visible width; a CSI sequence starts at ESC (0x1b) and
// ends at a byte in 0x40–0x7e.
func TruncateVisual(s string, targetWidth int) string {
	visualWidth := lipgloss.Width(s)
	if visualWidth <= targetWidth {
		return s
	}
	if targetWidth <= 3 {
		return strings.Repeat(".", targetWidth)
	}

	result := ""
	width := 0
	inEsc := false
	for _, r := range s {
		if inEsc {
			result += string(r)
			if r >= 0x40 && r <= 0x7e {
				inEsc = false
			}
			continue
		}
		if r == 0x1b {
			inEsc = true
			result += string(r)
			continue
		}
		runeWidth := lipgloss.Width(string(r))
		if width+runeWidth > targetWidth-3 {
			break
		}
		result += string(r)
		width += runeWidth
	}

	for width < targetWidth-3 {
		result += " "
		width++
	}
	return result + "..."
}

// Center centers s within width visual cells (ANSI-aware). Strings wider than
// width are truncated.
func Center(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return TruncateVisual(s, width)
	}
	left := (width - w) / 2
	right := width - w - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

// FormatCompact formats a number in compact SI form: 0, 999, 1.0K, 57.3K, 1.2M, 3.4B.
func FormatCompact(v float64) string {
	abs := v
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs < 1000:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return trimZeros(fmt.Sprintf("%.2f", v))
	case abs < 1_000_000:
		return fmt.Sprintf("%.1fK", v/1000)
	case abs < 1_000_000_000:
		return fmt.Sprintf("%.1fM", v/1_000_000)
	default:
		return fmt.Sprintf("%.1fB", v/1_000_000_000)
	}
}

func trimZeros(s string) string {
	s = strings.TrimRight(s, "0")
	return strings.TrimRight(s, ".")
}
