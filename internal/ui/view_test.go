package ui

import (
	"strings"
	"testing"

	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
)

// Helper to create a model for view testing
func createTestModelForView(mode navMode) Model {
	// Minimal config with small grid to avoid "Terminal too small" overlay
	cfg := config.Config{
		X:            2,
		Y:            2,
		Theme:        "default",
		DefaultShell: "/bin/bash",
	}

	// Create a grid
	grid := [][]string{
		{"Cmd1", "Cmd2"},
		{"Cmd3", "Cmd4"},
	}

	m := Model{
		mode:       mode,
		termWidth:  100, // Generous width
		termHeight: 50,  // Generous height
		Config:     cfg,
		grid:       grid,
		// Inventory model needed for inventory mode
		inventory: inventoryModel{
			State: core.NewInventoryState(
				[]string{"A.profile.toml"},
				[]string{"B.profile.toml"},
			),
			focusedList: 0,
		},
		activeProfileIndex: 0,
		profiles: []config.ProfileInfo{
			{Name: "Core"},
		},
	}

	// Force initialization of layout-dependent fields if any?
	// View() usually calculates layout on the fly.

	return m
}

func TestView_GridMode(t *testing.T) {
	m := createTestModelForView(gridMode)
	output := m.View()

	// Check for Grid-specific elements
	if !strings.Contains(output, "Grid Mode") {
		t.Errorf("View output missing 'Grid Mode' indicator. Got:\n%s", output)
	}
	if !strings.Contains(output, "Cmd1") {
		t.Error("View output missing grid content 'Cmd1'")
	}
	// Check for Header Column indicators (e.g., [A], [B])
	if !strings.Contains(output, "[A]") {
		t.Error("View output missing column header '[A]'")
	}
	// Check for Row Number indicators (e.g., 1❭)
	if !strings.Contains(output, "1❭") {
		t.Errorf("View output missing row number '1❭'. Got:\n%s", output)
	}
}

func TestView_InventoryMode(t *testing.T) {
	m := createTestModelForView(inventoryMode)
	output := m.View()

	if !strings.Contains(output, "Inventory Management") {
		t.Errorf("View output missing 'Inventory Management' title. Got:\n%s", output)
	}
	if !strings.Contains(output, "Equipped Items") {
		t.Error("View output missing 'Equipped Items' header")
	}
	if !strings.Contains(output, "Inventory Items") {
		t.Error("View output missing 'Inventory Items' header")
	}
	if !strings.Contains(output, "A.profile.toml") {
		t.Error("View output missing visible item 'A.profile.toml'")
	}
	if !strings.Contains(output, "[ Apply Changes ]") {
		t.Error("View output missing Apply button")
	}
}

func TestView_LockedMode(t *testing.T) {
	m := createTestModelForView(lockedMode)
	output := m.View()

	if !strings.Contains(output, "Session Locked") {
		t.Errorf("View output missing 'Session Locked' title. Got:\n%s", output)
	}
	if !strings.Contains(output, "Pump") {
		t.Error("View output missing 'Pump' instruction")
	}
}
