package ui

import (
	"slices"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/core"
	"github.com/lucky7xz/drako/internal/multiplex"
)

// Batch mode: mark several cells, launch them together in a multiplexer. All state
// lives in batchState and all behavior in this file + batch_view.go — the
// feature is subtractable (spec: docs/tutorial/2026-06-23-batch-launch-design.md).

// resolveBackend is the capability gate seam: batch mode only exists when a
// multiplexer drako can reach is available.
var resolveBackend = multiplex.Resolve

type batchState struct {
	// marked holds the marked cells or folder items in marking order — the
	// launch order, which decides how cells group into tabs.
	marked []core.Selection
	// dropdown scopes the mark set to the open folder's items: an inside
	// batch never mixes with grid cells, and vice versa.
	dropdown bool

	// tabs is the layout being chosen: nil while marking, set once the
	// dialog opens. focus is 0 for the tab-count field, 1..len(tabs) for a
	// tab's pane count.
	tabs  []int
	focus int
}

// choosing reports whether the layout dialog is open.
func (b batchState) choosing() bool { return b.tabs != nil }

// mark is sel's 1-based position in the mark set, or 0 when unmarked.
func (b batchState) mark(sel core.Selection) int {
	return slices.Index(b.marked, sel) + 1
}

// toggle adds sel to the mark set or drops it, keeping the rest in order. It
// builds a new slice, so a copied Model never shares its marks.
func (b *batchState) toggle(sel core.Selection) {
	if i := slices.Index(b.marked, sel); i >= 0 {
		b.marked = slices.Delete(slices.Clone(b.marked), i, i+1)
		return
	}
	b.marked = append(slices.Clone(b.marked), sel)
}

// batchGate is the single availability check for every batch entry point.
// Glassroot never offers batch: a batch hands the guest a full tmux session
// (new windows, shells) — far outside the kiosk contract.
func (m *Model) batchGate() (bool, tea.Cmd) {
	if m.GlassrootMode {
		return false, m.setProfileStatus("Batch unavailable here", false)
	}
	if _, err := resolveBackend(m.Config.BatchForceTmux, multiplex.OSEnv()); err != nil {
		return false, m.setProfileStatus(err.Error(), false)
	}
	return true, nil
}

// enterBatchMode gates and switches into batch mode with an empty mark set.
func (m Model) enterBatchMode() (tea.Model, tea.Cmd) {
	if ok, cmd := m.batchGate(); !ok {
		return m, cmd
	}
	m.batch = batchState{}
	m.mode = batchMode
	return m, nil
}

// enterDropdownBatch starts marking the open folder's items; the mode stays
// dropdownMode — marking is a property of the mark set, not a new mode.
func (m Model) enterDropdownBatch() (tea.Model, tea.Cmd) {
	if ok, cmd := m.batchGate(); !ok {
		return m, cmd
	}
	m.batch = batchState{dropdown: true}
	return m, nil
}

// updateDropdownMarking consumes a key while a dropdown batch is active.
// handled=false hands the key back for the dropdown's normal behavior
// (navigation, digit jumps, explain).
func (m Model) updateDropdownMarking(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if IsCancel(m.Config.Keys, msg) {
		// End marking, keep the dropdown open; the next esc/q closes it.
		m.batch = batchState{}
		return m, nil, true
	}

	switch msg.String() {
	case " ":
		if m.dropdown.selectedIdx < 0 || m.dropdown.selectedIdx >= len(m.dropdown.items) {
			return m, nil, true
		}
		if m.dropdown.items[m.dropdown.selectedIdx].Command == "" {
			return m, m.setProfileStatus("Item not batchable", false), true
		}
		// By position: two items in a folder may share a name.
		sel := core.Selection{Cell: m.dropdownCell(), Item: m.dropdown.selectedIdx + 1}
		if m.batch.mark(sel) == 0 && len(m.batch.marked) >= multiplex.MaxCommands {
			return m, m.setProfileStatus("Batch is full (9 max)", false), true
		}
		m.batch.toggle(sel)
		return m, nil, true

	case "enter":
		next, cmd := m.beginLaunch()
		return next, cmd, true
	}
	return m, nil, false
}

// markable reports whether a grid cell can join a batch: it must resolve to a
// direct, non-empty command. Folder (dropdown) cells and platform-empty cells
// stay out in v1.
func (m Model) markable(name string) bool {
	cell, _, found := core.Selection{Cell: name}.Resolve(m.Config)
	return found && cell.Command != "" && len(cell.Items) == 0
}

// updateBatchMode: navigation as usual; Space toggles the cursor cell's mark,
// Enter launches everything marked, Esc cancels. Marks live only in
// batchState — the grid itself never learns about them.
func (m Model) updateBatchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// The layout dialog owns every key once it is open.
	if m.batch.choosing() {
		return m.updateBatchDialog(msg)
	}

	// Quick navigation works exactly like grid mode: digits jump the cursor.
	if num, err := strconv.Atoi(key); err == nil && num >= 1 && num <= 9 {
		return m.quickNav(num - 1)
	}
	// Any non-numeric key ends a quick-nav sequence in progress.
	m.gridNav.cancel()

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
		sel := core.Selection{Cell: name}
		if m.batch.mark(sel) == 0 && len(m.batch.marked) >= multiplex.MaxCommands {
			return m, m.setProfileStatus("Batch is full (9 max)", false)
		}
		m.batch.toggle(sel)
		return m, nil

	case key == "enter":
		return m.beginLaunch()

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
