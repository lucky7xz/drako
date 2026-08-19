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
