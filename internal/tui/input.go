package tui

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/settings"
)

// Which key does what is the key table's to say (settings/keymap.go): a
// handler asks act and switches on the answer. What is left here is telling
// the directions of a move apart.

// act is what this key does on the screen the model is showing.
func (m Model) act(msg tea.KeyMsg) settings.Action {
	return m.Config.Screen(m.keyScreen()).Act(msg.String())
}

// keyScreen is the key table's name for the state the model is in.
func (m Model) keyScreen() settings.ScreenID {
	switch m.mode {
	case gridMode:
		if m.leader.pending {
			return settings.GridLeader
		}
		return settings.Grid
	case dropdownMode:
		switch {
		case m.batch.choosing():
			return settings.BatchLayout
		case m.leader.pending:
			return settings.FolderLeader
		case m.batch.dropdown:
			return settings.FolderMarking
		}
		return settings.Folder
	case batchMode:
		if m.batch.choosing() {
			return settings.BatchLayout
		}
		return settings.BatchMarking
	case pathMode, childMode:
		if m.path.Searching {
			return settings.Typing
		}
		if m.mode == childMode {
			return settings.PathChildren
		}
		return settings.PathBar
	case inventoryMode:
		switch {
		case m.inventory.err != nil, m.inventory.pending != nil:
			return settings.Typing
		case m.inventory.specs != nil:
			return settings.InventorySpecs
		}
		return settings.Inventory
	case infoMode:
		return settings.ExplainPopup
	}
	return settings.Typing
}

// digit is the 0-based number a 1-9 key, bare or in a chord, ends in.
func digit(msg tea.KeyMsg) int {
	key := msg.String()
	return int(key[len(key)-1] - '1')
}

// IsUp checks if the key matches any "up" navigation key.
func IsUp(c settings.InputConfig, msg tea.KeyMsg) bool {
	return slices.Contains(c.NavUp, msg.String())
}

// IsDown checks if the key matches any "down" navigation key.
func IsDown(c settings.InputConfig, msg tea.KeyMsg) bool {
	return slices.Contains(c.NavDown, msg.String())
}

// IsLeft checks if the key matches any "left" navigation key.
func IsLeft(c settings.InputConfig, msg tea.KeyMsg) bool {
	return slices.Contains(c.NavLeft, msg.String())
}

// IsRight checks if the key matches any "right" navigation key.
func IsRight(c settings.InputConfig, msg tea.KeyMsg) bool {
	return slices.Contains(c.NavRight, msg.String())
}
