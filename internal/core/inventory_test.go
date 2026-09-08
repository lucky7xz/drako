package core

import (
	"testing"
)

func TestNewInventoryState(t *testing.T) {
	v := []string{"a", "b"}
	i := []string{"c"}
	s := NewInventoryState(v, i, 9)

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
	s := NewInventoryState([]string{"a", "b"}, []string{}, 9)

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
	s := NewInventoryState([]string{"b"}, []string{}, 9)

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

// The cap only blocks growth. Rearranging and stashing must keep working, or a
// list that arrived over the cap would lock its owner out of fixing it.
func TestPlaceItemCap(t *testing.T) {
	t.Run("equipping at the cap is refused", func(t *testing.T) {
		s := NewInventoryState([]string{"a", "b"}, []string{"c"}, 2)
		if err := s.PickUpItem(ListInventory, 0); err != nil {
			t.Fatalf("PickUpItem: %v", err)
		}
		if err := s.PlaceItem(ListVisible, 0); err == nil {
			t.Fatal("expected the cap to refuse a third equipped profile")
		}
		if len(s.Visible) != 2 {
			t.Errorf("refused place must not mutate Visible: %v", s.Visible)
		}
	})

	t.Run("equipping below the cap is allowed", func(t *testing.T) {
		s := NewInventoryState([]string{"a"}, []string{"c"}, 2)
		if err := s.PickUpItem(ListInventory, 0); err != nil {
			t.Fatalf("PickUpItem: %v", err)
		}
		if err := s.PlaceItem(ListVisible, 1); err != nil {
			t.Fatalf("PlaceItem below cap: %v", err)
		}
		if len(s.Visible) != 2 || s.Visible[1] != "c" {
			t.Errorf("Visible = %v, want [a c]", s.Visible)
		}
	})

	t.Run("rearranging over the cap is allowed", func(t *testing.T) {
		s := NewInventoryState([]string{"a", "b", "c", "d"}, nil, 2)
		if err := s.PickUpItem(ListVisible, 3); err != nil {
			t.Fatalf("PickUpItem: %v", err)
		}
		if err := s.PlaceItem(ListVisible, 0); err != nil {
			t.Fatalf("rearrange over cap: %v", err)
		}
		if len(s.Visible) != 4 || s.Visible[0] != "d" {
			t.Errorf("Visible = %v, want [d a b c]", s.Visible)
		}
	})

	t.Run("stashing over the cap is allowed", func(t *testing.T) {
		s := NewInventoryState([]string{"a", "b", "c"}, nil, 2)
		if err := s.PickUpItem(ListVisible, 0); err != nil {
			t.Fatalf("PickUpItem: %v", err)
		}
		if err := s.PlaceItem(ListInventory, 0); err != nil {
			t.Fatalf("stash over cap: %v", err)
		}
		if len(s.Visible) != 2 || len(s.Inventory) != 1 {
			t.Errorf("Visible = %v, Inventory = %v", s.Visible, s.Inventory)
		}
	})
}

func TestArrange_OrderFollowsDesired(t *testing.T) {
	s := NewInventoryState([]string{"a", "b"}, []string{"c", "d"}, 9)

	if err := s.Arrange([]string{"d", "a"}); err != nil {
		t.Fatalf("Arrange: %v", err)
	}
	if len(s.Visible) != 2 || s.Visible[0] != "d" || s.Visible[1] != "a" {
		t.Errorf("Visible = %v, want [d a]", s.Visible)
	}
}

func TestArrange_LeftoversStashedSorted(t *testing.T) {
	s := NewInventoryState([]string{"b", "a"}, []string{"d", "c"}, 9)

	if err := s.Arrange([]string{"a"}); err != nil {
		t.Fatalf("Arrange: %v", err)
	}
	if len(s.Inventory) != 3 || s.Inventory[0] != "b" || s.Inventory[1] != "c" || s.Inventory[2] != "d" {
		t.Errorf("Inventory = %v, want [b c d]", s.Inventory)
	}
}

// A spec may name a profile that was never installed. Arrange stages what it
// has; deciding whether that is acceptable belongs to the caller, which can
// see the spec and say what is missing.
func TestArrange_IgnoresUnknownNames(t *testing.T) {
	s := NewInventoryState([]string{"a"}, []string{"b"}, 9)

	if err := s.Arrange([]string{"b", "ghost"}); err != nil {
		t.Fatalf("Arrange: %v", err)
	}
	if len(s.Visible) != 1 || s.Visible[0] != "b" {
		t.Errorf("Visible = %v, want [b]", s.Visible)
	}
	if len(s.Inventory) != 1 || s.Inventory[0] != "a" {
		t.Errorf("Inventory = %v, want [a]", s.Inventory)
	}
}

func TestArrange_EmptyDesiredStashesEverything(t *testing.T) {
	s := NewInventoryState([]string{"a", "b"}, []string{"c"}, 9)

	if err := s.Arrange(nil); err != nil {
		t.Fatalf("Arrange: %v", err)
	}
	if len(s.Visible) != 0 {
		t.Errorf("Visible = %v, want empty", s.Visible)
	}
	if len(s.Inventory) != 3 {
		t.Errorf("Inventory = %v, want all three", s.Inventory)
	}
}

// The cap is refused here rather than at Apply: profiles.Reconcile would
// reject the growth anyway, and an arrangement you cannot apply is a dead end.
func TestArrange_RefusesOverCap(t *testing.T) {
	s := NewInventoryState([]string{"a"}, []string{"b", "c"}, 2)

	if err := s.Arrange([]string{"a", "b", "c"}); err == nil {
		t.Fatal("arranging 3 into a cap of 2 should fail")
	}
	if len(s.Visible) != 1 || s.Visible[0] != "a" {
		t.Errorf("a refused Arrange must not touch the lists, got %v", s.Visible)
	}
}

func TestArrange_RefusesWhileHolding(t *testing.T) {
	s := NewInventoryState([]string{"a", "b"}, nil, 9)
	if err := s.PickUpItem(ListVisible, 0); err != nil {
		t.Fatalf("PickUpItem: %v", err)
	}

	if err := s.Arrange([]string{"b"}); err == nil {
		t.Fatal("Arrange should refuse while an item is held")
	}
	if s.HeldItem == nil {
		t.Error("the held item must survive a refused Arrange")
	}
}

func TestArrange_DuplicatesCountOnce(t *testing.T) {
	s := NewInventoryState([]string{"a"}, []string{"b"}, 9)

	if err := s.Arrange([]string{"a", "a", "b"}); err != nil {
		t.Fatalf("Arrange: %v", err)
	}
	if len(s.Visible) != 2 || s.Visible[0] != "a" || s.Visible[1] != "b" {
		t.Errorf("Visible = %v, want [a b]", s.Visible)
	}
}

// The cap refuses growth, not the cap itself — the same monotonic rule
// PlaceItem keeps. A list already over the limit (a spec applied from the CLI
// with consent) has to stay workable, and stashing is how you get back under.
func TestArrange_AlreadyOverCapCanStillShrink(t *testing.T) {
	s := NewInventoryState([]string{"a", "b", "c", "d"}, nil, 2)

	if err := s.Arrange([]string{"a", "b", "c"}); err != nil {
		t.Fatalf("shrinking an over-cap list should be allowed: %v", err)
	}
	if len(s.Visible) != 3 {
		t.Errorf("Visible = %v, want three", s.Visible)
	}

	// Growing it again is still refused.
	if err := s.Arrange([]string{"a", "b", "c", "d"}); err == nil {
		t.Error("growing back past the cap should be refused")
	}
}

// CanArrange is what a picker asks before offering a row, so the two must
// never disagree: anything it accepts, Arrange performs.
func TestCanArrange_AgreesWithArrange(t *testing.T) {
	cases := []struct {
		name    string
		state   *InventoryState
		desired []string
	}{
		{"fits", NewInventoryState([]string{"a"}, []string{"b"}, 9), []string{"a", "b"}},
		{"over cap", NewInventoryState([]string{"a"}, []string{"b", "c"}, 2), []string{"a", "b", "c"}},
		{"unknown names", NewInventoryState([]string{"a"}, nil, 9), []string{"ghost"}},
		{"empty", NewInventoryState([]string{"a"}, nil, 9), nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			check := tc.state.CanArrange(tc.desired)
			do := tc.state.Arrange(tc.desired)
			if (check == nil) != (do == nil) {
				t.Errorf("CanArrange = %v but Arrange = %v", check, do)
			}
		})
	}
}

func TestCanArrange_DoesNotMutate(t *testing.T) {
	s := NewInventoryState([]string{"a"}, []string{"b"}, 9)

	if err := s.CanArrange([]string{"b"}); err != nil {
		t.Fatalf("CanArrange: %v", err)
	}
	if len(s.Visible) != 1 || s.Visible[0] != "a" {
		t.Errorf("CanArrange must not touch the lists, Visible = %v", s.Visible)
	}
}
