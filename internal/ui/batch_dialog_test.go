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

func pressDialog(t *testing.T, m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.updateBatchMode(msg)
	out, ok := next.(Model)
	if !ok {
		t.Fatalf("updateBatchMode returned %T", next)
	}
	return out, cmd
}

var (
	keyUp    = tea.KeyMsg{Type: tea.KeyUp}
	keyDown  = tea.KeyMsg{Type: tea.KeyDown}
	keyLeft  = tea.KeyMsg{Type: tea.KeyLeft}
	keyRight = tea.KeyMsg{Type: tea.KeyRight}
	keyEnter = tea.KeyMsg{Type: tea.KeyEnter}
)

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

// The tab-count row holds focus first; right asks for another tab and the
// cells redistribute evenly.
func TestDialog_TabCountKnobRedistributes(t *testing.T) {
	m := dialogModel(t, 6)
	m, _ = pressDialog(t, m, keyUp)
	if !slices.Equal(m.batch.tabs, []int{2, 2, 2}) {
		t.Fatalf("up on the tab count should give [2 2 2], got %v", m.batch.tabs)
	}
	m, _ = pressDialog(t, m, keyDown)
	if !slices.Equal(m.batch.tabs, []int{4, 2}) {
		t.Fatalf("down should return to [4 2], got %v", m.batch.tabs)
	}
}

func TestDialog_TabCountStopsAtTheLimits(t *testing.T) {
	m := dialogModel(t, 6)
	for range 10 {
		m, _ = pressDialog(t, m, keyDown)
	}
	if !slices.Equal(m.batch.tabs, []int{4, 2}) {
		t.Errorf("six cells cannot use fewer than two tabs, got %v", m.batch.tabs)
	}
	for range 10 {
		m, _ = pressDialog(t, m, keyUp)
	}
	if !slices.Equal(m.batch.tabs, []int{1, 1, 1, 1, 1, 1}) {
		t.Errorf("six cells cannot use more than six tabs, got %v", m.batch.tabs)
	}
}

// Right moves onto a tab's box, where up/down move a single pane.
func TestDialog_PaneKnobShiftsOnePane(t *testing.T) {
	m := dialogModel(t, 6)
	m, _ = pressDialog(t, m, keyRight)
	if m.batch.focus != 1 {
		t.Fatalf("right should focus the first tab, got %d", m.batch.focus)
	}
	m, _ = pressDialog(t, m, keyDown)
	if !slices.Equal(m.batch.tabs, []int{3, 3}) {
		t.Fatalf("left on tab 1 should give [3 3], got %v", m.batch.tabs)
	}
	if len(m.batch.tabs) != 2 {
		t.Error("a pane shift must not change the tab count")
	}
}

// A shift with nowhere to go is refused and says so, rather than silently
// doing nothing.
func TestDialog_RefusedShiftExplainsItself(t *testing.T) {
	m := dialogModel(t, 8) // [4 4] — rigid
	m, _ = pressDialog(t, m, keyRight)
	m, _ = pressDialog(t, m, keyDown)
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
		m, _ = pressDialog(t, m, keyRight)
	}
	if m.batch.focus != 2 {
		t.Errorf("focus should stop at the last tab, got %d", m.batch.focus)
	}
	for range 6 {
		m, _ = pressDialog(t, m, keyLeft)
	}
	if m.batch.focus != 0 {
		t.Errorf("focus should stop at the tab count, got %d", m.batch.focus)
	}
}

// Shrinking the tab count must not leave focus pointing past the last row.
func TestDialog_FocusFollowsAShrinkingLayout(t *testing.T) {
	m := dialogModel(t, 6)
	m, _ = pressDialog(t, m, keyUp) // [2 2 2]
	for range 5 {
		m, _ = pressDialog(t, m, keyRight)
	}
	if m.batch.focus != 3 {
		t.Fatalf("focus should be on the third tab, got %d", m.batch.focus)
	}
	for range 5 {
		m, _ = pressDialog(t, m, keyLeft) // back to the count
	}
	m, _ = pressDialog(t, m, keyDown) // [4 2] — one box fewer
	if m.batch.focus > len(m.batch.tabs) {
		t.Errorf("focus %d points past %v", m.batch.focus, m.batch.tabs)
	}
}

func TestDialog_EnterLaunchesWithTheChosenLayout(t *testing.T) {
	m := dialogModel(t, 6)
	m, _ = pressDialog(t, m, keyUp) // [2 2 2]

	m, cmd := pressDialog(t, m, keyEnter)
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

// The fields are the tab count and one box per tab, each holding the marks it
// will run — grouping checked here rather than through the rendered padding.
func TestDialog_FieldsGroupTheMarks(t *testing.T) {
	m := dialogModel(t, 6)
	labels, bodies := m.layoutFields()
	if !slices.Equal(labels, []string{"tabs", "T1", "T2"}) {
		t.Errorf("labels = %v, want the count then one per tab", labels)
	}
	if !slices.Equal(bodies, []string{"2", "① ② ③ ④", "⑤ ⑥"}) {
		t.Errorf("bodies = %v, want the marks split [4 2]", bodies)
	}

	m, _ = pressDialog(t, m, keyUp) // [2 2 2]
	labels, bodies = m.layoutFields()
	if !slices.Equal(labels, []string{"tabs", "T1", "T2", "T3"}) {
		t.Errorf("a third tab must add a field, got %v", labels)
	}
	if !slices.Equal(bodies, []string{"3", "① ②", "③ ④", "⑤ ⑥"}) {
		t.Errorf("bodies = %v, want the marks regrouped [2 2 2]", bodies)
	}
}

// Every field is drawn with the grid's own cell chrome, so the dialog reads as
// drako rather than as a form.
func TestDialog_OverlayDrawsFieldsAsGridCells(t *testing.T) {
	m := dialogModel(t, 6)
	m.termWidth, m.termHeight = 120, 40

	out := ansi.Strip(m.View())
	for _, want := range []string{"[tabs]", "[T1]", "[T2]", "┍━", "┕━"} {
		if !strings.Contains(out, want) {
			t.Errorf("overlay missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "[T3]") {
		t.Errorf("two tabs must draw two boxes:\n%s", out)
	}
}

// The focused field's label rule turns double, so focus reads without colour
// — and it follows left/right rather than up/down.
func TestDialog_OverlayMarksTheFocusedField(t *testing.T) {
	m := dialogModel(t, 6)
	m.termWidth, m.termHeight = 120, 40

	if out := ansi.Strip(m.View()); !strings.Contains(out, "═[tabs]═") {
		t.Errorf("the tab count holds focus first:\n%s", out)
	}
	m, _ = pressDialog(t, m, keyRight)
	out := ansi.Strip(m.View())
	if !strings.Contains(out, "═[T1]═") {
		t.Errorf("right must move focus to the first tab:\n%s", out)
	}
	if !strings.Contains(out, "─[tabs]─") {
		t.Errorf("the tab count must lose focus:\n%s", out)
	}
}

// A box never resizes while the knobs turn, so the row stays put under the
// cursor instead of jumping on every keypress.
func TestDialog_BoxWidthDoesNotMoveWithTheLayout(t *testing.T) {
	m := dialogModel(t, 6)
	want := m.layoutFieldWidth()
	for range 5 {
		m, _ = pressDialog(t, m, keyUp)
		if got := m.layoutFieldWidth(); got != want {
			t.Fatalf("box width changed from %d to %d at %v", want, got, m.batch.tabs)
		}
	}
}
