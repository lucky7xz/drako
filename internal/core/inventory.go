package core

import (
	"errors"
	"fmt"
	"sort"
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

// Arrange stages a whole arrangement at once: every file named in desired
// becomes Visible, in desired's order; everything else the state holds is
// stashed into Inventory, sorted. Names the state does not hold are ignored,
// and a name repeated in desired is placed once.
//
// This is the one primitive behind every spec function. Equipping a spec,
// stashing a spec and stripping differ only in the desired list handed to it,
// which is why none of them needs its own file-moving code: the arrangement is
// staged, and Apply reconciles it exactly as it reconciles a hand-dragged one.
//
// desired carries filenames — the strings the lists already hold. Matching a
// user's bare profile name to a file is the caller's job.
func (s *InventoryState) Arrange(desired []string) error {
	visible, inventory, err := s.planArrange(desired)
	if err != nil {
		return err
	}
	s.Visible, s.Inventory = visible, inventory
	return nil
}

// CanArrange reports whether Arrange would succeed, without doing it. A caller
// that offers arrangements to choose from asks this first, so it can refuse at
// the point of choice rather than staging something Apply would reject.
//
// It exists so the rule is stated once. A UI that re-implemented "would this
// exceed the cap?" would be a second copy of a policy that lives here.
func (s *InventoryState) CanArrange(desired []string) error {
	_, _, err := s.planArrange(desired)
	return err
}

// planArrange computes the arrangement without applying it. Both Arrange and
// CanArrange go through here, so a plan can never be judged by one rule and
// applied under another.
func (s *InventoryState) planArrange(desired []string) (visible, inventory []string, err error) {
	if s.HeldItem != nil {
		return nil, nil, errors.New("place the held item first")
	}

	// Index what actually exists, so desired can name profiles that don't.
	held := make(map[string]bool, len(s.Visible)+len(s.Inventory))
	for _, f := range s.Visible {
		held[f] = true
	}
	for _, f := range s.Inventory {
		held[f] = true
	}

	visible = make([]string, 0, len(desired))
	taken := make(map[string]bool, len(desired))
	for _, f := range desired {
		if !held[f] || taken[f] {
			continue
		}
		taken[f] = true
		visible = append(visible, f)
	}

	// Refuse growth past the cap, not the cap itself — the same monotonic rule
	// PlaceItem and profiles.Reconcile keep. A list already over it (a spec
	// applied from the CLI, with consent) has to stay workable, and stashing
	// is how you get back under.
	if len(visible) > s.MaxVisible && len(visible) > len(s.Visible) {
		return nil, nil, fmt.Errorf("that would equip %d profiles, over the limit of %d", len(visible), s.MaxVisible)
	}

	for f := range held {
		if !taken[f] {
			inventory = append(inventory, f)
		}
	}
	sort.Strings(inventory)

	return visible, inventory, nil
}
