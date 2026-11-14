package settings

import "testing"

func defaults() Config {
	var c Config
	c.ApplyDefaults()
	return c
}

// A screen gives a key to the first bind that claims it; that order is what
// the TUI does when a config puts two actions on one key.
func TestScreenGivesAKeyToTheFirstBind(t *testing.T) {
	c := defaults()
	c.Keys.Explain, c.Keys.Lock, c.Keys.Inventory = "q", "3", "w"
	grid := c.Screen(Grid)
	for key, want := range map[string]Action{
		"q": Back,          // drako's quit comes before explain
		"3": PinDeck,       // the deck keys come before the 1-9 jump
		"w": OpenInventory, // inventory comes before the move keys
		"z": NoAction,
	} {
		if got := grid.Act(key); got != want {
			t.Errorf("grid: %q does %v, want %v", key, got, want)
		}
	}
}

// ctrl+c and the session lock work on every screen, a text field included,
// ahead of anything a config binds.
func TestEveryScreenKeepsTheKeysThatWorkEverywhere(t *testing.T) {
	c := defaults()
	c.Keys.Help, c.Keys.Explain = "ctrl+c", "alt+r"
	for id := Typing; id < screenCount; id++ {
		s := c.Screen(id)
		if s.Act("ctrl+c") != Quit || s.Act("alt+r") != LockSession {
			t.Errorf("%s: ctrl+c does %v, alt+r does %v", s.Name, s.Act("ctrl+c"), s.Act("alt+r"))
		}
	}
}

// Under the defaults no screen has two binds on one key.
func TestDefaultsNeverShareAKey(t *testing.T) {
	for _, s := range defaults().Screens() {
		seen := map[string]Bind{}
		for _, b := range s.Binds {
			for _, k := range b.Keys {
				if first, taken := seen[k]; taken {
					t.Errorf("%s: %q is both %q and %q", s.Name, k, first.Does, b.Does)
				}
				seen[k] = b
			}
		}
	}
}
