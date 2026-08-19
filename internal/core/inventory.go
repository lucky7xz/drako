package core

import (
	"errors"
	"fmt"
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
	// HeldFrom is the list PickUpItem lifted from, which is what tells a
	// rearrange apart from an equip. PickUpItem is the only thing that sets
	// HeldItem, so it is always meaningful.
	HeldFrom int
	// MaxVisible caps how many profiles may be equipped. Lists already over
	// it stay usable; see PlaceItem.
	MaxVisible int
}

// NewInventoryState creates a new state with copies of the provided lists,
// capped at maxVisible equipped profiles.
func NewInventoryState(visible, inventory []string, maxVisible int) *InventoryState {
	v := make([]string, len(visible))
	copy(v, visible)
	i := make([]string, len(inventory))
	copy(i, inventory)
	return &InventoryState{
		Visible:    v,
		Inventory:  i,
		MaxVisible: maxVisible,
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
	s.HeldFrom = listID

	// Remove item from list
	*listPtr = append(list[:index], list[index+1:]...)
	return nil
}

// PlaceItem inserts the held item into the destination list at the given index.
// Returns error if no item is held, or if equipping this item would push the
// visible list past MaxVisible. Rearranging within the visible list is always
// allowed — the count doesn't change — so a list already over the cap stays
// workable instead of locking its owner out.
func (s *InventoryState) PlaceItem(listID, index int) error {
	if s.HeldItem == nil {
		return errors.New("no item held")
	}

	listPtr, err := s.GetList(listID)
	if err != nil {
		return err
	}
	list := *listPtr

	if listID == ListVisible && s.HeldFrom != ListVisible && len(list) >= s.MaxVisible {
		return fmt.Errorf("equipped is full (%d max) — stash one first", s.MaxVisible)
	}

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
