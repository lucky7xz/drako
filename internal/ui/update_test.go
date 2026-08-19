package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
	"github.com/lucky7xz/drako/internal/profiles"
)

// Helper to create a basic model for Grid testing
func createTestGridModel() Model {
	m := Model{
		mode: gridMode,
		gridNav: gridNav{
			grid: [][]string{
				{"A", "B"},
				{"C", "D"},
			},
		},
		Config: config.Config{
			Keys: config.InputConfig{
				NavUp:    []string{"up", "k"},
				NavDown:  []string{"down", "j"},
				NavLeft:  []string{"left", "h"},
				NavRight: []string{"right", "l"},
			},
		},
	}
	return m
}

func TestUpdateGridMode_Navigation(t *testing.T) {
	m := createTestGridModel()
	var tm tea.Model

	// Test Right
	tm, _ = m.updateGridMode(tea.KeyMsg{Type: tea.KeyRight})
	m = tm.(Model)
	if m.gridNav.cursorCol != 1 {
		t.Errorf("Expected cursorCol 1, got %d", m.gridNav.cursorCol)
	}

	// Test Down
	tm, _ = m.updateGridMode(tea.KeyMsg{Type: tea.KeyDown})
	m = tm.(Model)
	if m.gridNav.cursorRow != 1 {
		t.Errorf("Expected cursorRow 1, got %d", m.gridNav.cursorRow)
	}

	// Test Left
	tm, _ = m.updateGridMode(tea.KeyMsg{Type: tea.KeyLeft})
	m = tm.(Model)
	if m.gridNav.cursorCol != 0 {
		t.Errorf("Expected cursorCol 0, got %d", m.gridNav.cursorCol)
	}

	// Test Up
	tm, _ = m.updateGridMode(tea.KeyMsg{Type: tea.KeyUp})
	m = tm.(Model)
	if m.gridNav.cursorRow != 0 {
		t.Errorf("Expected cursorRow 0, got %d", m.gridNav.cursorRow)
	}
}

func createTestInventoryModel() Model {
	state := core.NewInventoryState(
		[]string{"a.profile.toml", "b.profile.toml"}, // Visible
		[]string{"c.profile.toml", "d.profile.toml"}, // Inventory
		profiles.MaxEquipped,
	)
	m := Model{
		mode: inventoryMode,
		inventory: inventoryModel{
			State:       state,
			cursor:      0,
			focusedList: 0, // 0=Visible, 1=Inventory, 2=Apply, 3=Rescue
		},
		Config: config.Config{
			Keys: config.InputConfig{
				NavUp:    []string{"up", "k"},
				NavDown:  []string{"down", "j"},
				NavLeft:  []string{"left", "h"},
				NavRight: []string{"right", "l"},
			},
		},
	}
	return m
}

func TestUpdateInventoryMode_Navigation(t *testing.T) {
	m := createTestInventoryModel()
	var tm tea.Model

	// Initial State: Focused List 0 (Visible), Cursor 0

	// Test Right (Next item in list)
	tm, _ = m.updateInventoryMode(tea.KeyMsg{Type: tea.KeyRight})
	m = tm.(Model)
	if m.inventory.cursor != 1 {
		t.Errorf("Expected cursor 1, got %d", m.inventory.cursor)
	}

	// Test Down (Switch to Inventory List - index 1)
	tm, _ = m.updateInventoryMode(tea.KeyMsg{Type: tea.KeyDown})
	m = tm.(Model)
	if m.inventory.focusedList != 1 {
		t.Errorf("Expected focusedList 1, got %d", m.inventory.focusedList)
	}
	// Cursor should reset to 0 on list switch
	if m.inventory.cursor != 0 {
		t.Errorf("Expected cursor 0 after list switch, got %d", m.inventory.cursor)
	}

	// Test Up (Switch back to Visible List - index 0)
	tm, _ = m.updateInventoryMode(tea.KeyMsg{Type: tea.KeyUp})
	m = tm.(Model)
	if m.inventory.focusedList != 0 {
		t.Errorf("Expected focusedList 0, got %d", m.inventory.focusedList)
	}
}
