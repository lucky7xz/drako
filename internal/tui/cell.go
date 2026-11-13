package tui

import (
	"github.com/lucky7xz/drako/internal/launch"
	"github.com/lucky7xz/drako/internal/profile"
)

// A cell is known by the square it stands on. Its name is a label: two cells
// may share one, and then the name says nothing about which is meant.

// cellAt is the command the grid shows at row and col, with the Selection
// that picks it; nil and the zero Selection for an empty square.
func (m Model) cellAt(row, col int) (*profile.Command, launch.Selection) {
	i := profile.ShownAt(m.Config.Commands, m.Config.X, m.Config.Y, row, col)
	if i < 0 {
		return nil, launch.Selection{}
	}
	return &m.Config.Commands[i], launch.Selection{N: i + 1}
}

// cursorCell is the cell under the cursor. A cell without a name is drawn
// empty and cannot be picked.
func (m Model) cursorCell() (*profile.Command, launch.Selection) {
	cell, sel := m.cellAt(m.gridNav.cursorRow, m.gridNav.cursorCol)
	if cell == nil || cell.Name == "" {
		return nil, launch.Selection{}
	}
	return cell, sel
}

// dropdownCell is the folder cell the open dropdown belongs to.
func (m Model) dropdownCell() (*profile.Command, launch.Selection) {
	return m.cellAt(m.dropdown.row, m.dropdown.col)
}
