package ui

import (
	"testing"

	"github.com/lucky7xz/drako/internal/config"
)

// quickNavModel builds a grid model with a known sparse layout for driving
// the 1-9 quick-navigation sequence.
//
//	col:   0    1    2
//	row0   A    .    C
//	row1   .    E    F
//	row2   G    .    .
func quickNavModel() Model {
	return Model{
		mode: gridMode,
		gridNav: gridNav{
			grid: [][]string{
				{"A", "", "C"},
				{"", "E", "F"},
				{"G", "", ""},
			},
		},
		Config: config.Config{Keys: config.InputConfig{}},
	}
}

func TestQuickNav_Characterization(t *testing.T) {
	t.Run("first press selects column, parks on its first populated row", func(t *testing.T) {
		m := quickNavModel()
		tm, cmd := m.updateGridMode(keyRune("3")) // column index 2
		m = tm.(Model)
		if m.gridNav.cursorCol != 2 || m.gridNav.cursorRow != 0 {
			t.Errorf("cursor = (%d,%d), want (0,2)", m.gridNav.cursorRow, m.gridNav.cursorCol)
		}
		if m.gridNav.timer == nil {
			t.Error("expected quicknav timer to be armed")
		} else {
			m.gridNav.timer.Stop()
		}
		if cmd == nil {
			t.Error("expected a timeout command")
		}
	})

	t.Run("column index clamps to last populated column", func(t *testing.T) {
		m := quickNavModel()
		tm, _ := m.updateGridMode(keyRune("9")) // clamps to col 2
		m = tm.(Model)
		if m.gridNav.timer != nil {
			m.gridNav.timer.Stop()
		}
		if m.gridNav.cursorCol != 2 {
			t.Errorf("cursorCol = %d, want 2 (clamped)", m.gridNav.cursorCol)
		}
	})

	t.Run("second press selects row within the chosen column", func(t *testing.T) {
		m := quickNavModel()
		tm, _ := m.updateGridMode(keyRune("3")) // first press: column 2
		m = tm.(Model)
		tm, _ = m.updateGridMode(keyRune("2")) // second press: row index 1
		m = tm.(Model)
		if m.gridNav.cursorRow != 1 || m.gridNav.cursorCol != 2 {
			t.Errorf("cursor = (%d,%d), want (1,2)", m.gridNav.cursorRow, m.gridNav.cursorCol)
		}
		if m.gridNav.timer != nil {
			t.Error("expected quicknav timer cleared after second press")
		}
	})

	t.Run("row index clamps to last populated row in the column", func(t *testing.T) {
		m := quickNavModel()
		tm, _ := m.updateGridMode(keyRune("1")) // column 0 (rows 0 and 2 populated)
		m = tm.(Model)
		tm, _ = m.updateGridMode(keyRune("9")) // clamps to last populated row = 2
		m = tm.(Model)
		if m.gridNav.cursorRow != 2 || m.gridNav.cursorCol != 0 {
			t.Errorf("cursor = (%d,%d), want (2,0)", m.gridNav.cursorRow, m.gridNav.cursorCol)
		}
	})
}
