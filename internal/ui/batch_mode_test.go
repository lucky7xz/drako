package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
)

// batchTestModel: 2x2 grid — "one" (a0) and "two" (b0) are runnable, "folder"
// (a1) is a dropdown without its own command, b1 is empty.
func batchTestModel() Model {
	cfg := config.Config{X: 2, Y: 2, Commands: []config.Command{
		{Name: "one", Command: "echo 1", Row: 0, Col: "a"},
		{Name: "two", Command: "echo 2", Row: 0, Col: "b"},
		{Name: "folder", Row: 1, Col: "a", Items: []config.CommandItem{{Name: "child", Command: "true"}}},
	}}
	cfg.ApplyDefaults()
	return Model{
		Config:  cfg,
		styles:  BuildStyles(cfg),
		mode:    batchMode,
		gridNav: gridNav{grid: config.BuildGrid(cfg)},
		batch:   batchState{marked: map[string]bool{}},
	}
}

func pressBatch(t *testing.T, m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.updateBatchMode(msg)
	out, ok := next.(Model)
	if !ok {
		t.Fatalf("updateBatchMode returned %T", next)
	}
	return out, cmd
}

var spaceKey = tea.KeyMsg{Type: tea.KeySpace}

func TestBatchMarkToggle(t *testing.T) {
	m := batchTestModel()

	m, _ = pressBatch(t, m, spaceKey)
	if !m.batch.marked["one"] {
		t.Fatal("space must mark the cursor cell")
	}
	m, _ = pressBatch(t, m, spaceKey)
	if m.batch.marked["one"] {
		t.Fatal("space again must unmark")
	}
}

func TestBatchFolderCellNotMarkable(t *testing.T) {
	m := batchTestModel()
	m.gridNav.cursorRow, m.gridNav.cursorCol = 1, 0 // "folder"

	m, _ = pressBatch(t, m, spaceKey)
	if m.batch.marked["folder"] {
		t.Fatal("cells without a direct command must not be markable")
	}
	if m.profile.statusMessage == "" {
		t.Error("refusing a mark should explain itself in the status line")
	}
}

func TestBatchCapAtNine(t *testing.T) {
	m := batchTestModel()
	for _, n := range []string{"c1", "c2", "c3", "c4", "c5", "c6", "c7", "c8", "c9"} {
		m.batch.marked[n] = true
	}

	m, _ = pressBatch(t, m, spaceKey)
	if m.batch.marked["one"] {
		t.Fatal("the tenth mark must be refused")
	}
	if !strings.Contains(m.profile.statusMessage, "9") {
		t.Errorf("status should state the cap, got %q", m.profile.statusMessage)
	}
}

func TestBatchLaunchCollectsInGridOrder(t *testing.T) {
	m := batchTestModel()
	m.batch.marked["two"] = true // marked out of order
	m.batch.marked["one"] = true

	m, cmd := pressBatch(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	want := []string{"one", "two"} // row-major grid scan, not mark order
	if len(m.SelectedBatch) != 2 || m.SelectedBatch[0] != want[0] || m.SelectedBatch[1] != want[1] {
		t.Fatalf("SelectedBatch = %v, want %v", m.SelectedBatch, want)
	}
	if cmd == nil {
		t.Fatal("launch must quit the TUI to hand the terminal over")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("launch cmd = %T, want tea.QuitMsg", cmd())
	}
	if m.Quitting {
		t.Error("batch launch relaunches the TUI afterwards — Quitting must stay false")
	}
}

func TestBatchLaunchWithNothingMarked(t *testing.T) {
	m := batchTestModel()

	m, cmd := pressBatch(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if len(m.SelectedBatch) != 0 {
		t.Fatal("enter with zero marks must not launch")
	}
	if cmd != nil {
		if _, quit := cmd().(tea.QuitMsg); quit {
			t.Fatal("enter with zero marks must not quit the TUI")
		}
	}
	if m.profile.statusMessage == "" {
		t.Error("empty launch should explain itself")
	}
}

func TestBatchEscCancelsAndClears(t *testing.T) {
	m := batchTestModel()
	m.batch.marked["one"] = true

	m, _ = pressBatch(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != gridMode {
		t.Fatalf("esc must return to grid mode, got %v", m.mode)
	}
	if len(m.batch.marked) != 0 {
		t.Error("cancel must clear the marks")
	}
}

func TestBatchNavigationStillMoves(t *testing.T) {
	m := batchTestModel()

	m, _ = pressBatch(t, m, tea.KeyMsg{Type: tea.KeyRight})
	if m.gridNav.cursorCol != 1 {
		t.Errorf("navigation must keep working in batch mode, col = %d", m.gridNav.cursorCol)
	}
}

func TestBatchQuickNavJumps(t *testing.T) {
	m := batchTestModel()

	m, _ = pressBatch(t, m, keyRunes("2"))
	if m.gridNav.cursorCol != 1 {
		t.Fatalf("quick-nav must work in batch mode: col = %d, want 1", m.gridNav.cursorCol)
	}

	// The jump composes with marking: space marks the cell we jumped to.
	m, _ = pressBatch(t, m, spaceKey)
	if !m.batch.marked["two"] {
		t.Error("space after a quick-nav jump should mark the target cell")
	}
}

func TestBatchViewShowsMarksAndCounter(t *testing.T) {
	m := batchTestModel()
	m.termWidth, m.termHeight = 100, 40
	m.batch.marked["one"] = true

	out := m.View()
	if !strings.Contains(out, "BATCH 1/9") {
		t.Errorf("view should show the batch counter, got:\n%s", out)
	}
	if !strings.Contains(out, "◉") || !strings.Contains(out, "○") {
		t.Errorf("view should show marked and unmarked glyphs, got:\n%s", out)
	}
}
