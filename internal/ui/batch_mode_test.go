package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
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
		batch:   batchState{},
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
	if m.batch.mark("one") != 1 {
		t.Fatal("space must mark the cursor cell")
	}
	m, _ = pressBatch(t, m, spaceKey)
	if m.batch.mark("one") != 0 {
		t.Fatal("space again must unmark")
	}
}

func TestBatchFolderCellNotMarkable(t *testing.T) {
	m := batchTestModel()
	m.gridNav.cursorRow, m.gridNav.cursorCol = 1, 0 // "folder"

	m, _ = pressBatch(t, m, spaceKey)
	if m.batch.mark("folder") != 0 {
		t.Fatal("cells without a direct command must not be markable")
	}
	if m.profile.statusMessage == "" {
		t.Error("refusing a mark should explain itself in the status line")
	}
}

func TestBatchCapAtNine(t *testing.T) {
	m := batchTestModel()
	m.batch.marked = []string{"c1", "c2", "c3", "c4", "c5", "c6", "c7", "c8", "c9"}

	m, _ = pressBatch(t, m, spaceKey)
	if m.batch.mark("one") != 0 {
		t.Fatal("the tenth mark must be refused")
	}
	if !strings.Contains(m.profile.statusMessage, "9") {
		t.Errorf("status should state the cap, got %q", m.profile.statusMessage)
	}
}

func TestBatchLaunchCollectsInSelectionOrder(t *testing.T) {
	m := batchTestModel()
	// Mark "two" first, then "one" — the launch must follow the marking,
	// not the grid, because the order decides how cells group into tabs.
	m, _ = pressBatch(t, m, tea.KeyMsg{Type: tea.KeyRight})
	m, _ = pressBatch(t, m, spaceKey)
	m, _ = pressBatch(t, m, tea.KeyMsg{Type: tea.KeyLeft})
	m, _ = pressBatch(t, m, spaceKey)

	// Enter opens the layout dialog; a second Enter accepts it and launches.
	m, _ = pressBatch(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, cmd := pressBatch(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	want := []string{"two", "one"}
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

	m, _ = pressBatch(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if len(m.SelectedBatch) != 0 {
		t.Fatal("enter with zero marks must not launch")
	}
	if m.profile.statusMessage == "" {
		t.Error("empty launch should explain itself")
	}
}

func TestBatchEscCancelsAndClears(t *testing.T) {
	m := batchTestModel()
	m.batch.marked = []string{"one"}

	m, _ = pressBatch(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != gridMode {
		t.Fatalf("esc must return to grid mode, got %v", m.mode)
	}
	if len(m.batch.marked) != 0 {
		t.Error("cancel must clear the marks")
	}
}

// q cancels like esc — the footer advertises both, and q is the cancel key
// everywhere else in the TUI.
func TestBatchQCancelsAndClears(t *testing.T) {
	m := batchTestModel()
	m.batch.marked = []string{"one"}

	m, _ = pressBatch(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if m.mode != gridMode {
		t.Fatalf("q must return to grid mode, got %v", m.mode)
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
	if m.batch.mark("two") == 0 {
		t.Error("space after a quick-nav jump should mark the target cell")
	}
}

func TestBatchViewShowsMarkPositions(t *testing.T) {
	m := batchTestModel()
	m.termWidth, m.termHeight = 100, 40
	// "two" marked first, so it carries ① and "one" carries ②.
	m, _ = pressBatch(t, m, tea.KeyMsg{Type: tea.KeyRight})
	m, _ = pressBatch(t, m, spaceKey)
	m, _ = pressBatch(t, m, tea.KeyMsg{Type: tea.KeyLeft})
	m, _ = pressBatch(t, m, spaceKey)

	out := ansi.Strip(m.View())
	if !strings.Contains(out, "BATCH 2/9") {
		t.Errorf("view should show the batch counter, got:\n%s", out)
	}
	if !strings.Contains(out, "① two") || !strings.Contains(out, "② one") {
		t.Errorf("marks must show their selection position, got:\n%s", out)
	}
}

func TestBatchViewShowsUnmarkedGlyph(t *testing.T) {
	m := batchTestModel()
	m.termWidth, m.termHeight = 100, 40

	out := ansi.Strip(m.View())
	if !strings.Contains(out, "○ one") || !strings.Contains(out, "○ two") {
		t.Errorf("markable cells start unmarked, got:\n%s", out)
	}
}
