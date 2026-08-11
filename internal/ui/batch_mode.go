package ui

import (
	"os/exec"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/core"
	"github.com/lucky7xz/drako/internal/multiplex"
)

// Batch mode: mark several cells, launch them together in tmux. All state
// lives in batchState and all behavior in this file + batch_view.go — the
// feature is subtractable (spec: docs/tutorial/2026-06-23-batch-launch-design.md).

// tmuxLookPath is the capability gate seam; batch mode only exists when tmux
// is installed.
var tmuxLookPath = exec.LookPath

type batchState struct {
	marked map[string]bool // cell/item name → marked
	// dropdown scopes the mark set to the open folder's items: an inside
	// batch never mixes with grid cells, and vice versa.
	dropdown bool
}

// batchGate is the single availability check for every batch entry point.
// Glassroot never offers batch: a batch hands the guest a full tmux session
// (new windows, shells) — far outside the kiosk contract.
func (m *Model) batchGate() (bool, tea.Cmd) {
	if m.GlassrootMode {
		return false, m.setProfileStatus("Batch unavailable here", false)
	}
	if _, err := tmuxLookPath("tmux"); err != nil {
		return false, m.setProfileStatus("Batch needs tmux installed", false)
	}
	return true, nil
}

// enterBatchMode gates and switches into batch mode with an empty mark set.
func (m Model) enterBatchMode() (tea.Model, tea.Cmd) {
	if ok, cmd := m.batchGate(); !ok {
		return m, cmd
	}
	m.batch = batchState{marked: map[string]bool{}}
	m.mode = batchMode
	return m, nil
}

// enterDropdownBatch starts marking the open folder's items; the mode stays
// dropdownMode — marking is a property of the mark set, not a new mode.
func (m Model) enterDropdownBatch() (tea.Model, tea.Cmd) {
	if ok, cmd := m.batchGate(); !ok {
		return m, cmd
	}
	m.batch = batchState{marked: map[string]bool{}, dropdown: true}
	return m, nil
}

// updateDropdownMarking consumes a key while a dropdown batch is active.
// handled=false hands the key back for the dropdown's normal behavior
// (navigation, digit jumps, explain).
func (m Model) updateDropdownMarking(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case " ":
		if m.dropdown.selectedIdx < 0 || m.dropdown.selectedIdx >= len(m.dropdown.items) {
			return m, nil, true
		}
		item := m.dropdown.items[m.dropdown.selectedIdx]
		if item.Command == "" {
			return m, m.setProfileStatus("Item not batchable", false), true
		}
		if m.batch.marked[item.Name] {
			delete(m.batch.marked, item.Name)
			return m, nil, true
		}
		if len(m.batch.marked) >= multiplex.MaxCommands {
			return m, m.setProfileStatus("Batch is full (9 max)", false), true
		}
		m.batch.marked[item.Name] = true
		return m, nil, true

	case "enter":
		var names []string
		for _, item := range m.dropdown.items {
			if m.batch.marked[item.Name] {
				names = append(names, item.Name)
			}
		}
		if len(names) == 0 {
			return m, m.setProfileStatus("Nothing marked", false), true
		}
		m.SelectedBatch = names
		return m, tea.Quit, true

	case "esc":
		// End marking, keep the dropdown open; the next esc closes it.
		m.batch = batchState{}
		return m, nil, true
	}
	return m, nil, false
}

// markable reports whether a grid cell can join a batch: it must resolve to a
// direct, non-empty command. Folder (dropdown) cells and platform-empty cells
// stay out in v1.
func (m Model) markable(name string) bool {
	parent, item, found := core.FindCommandByName(m.Config, name)
	return found && item == nil && parent.Command != ""
}

// updateBatchMode: navigation as usual; Space toggles the cursor cell's mark,
// Enter launches everything marked, Esc cancels. Marks live only in
// batchState — the grid itself never learns about them.
func (m Model) updateBatchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Quick navigation works exactly like grid mode: digits jump the cursor.
	if num, err := strconv.Atoi(key); err == nil && num >= 1 && num <= 9 {
		return m.quickNav(num - 1)
	}
	// Any non-numeric key ends a quick-nav sequence in progress.
	if m.gridNav.timer != nil {
		m.gridNav.timer.Stop()
		m.gridNav.timer = nil
	}

	switch {
	case IsCancel(m.Config.Keys, msg):
		m.batch = batchState{}
		m.mode = gridMode
		return m, nil

	case key == " ":
		name := m.currentCellName()
		if name == "" || !m.markable(name) {
			return m, m.setProfileStatus("Cell not batchable", false)
		}
		if m.batch.marked[name] {
			delete(m.batch.marked, name)
			return m, nil
		}
		if len(m.batch.marked) >= multiplex.MaxCommands {
			return m, m.setProfileStatus("Batch is full (9 max)", false)
		}
		m.batch.marked[name] = true
		return m, nil

	case key == "enter":
		names := m.markedInGridOrder()
		if len(names) == 0 {
			return m, m.setProfileStatus("Nothing marked", false)
		}
		m.SelectedBatch = names
		return m, tea.Quit

	case IsUp(m.Config.Keys, msg):
		m.gridNav.moveCursor(-1, 0)
	case IsDown(m.Config.Keys, msg):
		m.gridNav.moveCursor(1, 0)
	case IsLeft(m.Config.Keys, msg):
		m.gridNav.moveCursor(0, -1)
	case IsRight(m.Config.Keys, msg):
		m.gridNav.moveCursor(0, 1)
	}
	return m, nil
}

// currentCellName is the grid content under the cursor ("" for empty cells).
func (m Model) currentCellName() string {
	g := m.gridNav.grid
	if m.gridNav.cursorRow >= len(g) || m.gridNav.cursorCol >= len(g[m.gridNav.cursorRow]) {
		return ""
	}
	return g[m.gridNav.cursorRow][m.gridNav.cursorCol]
}

// markedInGridOrder collects the marked cells in row-major grid order, so the
// launch layout matches what the user sees, not the order they marked in.
func (m Model) markedInGridOrder() []string {
	var names []string
	for _, row := range m.gridNav.grid {
		for _, name := range row {
			if name != "" && m.batch.marked[name] {
				names = append(names, name)
			}
		}
	}
	return names
}
