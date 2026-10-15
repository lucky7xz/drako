package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
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
		if !m.gridNav.navPending {
			t.Error("expected quicknav to await the row digit")
		}
		if cmd == nil {
			t.Error("expected a timeout command")
		}
	})

	t.Run("column index clamps to last populated column", func(t *testing.T) {
		m := quickNavModel()
		tm, _ := m.updateGridMode(keyRune("9")) // clamps to col 2
		m = tm.(Model)
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
		if m.gridNav.navPending {
			t.Error("expected quicknav sequence ended after second press")
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

	t.Run("an empty column keeps the cursor on the grid", func(t *testing.T) {
		m := quickNavModel()
		m.gridNav.grid[1][1] = "" // column 1 now has no cells
		tm, _ := m.updateGridMode(keyRune("2"))
		m = tm.(Model)
		tm, _ = m.updateGridMode(keyRune("1"))
		m = tm.(Model)
		if m.gridNav.cursorRow != 0 || m.gridNav.cursorCol != 1 {
			t.Errorf("cursor = (%d,%d), want (0,1)", m.gridNav.cursorRow, m.gridNav.cursorCol)
		}
		m.updateGridMode(keyRune("e")) // indexed the grid at row -1 and panicked
	})
}

// Each digit lights its own label, the column letter and then the row number,
// and nothing else on the grid changes.
func TestQuickNavLightsOnlyTheLabel(t *testing.T) {
	forceTrueColor(t)
	for name, mode := range map[string]navMode{"grid": gridMode, "batch": batchMode} {
		t.Run(name, func(t *testing.T) {
			m := makeScrollTestModel(3, 3, 120, 60)
			m.mode = mode

			tm, _ := m.quickNav(1) // column B
			m = tm.(Model)
			rest := m
			rest.gridNav.navPending = false
			assertOnlyLit(t, m.renderGrid(40), rest.renderGrid(40), m.styles.PickedLabel, "[B]")

			tm, _ = m.quickNav(2) // row 3
			m = tm.(Model)
			rest = m
			rest.gridNav.flashActive = false
			assertOnlyLit(t, m.renderGrid(40), rest.renderGrid(40), m.styles.PickedLabel, "3❭")
		})
	}
}

// assertOnlyLit checks that got is rest with exactly one label lit.
func assertOnlyLit(t *testing.T, got, rest string, lit lipgloss.Style, label string) {
	t.Helper()
	on := lit.Render(label)
	if on == label || !strings.Contains(got, on) {
		t.Fatalf("%s is not lit:\n%s", label, got)
	}
	if strings.Replace(got, on, label, 1) != rest {
		t.Errorf("lighting %s changed more than the label:\n%s", label, got)
	}
}
