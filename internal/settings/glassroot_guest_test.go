package settings

import "testing"

// What a glassroot guest may do is decided here, by name. The key table marks
// what glassroot blocks (Gated), so on its own a new key or screen would be
// open to guests until someone remembered to gate it. These two lists turn
// that around: a new action or screen fails this test until it is either
// gated in the table or added here on purpose.

// guestActions are the actions a guest may use.
var guestActions = map[Action]bool{
	Quit:        true,
	LockSession: true,
	Help:        true,
	SwitchDeck:  true, // every equipped profile is the guest's: the operator's contract
	PickDeck:    true,
	PrevDeck:    true,
	NextDeck:    true,
	Leader:      true,
	Jump:        true,
	Back:        true,
	Move:        true,
	MoveV:       true,
	Explain:     true, // shows the command text
	Confirm:     true, // runs the cell: what the kiosk is for
	Scroll:      true,
	Close:       true,
}

// guestScreens are the screens a guest can reach. Every other screen lies
// behind a gated key (batch, path mode, inventory) and is named in
// closedScreens, so a new screen has to be put in one list or the other.
var guestScreens = map[ScreenID]bool{
	Typing:       true,
	Grid:         true,
	GridLeader:   true,
	Folder:       true,
	FolderLeader: true,
	ExplainPopup: true,
}

var closedScreens = map[ScreenID]bool{
	FolderMarking:  true, // behind batch
	BatchMarking:   true,
	BatchLayout:    true,
	PathBar:        true, // behind path mode
	PathChildren:   true,
	Inventory:      true, // behind the inventory key
	InventorySpecs: true,
}

func TestEveryScreenIsDecidedForGlassroot(t *testing.T) {
	for id := Typing; id < screenCount; id++ {
		if guestScreens[id] == closedScreens[id] {
			t.Errorf("screen %d (%s): decide whether a glassroot guest can reach it, and list it once in glassroot_guest_test.go",
				id, defaults().Screen(id).Name)
		}
	}
}

func TestGuestScreensOfferOnlyGuestActions(t *testing.T) {
	cfg := defaults()
	for id := range guestScreens {
		s := cfg.Screen(id)
		for _, b := range s.Binds {
			if !b.Gated && !guestActions[b.Act] {
				t.Errorf("%s: %q (%s) is open to a glassroot guest: gate it in keymap.go or add its action to guestActions",
					s.Name, b.Label(), b.Does)
			}
		}
	}
}

// The way into each closed screen is gated; were one opened, its screens
// would be the guest's without this test noticing.
func TestTheWaysIntoClosedScreensAreGated(t *testing.T) {
	cfg := defaults()
	for _, way := range []struct {
		screen ScreenID
		act    Action
	}{
		{Grid, OpenInventory}, {Grid, PathMode},
		{GridLeader, Batch}, {FolderLeader, Batch},
	} {
		b, ok := cfg.Screen(way.screen).Bind(way.act)
		if !ok || !b.Gated {
			t.Errorf("%s: action %d must be gated for glassroot", cfg.Screen(way.screen).Name, way.act)
		}
	}
}
