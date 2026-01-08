package core

import (
	"path/filepath"
	"testing"
)

func TestNewInventoryState(t *testing.T) {
	v := []string{"a", "b"}
	i := []string{"c"}
	s := NewInventoryState(v, i)

	if len(s.Visible) != 2 || s.Visible[0] != "a" {
		t.Errorf("Visible list initialization failed")
	}
	if len(s.Inventory) != 1 || s.Inventory[0] != "c" {
		t.Errorf("Inventory list initialization failed")
	}
	// Verify deep copy
	v[0] = "z"
	if s.Visible[0] == "z" {
		t.Errorf("NewInventoryState should deep copy input slices")
	}
}

func TestPickUpItem(t *testing.T) {
	s := NewInventoryState([]string{"a", "b"}, []string{})

	// Test valid pickup
	err := s.PickUpItem(ListVisible, 0)
	if err != nil {
		t.Fatalf("PickUpItem failed: %v", err)
	}
	if s.HeldItem == nil || *s.HeldItem != "a" {
		t.Errorf("HeldItem incorrect, got %v", s.HeldItem)
	}
	if len(s.Visible) != 1 || s.Visible[0] != "b" {
		t.Errorf("Item not removed from list properly")
	}

	// Test pickup while holding
	err = s.PickUpItem(ListVisible, 0)
	if err == nil {
		t.Error("Should not allow pickup while holding item")
	}

	// Test out of bounds
	s.HeldItem = nil // Reset
	err = s.PickUpItem(ListVisible, 99)
	if err == nil {
		t.Error("Should not allow pickup out of bounds")
	}
}

func TestPlaceItem(t *testing.T) {
	s := NewInventoryState([]string{"b"}, []string{})

	// Test place without holding
	err := s.PlaceItem(ListVisible, 0)
	if err == nil {
		t.Error("Should not allow place without holding")
	}

	// Test valid place
	item := "a"
	s.HeldItem = &item
	err = s.PlaceItem(ListVisible, 0)
	if err != nil {
		t.Fatalf("PlaceItem failed: %v", err)
	}
	if len(s.Visible) != 2 || s.Visible[0] != "a" || s.Visible[1] != "b" {
		t.Errorf("Item placed incorrectly: %v", s.Visible)
	}
	if s.HeldItem != nil {
		t.Error("HeldItem should be nil after place")
	}

	// Test append to end
	item = "c"
	s.HeldItem = &item
	err = s.PlaceItem(ListVisible, 2)
	if err != nil {
		t.Fatalf("PlaceItem at end failed: %v", err)
	}
	if s.Visible[2] != "c" {
		t.Errorf("Item not appended correctly")
	}
}

func TestCalculateMoves(t *testing.T) {
	initialVisible := []string{"keep.toml", "move_to_inv.toml"}
	initialInventory := []string{"keep_inv.toml", "move_to_vis.toml"}

	// Simulate the moves by manually setting the state
	// Resulting state:
	// Visible: keep.toml, move_to_vis.toml
	// Inventory: keep_inv.toml, move_to_inv.toml
	currentVisible := []string{"keep.toml", "move_to_vis.toml"}
	currentInventory := []string{"keep_inv.toml", "move_to_inv.toml"}

	s := NewInventoryState(currentVisible, currentInventory)

	configDir := "/test/config"
	moves, err := s.CalculateMoves(configDir, initialVisible, initialInventory)
	if err != nil {
		t.Fatalf("CalculateMoves failed: %v", err)
	}

	expectedMoves := 2
	if len(moves) != expectedMoves {
		t.Errorf("Expected %d moves, got %d", expectedMoves, len(moves))
	}

	// Verify Move To Inventory
	src1 := filepath.Join(configDir, "move_to_inv.toml")
	dst1 := filepath.Join(configDir, "inventory", "move_to_inv.toml")
	if moves[src1] != dst1 {
		t.Errorf("Missing move for move_to_inv.toml: got %v", moves[src1])
	}

	// Verify Move To Visible
	src2 := filepath.Join(configDir, "inventory", "move_to_vis.toml")
	dst2 := filepath.Join(configDir, "move_to_vis.toml")
	if moves[src2] != dst2 {
		t.Errorf("Missing move for move_to_vis.toml: got %v", moves[src2])
	}
}
