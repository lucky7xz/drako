package ui

import "github.com/charmbracelet/lipgloss"

// The grid viewport is stateless: the visible window on each axis is derived
// from the cursor at render time, so no offset needs to be stored or kept in
// sync with cursor movement.

// gridWindow is the visible slice of one grid axis. end is exclusive.
type gridWindow struct {
	start, end                int
	hiddenBefore, hiddenAfter int
}

func (w gridWindow) scrolling() bool {
	return w.hiddenBefore+w.hiddenAfter > 0
}

// window centers the visible span on the cursor and clamps it to the grid.
// Axis-agnostic: used for rows and columns alike.
func window(cursor, total, visible int) gridWindow {
	if total <= 0 {
		return gridWindow{}
	}
	if visible >= total {
		return gridWindow{start: 0, end: total}
	}
	if visible < 1 {
		visible = 1
	}
	cursor = max(0, min(cursor, total-1))
	offset := max(0, min(cursor-visible/2, total-visible))
	return gridWindow{
		start:        offset,
		end:          offset + visible,
		hiddenBefore: offset,
		hiddenAfter:  total - (offset + visible),
	}
}

// visibleCount is how many cells of size cellSize fit in budget. When the
// whole axis fits, it returns total and the grid renders exactly as before.
// Otherwise reserve is subtracted (space for scroll indicators) and at least
// one cell is always shown.
func visibleCount(budget, cellSize, total, reserve int) int {
	if cellSize <= 0 || total <= 0 {
		return total
	}
	if budget/cellSize >= total {
		return total
	}
	return max(1, (budget-reserve)/cellSize)
}

// cellFootprint is the on-screen width of one rendered cell: the widest
// content across ALL cells (so the grid keeps its shape while scrolling),
// capped at GridMaxTextWidth, plus padding (1+1) and border (1+1).
func cellFootprint(grid [][]string) int {
	widest := 0
	for _, row := range grid {
		for _, cell := range row {
			if w := lipgloss.Width(cell); w > widest {
				widest = w
			}
		}
	}
	return min(widest, GridMaxTextWidth) + 4
}

// gridRowBudget is the vertical space left for the grid block after the
// chrome around it. Heights are measured from the rendered strings, not
// estimated: an empty string costs one line, exactly as in JoinVertical.
func gridRowBudget(termHeight int, chrome ...string) int {
	budget := termHeight - appStyle.GetVerticalMargins()
	for _, c := range chrome {
		budget -= lipgloss.Height(c)
	}
	return budget
}
