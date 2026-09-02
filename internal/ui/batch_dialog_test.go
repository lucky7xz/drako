package ui

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lucky7xz/drako/internal/config"
)

// dialogModel: a 3x3 grid of runnable cells, n of them already marked and the
// layout dialog open on the default distribution.
func dialogModel(t *testing.T, n int) Model {
	t.Helper()
	var cmds []config.Command
	names := []string{"c1", "c2", "c3", "c4", "c5", "c6", "c7", "c8", "c9"}
	for i, name := range names {
		cmds = append(cmds, config.Command{
			Name: name, Command: "true", Row: i / 3, Col: string(rune('a' + i%3)),
		})
	}
	cfg := config.Config{X: 3, Y: 3, Commands: cmds}
	cfg.ApplyDefaults()
	m := Model{
		Config:  cfg,
		styles:  BuildStyles(cfg),
		mode:    batchMode,
		gridNav: gridNav{grid: config.BuildGrid(cfg)},
		batch:   batchState{marked: names[:n]},
	}
	next, _ := m.updateBatchMode(tea.KeyMsg{Type: tea.KeyEnter})
	return next.(Model)
}

// Enter opens the layout dialog rather than launching straight away, so the
// tab split is a decision and not a default you never saw.
func TestDialog_EnterOpensItOnTheDefaultLayout(t *testing.T) {
	m := dialogModel(t, 6)
	if !slices.Equal(m.batch.tabs, []int{4, 2}) {
		t.Fatalf("six cells should open on [4 2], got %v", m.batch.tabs)
	}
	if len(m.SelectedBatch) != 0 {
		t.Error("opening the dialog must not launch yet")
	}
}

// One cell has one possible layout, so there is nothing to ask.
func TestDialog_SkippedWhenThereIsNoChoice(t *testing.T) {
	m := dialogModel(t, 1)
	if m.batch.tabs != nil {
		t.Errorf("a single cell needs no dialog, got %v", m.batch.tabs)
	}
	if !slices.Equal(m.SelectedBatch, []string{"c1"}) {
		t.Errorf("a single cell must launch straight away, got %v", m.SelectedBatch)
	}
}

func pressDialog(t *testing.T, m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.updateBatchMode(msg)
	out, ok := next.(Model)
	if !ok {
		t.Fatalf("updateBatchMode returned %T", next)
	}
	return out, cmd
}

// The tab-count field holds focus first: up asks for another tab and the cells
// redistribute evenly.
func TestDialog_TabCountKnobRedistributes(t *testing.T) {
	m := dialogModel(t, 6)
	m, _ = pressDialog(t, m, tea.KeyMsg{Type: tea.KeyUp})
	if !slices.Equal(m.batch.tabs, []int{2, 2, 2}) {
		t.Fatalf("up on the tab count should give [2 2 2], got %v", m.batch.tabs)
	}
	m, _ = pressDialog(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if !slices.Equal(m.batch.tabs, []int{4, 2}) {
		t.Fatalf("down should return to [4 2], got %v", m.batch.tabs)
	}
}

func TestDialog_TabCountStopsAtTheLimits(t *testing.T) {
	m := dialogModel(t, 6)
	for range 10 {
		m, _ = pressDialog(t, m, tea.KeyMsg{Type: tea.KeyDown})
	}
	if !slices.Equal(m.batch.tabs, []int{4, 2}) {
		t.Errorf("six cells cannot use fewer than two tabs, got %v", m.batch.tabs)
	}
	for range 10 {
		m, _ = pressDialog(t, m, tea.KeyMsg{Type: tea.KeyUp})
	}
	if !slices.Equal(m.batch.tabs, []int{1, 1, 1, 1, 1, 1}) {
		t.Errorf("six cells cannot use more than six tabs, got %v", m.batch.tabs)
	}
}

// Right moves onto the tab boxes, where up/down move a single pane.
func TestDialog_PaneKnobShiftsOnePane(t *testing.T) {
	m := dialogModel(t, 6)
	m, _ = pressDialog(t, m, tea.KeyMsg{Type: tea.KeyRight})
	if m.batch.focus != 1 {
		t.Fatalf("right should focus the first tab box, got %d", m.batch.focus)
	}
	m, _ = pressDialog(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if !slices.Equal(m.batch.tabs, []int{3, 3}) {
		t.Fatalf("down on tab 1 should give [3 3], got %v", m.batch.tabs)
	}
	if len(m.batch.tabs) != 2 {
		t.Error("a pane shift must not change the tab count")
	}
}

// A shift with nowhere to go is refused and says so, rather than silently
// doing nothing.
func TestDialog_RefusedShiftExplainsItself(t *testing.T) {
	m := dialogModel(t, 8) // [4 4] — rigid
	m, _ = pressDialog(t, m, tea.KeyMsg{Type: tea.KeyRight})
	m, _ = pressDialog(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if !slices.Equal(m.batch.tabs, []int{4, 4}) {
		t.Fatalf("[4 4] has nowhere to move a pane, got %v", m.batch.tabs)
	}
	if m.profile.statusMessage == "" {
		t.Error("a refused shift should explain itself in the status line")
	}
}

func TestDialog_FocusStaysInRange(t *testing.T) {
	m := dialogModel(t, 6) // two tabs: focus 0..2
	for range 6 {
		m, _ = pressDialog(t, m, tea.KeyMsg{Type: tea.KeyRight})
	}
	if m.batch.focus != 2 {
		t.Errorf("focus should stop at the last tab box, got %d", m.batch.focus)
	}
	for range 6 {
		m, _ = pressDialog(t, m, tea.KeyMsg{Type: tea.KeyLeft})
	}
	if m.batch.focus != 0 {
		t.Errorf("focus should stop at the tab count field, got %d", m.batch.focus)
	}
}

// Shrinking the tab count must not leave focus pointing past the last box.
func TestDialog_FocusFollowsAShrinkingLayout(t *testing.T) {
	m := dialogModel(t, 6)
	m, _ = pressDialog(t, m, tea.KeyMsg{Type: tea.KeyUp}) // [2 2 2]
	for range 5 {
		m, _ = pressDialog(t, m, tea.KeyMsg{Type: tea.KeyRight})
	}
	if m.batch.focus != 3 {
		t.Fatalf("focus should be on the third tab box, got %d", m.batch.focus)
	}
	m, _ = pressDialog(t, m, tea.KeyMsg{Type: tea.KeyLeft}) // back to the count
	m, _ = pressDialog(t, m, tea.KeyMsg{Type: tea.KeyLeft})
	m, _ = pressDialog(t, m, tea.KeyMsg{Type: tea.KeyLeft})
	m, _ = pressDialog(t, m, tea.KeyMsg{Type: tea.KeyDown}) // [4 2] — one box fewer
	if m.batch.focus > len(m.batch.tabs) {
		t.Errorf("focus %d points past %v", m.batch.focus, m.batch.tabs)
	}
}

func TestDialog_EnterLaunchesWithTheChosenLayout(t *testing.T) {
	m := dialogModel(t, 6)
	m, _ = pressDialog(t, m, tea.KeyMsg{Type: tea.KeyUp}) // [2 2 2]

	m, cmd := pressDialog(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !slices.Equal(m.SelectedBatch, []string{"c1", "c2", "c3", "c4", "c5", "c6"}) {
		t.Fatalf("launch must carry the marked cells, got %v", m.SelectedBatch)
	}
	if !slices.Equal(m.SelectedTabs, []int{2, 2, 2}) {
		t.Fatalf("launch must carry the chosen layout, got %v", m.SelectedTabs)
	}
	if cmd == nil {
		t.Fatal("launch must quit the TUI")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("cmd = %T, want QuitMsg", cmd())
	}
}

// Esc backs out of the dialog to marking, marks intact — it is a step in the
// launch, not a separate mode you fall out of.
func TestDialog_EscReturnsToMarking(t *testing.T) {
	m := dialogModel(t, 6)
	m, _ = pressDialog(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.batch.tabs != nil {
		t.Error("esc must close the dialog")
	}
	if len(m.batch.marked) != 6 {
		t.Errorf("esc must keep the marks, got %v", m.batch.marked)
	}
	if m.mode != batchMode {
		t.Errorf("esc returns to marking, not out of batch mode (mode %v)", m.mode)
	}
}

func TestDialog_RowShowsTheLayoutAndTheFocus(t *testing.T) {
	m := dialogModel(t, 6)
	m.termWidth, m.termHeight = 100, 40

	out := ansi.Strip(m.View())
	if !strings.Contains(out, "tabs‹2›") {
		t.Errorf("the tab count holds focus first:\n%s", out)
	}
	if !strings.Contains(out, "T1[4] T2[2]") {
		t.Errorf("every tab shows its pane count:\n%s", out)
	}

	m, _ = pressDialog(t, m, tea.KeyMsg{Type: tea.KeyRight})
	out = ansi.Strip(m.View())
	if !strings.Contains(out, "tabs[2]") || !strings.Contains(out, "T1‹4›") {
		t.Errorf("focus must move to the first tab box:\n%s", out)
	}
}
