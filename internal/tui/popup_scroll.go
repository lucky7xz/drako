package tui

import "github.com/charmbracelet/lipgloss"

// One rule for a popup whose rows do not fit the terminal: the rows scroll as
// one block between the popup's pinned top and bottom, with a scrollbar beside
// them. The key sheet, explain and the dropdown all use it, so "taller than
// the screen" has one answer.

// fitRows says how many of total rows show at once in room lines, and the
// largest scroll offset. At least three rows show, however small the terminal.
func fitRows(total, room int) (viewportH, maxOffset int) {
	viewportH = min(total, max(3, room))
	return viewportH, total - viewportH
}

// pageOffset moves offset a page for PgUp and PgDn, one row short of a full
// page so a line of context carries over. ok is false for any other key.
func pageOffset(key string, offset, viewportH, maxOffset int) (next int, ok bool) {
	switch key {
	case "pgup":
		offset -= viewportH - 1
	case "pgdown":
		offset += viewportH - 1
	default:
		return offset, false
	}
	return clampOffset(offset, maxOffset), true
}

func clampOffset(offset, maxOffset int) int {
	return min(max(offset, 0), max(maxOffset, 0))
}

// scrollRows is the block as drawn: viewportH rows from offset, each padded to
// width, with the scrollbar column when there are more rows than show.
func (m Model) scrollRows(lines []string, offset, viewportH, width int) []string {
	fill := m.popupFill()
	offset = clampOffset(offset, len(lines)-viewportH)
	body := padLinesTo(lines[offset:offset+viewportH], width, fill)
	if len(lines) > viewportH {
		bars := scrollbarColumn(len(lines), offset, viewportH, fill.Bold(true), fill.Foreground(hlComment))
		for i := range body {
			body[i] += fill.Render(" ") + bars[i]
		}
	}
	return body
}

// widestLine is the display width of the widest of lines.
func widestLine(lines ...[]string) int {
	w := 0
	for _, group := range lines {
		for _, ln := range group {
			w = max(w, lipgloss.Width(ln))
		}
	}
	return w
}
