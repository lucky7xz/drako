package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleCrossModeBindings applies the key bindings that reach across every
// mode from the grid: the profile lock and profile switch/cycle keys. They
// only fire outside a text field (so a typed character is never stolen)
// and only from the grid/child mode they act on. Returns handled=false to
// let the caller fall through to the normal mode dispatch.
func (m Model) handleCrossModeBindings(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if m.capturingText() {
		return m, nil, false
	}
	// Both the lock and profile switching act on the *active* profile, so
	// they belong to the grid. Offering the lock from the inventory would
	// silently pin something other than the item under the cursor.
	if m.mode != gridMode && m.mode != childMode {
		return m, nil, false
	}

	if IsLock(m.Config.Keys, msg) {
		cmd := m.toggleProfileLock()
		return m, cmd, true
	}

	// Profile switching with configurable modifier + Number or ~ (Shift + `)
	if ok, target := IsProfileSwitch(m.Config.Keys, msg, m.Config.NumbModifier); ok {
		if target < len(m.profile.profiles) {
			if updated, ok := m.switchToProfileIndex(target); ok {
				return updated, nil, true
			}
		}
		return m, nil, true
	}
	if IsProfilePrev(m.Config.Keys, msg) {
		next, cmd := m.handleProfileCycle(-1)
		return next.(Model), cmd, true
	}
	if IsProfileNext(m.Config.Keys, msg) {
		next, cmd := m.handleProfileCycle(1)
		return next.(Model), cmd, true
	}
	return m, nil, false
}
