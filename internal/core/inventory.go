package core

import (
	"errors"
	"path/filepath"
)

const (
	ListVisible   = 0
	ListInventory = 1
)

// InventoryState manages the state of profiles in the application.
type InventoryState struct {
	Visible   []string
	Inventory []string
	HeldItem  *string
}

// NewInventoryState creates a new state with copies of the provided lists.
func NewInventoryState(visible, inventory []string) *InventoryState {
	v := make([]string, len(visible))
	copy(v, visible)
	i := make([]string, len(inventory))
	copy(i, inventory)
	return &InventoryState{
		Visible:   v,
		Inventory: i,
	}
}

// GetList returns a pointer to the slice for the given list ID.
func (s *InventoryState) GetList(listID int) (*[]string, error) {
	switch listID {
	case ListVisible:
		return &s.Visible, nil
	case ListInventory:
		return &s.Inventory, nil
	default:
		return nil, errors.New("invalid list ID")
	}
}

// PickUpItem removes the item at the given index from the source list and holds it.
// Returns error if an item is already held or index is invalid.
func (s *InventoryState) PickUpItem(listID, index int) error {
	if s.HeldItem != nil {
		return errors.New("already holding an item")
	}

	listPtr, err := s.GetList(listID)
	if err != nil {
		return err
	}
	list := *listPtr

	if index < 0 || index >= len(list) {
		return errors.New("index out of bounds")
	}

	item := list[index]
	s.HeldItem = &item

	// Remove item from list
	*listPtr = append(list[:index], list[index+1:]...)
	return nil
}

// PlaceItem inserts the held item into the destination list at the given index.
// Returns error if no item is held.
func (s *InventoryState) PlaceItem(listID, index int) error {
	if s.HeldItem == nil {
		return errors.New("no item held")
	}

	listPtr, err := s.GetList(listID)
	if err != nil {
		return err
	}
	list := *listPtr

	// Clamp index
	if index < 0 {
		index = 0
	}
	if index > len(list) {
		index = len(list)
	}

	// Insert item
	*listPtr = append(list[:index], append([]string{*s.HeldItem}, list[index:]...)...)
	s.HeldItem = nil
	return nil
}

// CalculateMoves determines the file operations needed to transition from the original state to current state.
// Returns a map of source paths to destination paths.
// configDir is the root directory; inventoryDir is handled internally.
func (s *InventoryState) CalculateMoves(configDir string, originalVisible, originalInventory []string) (map[string]string, error) {
	moves := make(map[string]string)
	inventoryDir := filepath.Join(configDir, "inventory")

	// Helper to check existence in a slice
	contains := func(slice []string, item string) bool {
		for _, v := range slice {
			if v == item {
				return true
			}
		}
		return false
	}

	// 1. Moves from Visibility -> Inventory
	for _, file := range originalVisible {
		// If it was in Visible but is now in Inventory
		if contains(s.Inventory, file) && !contains(s.Visible, file) {
			src := filepath.Join(configDir, file)
			dst := filepath.Join(inventoryDir, file)
			moves[src] = dst
		}
	}

	// 2. Moves from Inventory -> Visibility
	for _, file := range originalInventory {
		// If it was in Inventory but is now in Visible
		if contains(s.Visible, file) && !contains(s.Inventory, file) {
			src := filepath.Join(inventoryDir, file)
			dst := filepath.Join(configDir, file)
			moves[src] = dst
		}
	}

	// Note: Reordering within the same list does not require file moves, only metadata updates (handled elsewhere).
	// However, we should check for duplicates or lost files if needed, but this basic logic covers the file system 'mv' cmds.

	return moves, nil
}
