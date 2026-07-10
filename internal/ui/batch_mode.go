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
	marked map[string]bool // cell name → marked
}

// enterBatchMode gates and switches into batch mode with an empty mark set.
// Glassroot never offers batch: a batch hands the guest a full tmux session
// (new windows, shells) — far outside the kiosk contract.
func (m Model) enterBatchMode() (tea.Model, tea.Cmd) {
	if m.GlassrootMode {
		return m, m.setProfileStatus("Batch unavailable here", false)
	}
	if _, err := tmuxLookPath("tmux"); err != nil {
		return m, m.setProfileStatus("Batch needs tmux installed", false)
	}
	m.batch = batchState{marked: map[string]bool{}}
	m.mode = batchMode
	return m, nil
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
	case key == "esc":
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
