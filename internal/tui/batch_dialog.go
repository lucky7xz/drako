package tui

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/multiplex"
	"github.com/lucky7xz/drako/internal/settings"
)

// The layout dialog: after Enter, before launch. It edits one value — how the
// marked cells split into tabs — with two knobs. The tab count re-runs the
// distribution; a tab's pane count moves a single pane between tabs.

// beginLaunch opens the dialog, or launches a single cell straight away: it has
// only one layout.
func (m Model) beginLaunch() (tea.Model, tea.Cmd) {
	n := len(m.batch.marked)
	if n == 0 {
		return m, m.setProfileStatus("Nothing marked", false)
	}
	if n == 1 {
		return m.launchBatch()
	}
	m.batch.tabs = multiplex.Distribute(n, multiplex.MinTabs(n))
	m.batch.focus = 0
	return m, nil
}

// launchBatch hands the marked cells and their layout to the app and quits;
// the TUI relaunches once the batch is done.
func (m Model) launchBatch() (tea.Model, tea.Cmd) {
	m.selectedBatch = slices.Clone(m.batch.marked)
	m.selectedTabs = slices.Clone(m.batch.tabs)
	return m, tea.Quit
}

// updateBatchDialog consumes a key while the layout dialog is open.
func (m Model) updateBatchDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Config.Screen(settings.BatchLayout).Act(msg.String()) {
	case settings.Back:
		m.batch.tabs, m.batch.focus = nil, 0
		return m, nil

	case settings.Launch:
		return m.launchBatch()

	// The fields sit in a row, so left/right picks one and up/down changes
	// the value it holds.
	case settings.MoveH:
		if IsLeft(m.Config.Keys, msg) {
			m.batch.focus = max(m.batch.focus-1, 0)
		} else {
			m.batch.focus = min(m.batch.focus+1, len(m.batch.tabs))
		}

	case settings.MoveV:
		if IsUp(m.Config.Keys, msg) {
			return m.adjustLayout(+1)
		}
		return m.adjustLayout(-1)
	}
	return m, nil
}

// adjustLayout applies delta to the focused field: the tab count redistributes
// every cell, a tab's pane count moves one pane and leaves the count alone.
func (m Model) adjustLayout(delta int) (tea.Model, tea.Cmd) {
	n := len(m.batch.marked)
	if m.batch.focus == 0 {
		k := min(max(len(m.batch.tabs)+delta, multiplex.MinTabs(n)), n)
		m.batch.tabs = multiplex.Distribute(n, k)
		return m, nil
	}
	tabs, ok := multiplex.Shift(m.batch.tabs, m.batch.focus-1, delta)
	if !ok {
		return m, m.setProfileStatus("No room to move that pane", false)
	}
	m.batch.tabs = tabs
	return m, nil
}
