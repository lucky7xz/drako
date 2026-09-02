package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lucky7xz/drako/internal/config"
)

// dropdownTestModel: one folder cell open in dropdown mode — two runnable
// items and one without a command (platform-empty).
func dropdownTestModel() Model {
	cfg := config.Config{X: 1, Y: 1, Commands: []config.Command{
		{Name: "folder", Row: 0, Col: "a", Items: []config.CommandItem{
			{Name: "first", Command: "echo 1"},
			{Name: "second", Command: "echo 2"},
			{Name: "bare"},
		}},
	}}
	cfg.ApplyDefaults()
	return Model{
		Config:   cfg,
		styles:   BuildStyles(cfg),
		mode:     dropdownMode,
		gridNav:  gridNav{grid: config.BuildGrid(cfg)},
		dropdown: dropdownState{items: cfg.Commands[0].Items},
	}
}

func pressDropdown(t *testing.T, m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.updateDropdownMode(msg)
	out, ok := next.(Model)
	if !ok {
		t.Fatalf("updateDropdownMode returned %T", next)
	}
	return out, cmd
}

func TestDropdownItemsAlwaysNumbered(t *testing.T) {
	out := ansi.Strip(dropdownTestModel().renderDropdownPopup())
	for _, want := range []string{"1 first", "2 second", "3 bare"} {
		if !strings.Contains(out, want) {
			t.Errorf("popup missing numbered item %q:\n%s", want, out)
		}
	}
}

func TestDropdownLeaderEntersMarking(t *testing.T) {
	withTmux(t, true)
	m := dropdownTestModel()

	m, _ = pressDropdown(t, m, keyRunes("m"))
	if !m.leader.pending {
		t.Fatal("leader must arm inside a dropdown")
	}
	m, _ = pressDropdown(t, m, keyRunes("b"))
	if !m.batch.dropdown {
		t.Fatal("m,b inside a dropdown must start item marking")
	}
	if m.mode != dropdownMode {
		t.Fatal("marking happens inside the dropdown, not a mode switch")
	}
}

func TestDropdownLeaderDigitSwallowed(t *testing.T) {
	m := dropdownTestModel()
	m.dropdown.selectedIdx = 0

	m, _ = pressDropdown(t, m, keyRunes("m"))
	m, _ = pressDropdown(t, m, keyRunes("3"))
	if m.dropdown.selectedIdx != 0 {
		t.Error("a pending-leader digit must be swallowed, not jump the selector")
	}
	if m.leader.pending {
		t.Error("sequence must disarm")
	}
}

func TestDropdownLeaderGlassrootRefused(t *testing.T) {
	withTmux(t, true)
	m := dropdownTestModel()
	m.GlassrootMode = true

	m, _ = pressDropdown(t, m, keyRunes("m"))
	m, _ = pressDropdown(t, m, keyRunes("b"))
	if m.batch.dropdown {
		t.Fatal("glassroot must refuse dropdown batching")
	}
}

func markingModel(t *testing.T) Model {
	t.Helper()
	withTmux(t, true)
	m := dropdownTestModel()
	m, _ = pressDropdown(t, m, keyRunes("m"))
	m, _ = pressDropdown(t, m, keyRunes("b"))
	return m
}

func TestDropdownMarkToggle(t *testing.T) {
	m := markingModel(t)

	m, _ = pressDropdown(t, m, spaceKey)
	if m.batch.mark("first") != 1 {
		t.Fatal("space must mark the selected item")
	}
	m, _ = pressDropdown(t, m, spaceKey)
	if m.batch.mark("first") != 0 {
		t.Fatal("space again must unmark")
	}
}

func TestDropdownBareItemNotMarkable(t *testing.T) {
	m := markingModel(t)
	m.dropdown.selectedIdx = 2 // "bare"

	m, _ = pressDropdown(t, m, spaceKey)
	if m.batch.mark("bare") != 0 {
		t.Fatal("items without a command must not be markable")
	}
}

func TestDropdownDigitsStillJumpWhileMarking(t *testing.T) {
	m := markingModel(t)

	m, _ = pressDropdown(t, m, keyRunes("2"))
	if m.dropdown.selectedIdx != 1 {
		t.Errorf("digits must keep jumping while marking, idx = %d", m.dropdown.selectedIdx)
	}
}

func TestDropdownLaunchCollectsSelectionOrder(t *testing.T) {
	m := markingModel(t)
	// Mark "second" before "first"; the launch follows the marking.
	m, _ = pressDropdown(t, m, keyRunes("2"))
	m, _ = pressDropdown(t, m, spaceKey)
	m, _ = pressDropdown(t, m, keyRunes("1"))
	m, _ = pressDropdown(t, m, spaceKey)

	// Enter opens the layout dialog; a second Enter accepts it and launches.
	m, _ = pressDropdown(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, cmd := pressDropdown(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	want := []string{"second", "first"}
	if len(m.SelectedBatch) != 2 || m.SelectedBatch[0] != want[0] || m.SelectedBatch[1] != want[1] {
		t.Fatalf("SelectedBatch = %v, want %v (selection order)", m.SelectedBatch, want)
	}
	if cmd == nil {
		t.Fatal("launch must quit the TUI")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("cmd = %T, want QuitMsg", cmd())
	}
	if m.Quitting {
		t.Error("Quitting must stay false for a relaunch")
	}
}

func TestDropdownLaunchWithNothingMarkedDoesNotRunItem(t *testing.T) {
	m := markingModel(t)

	m, _ = pressDropdown(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Selected != "" || len(m.SelectedBatch) != 0 {
		t.Fatal("enter with zero marks must neither launch nor run the selected item")
	}
	if m.profile.statusMessage == "" {
		t.Error("empty launch should explain itself")
	}
}

func TestDropdownEscExitsMarkingKeepsDropdown(t *testing.T) {
	m := markingModel(t)
	m.batch.marked = []string{"first"}

	m, _ = pressDropdown(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.batch.dropdown || len(m.batch.marked) != 0 {
		t.Fatal("esc must end marking and clear marks")
	}
	if m.mode != dropdownMode || len(m.dropdown.items) == 0 {
		t.Fatal("first esc keeps the dropdown open")
	}

	m, _ = pressDropdown(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != gridMode {
		t.Fatal("second esc closes the dropdown as before")
	}
}

func TestDropdownPopupShowsMarksWhileMarking(t *testing.T) {
	m := markingModel(t)
	// "second" marked first, so the item numbering (1 first, 2 second) and
	// the mark positions (② first, ① second) stay tellable apart.
	m, _ = pressDropdown(t, m, keyRunes("2"))
	m, _ = pressDropdown(t, m, spaceKey)
	m, _ = pressDropdown(t, m, keyRunes("1"))
	m, _ = pressDropdown(t, m, spaceKey)

	out := ansi.Strip(m.renderDropdownPopup())
	if !strings.Contains(out, "1 ② first") || !strings.Contains(out, "2 ① second") {
		t.Errorf("marking popup must show mark positions beside item numbers:\n%s", out)
	}
	if !strings.Contains(out, "3 bare") {
		t.Errorf("an unbatchable item gets no mark chrome at all:\n%s", out)
	}
	if !strings.Contains(out, "[ BATCH 2/9 ]") {
		t.Errorf("marking popup must show the prominent batch counter:\n%s", out)
	}

	plain := ansi.Strip(dropdownTestModel().renderDropdownPopup())
	if strings.Contains(plain, "①") || strings.Contains(plain, "○") || strings.Contains(plain, "/9") {
		t.Errorf("plain dropdown must not show batch chrome:\n%s", plain)
	}
}
