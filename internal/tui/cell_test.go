package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/profile"
	"github.com/lucky7xz/drako/internal/settings"
)

// sameNameModel: two cells called "same" side by side, and a third cell that
// shares the first one's square in a second deck row.
func sameNameModel(mode navMode) Model {
	cfg := settings.Config{X: 2, Y: 1, Commands: []profile.Command{
		{Name: "same", Command: "echo FIRST", Row: 0, Col: "a"},
		{Name: "same", Command: "echo SECOND", Row: 0, Col: "b"},
	}}
	cfg.ApplyDefaults()
	return Model{
		Config:  cfg,
		styles:  BuildStyles(cfg, ""),
		mode:    mode,
		gridNav: gridNav{grid: profile.BuildGrid(cfg.X, cfg.Y, cfg.Commands), cursorCol: 1},
	}
}

// I-01: with two cells of one name, the cell under the cursor is the one
// explained and the one run.
func TestSameNamedCells_TheCellUnderTheCursorRuns(t *testing.T) {
	m := sameNameModel(gridMode)

	next, _ := m.updateGridMode(keyRune("e"))
	if got := next.(Model).activeDetail; got == nil || got.Value != "echo SECOND" {
		t.Fatalf("explain shows %+v, want the second cell's command", got)
	}

	next, _ = m.updateGridMode(tea.KeyMsg{Type: tea.KeyEnter})
	out := next.(Model).Outcome()
	if out.Action != Run || len(out.Plans) != 1 || !strings.Contains(strings.Join(out.Plans[0].Argv, " "), "echo SECOND") {
		t.Fatalf("enter runs %+v, want the second cell's command", out)
	}
}

// B-01: marking the second cell of a name marks it, and leaves the first alone.
func TestSameNamedCells_AreMarkedApart(t *testing.T) {
	m := sameNameModel(batchMode)
	m.gridNav.cursorCol = 0
	m, _ = pressBatch(t, m, spaceKey)
	m.gridNav.cursorCol = 1
	m, _ = pressBatch(t, m, spaceKey)
	if len(m.batch.marked) != 2 || m.batch.marked[0] == m.batch.marked[1] {
		t.Fatalf("marks = %+v, want both cells, told apart", m.batch.marked)
	}
}

// After a run the cursor returns to the square it was on, not to the first
// cell of that name.
func TestSameNamedCells_TheCursorReturnsToItsSquare(t *testing.T) {
	m := sameNameModel(gridMode)
	s := m.Session()
	m.gridNav.cursorCol = 0
	m.Restore(s)
	if m.gridNav.cursorCol != 1 {
		t.Fatalf("cursor on column %d, want 1", m.gridNav.cursorCol)
	}
}

// A cell that moved while a command ran is still followed by its name.
func TestRestoreFollowsACellThatMoved(t *testing.T) {
	m := sameNameModel(gridMode)
	m.gridNav.grid = [][]string{{"other", "moved"}}
	m.Restore(Session{Cell: "moved", Row: 0, Col: 0})
	if m.gridNav.cursorCol != 1 {
		t.Fatalf("cursor on column %d, want 1", m.gridNav.cursorCol)
	}
}
