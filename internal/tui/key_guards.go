package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/settings"
)

// handleDeckKeys applies the keys that act on the deck in use: pinning it and
// switching to another. The key table offers them only on the grid; from the
// inventory or path mode a pin would land on something other than what is
// under the cursor. Returns handled=false to let the caller go on to the screen's own
// handler.
func (m Model) handleDeckKeys(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	switch m.act(msg) {
	case settings.PinDeck:
		cmd := m.toggleProfileLock()
		return m, cmd, true
	case settings.SwitchDeck:
		if updated, ok := m.switchToProfileIndex(digit(msg)); ok {
			return updated, nil, true
		}
		return m, nil, true
	case settings.PrevDeck:
		next, cmd := m.handleProfileCycle(-1)
		return next.(Model), cmd, true
	case settings.NextDeck:
		next, cmd := m.handleProfileCycle(1)
		return next.(Model), cmd, true
	}
	return m, nil, false
}
