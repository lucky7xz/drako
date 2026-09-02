package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
)

// altRunes is the chord bubbletea reports for Alt held with a letter.
func altRunes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s), Alt: true}
}

func lockTestModel(t *testing.T) Model {
	t.Helper()
	cfg := config.Config{X: 1, Y: 1}
	cfg.ApplyDefaults()
	return Model{Config: cfg, styles: BuildStyles(cfg), mode: gridMode}
}

// update drives one key through the top-level handler, which is where a
// binding that reaches across every mode has to be caught.
func update(t *testing.T, m Model, msg tea.KeyMsg) Model {
	t.Helper()
	next, _ := m.Update(msg)
	out, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	return out
}

func TestIsSessionLock(t *testing.T) {
	keys := config.InputConfig{Lock: "r", SessionLock: "r"}
	cases := []struct {
		name     string
		msg      tea.KeyMsg
		modifier string
		want     bool
	}{
		{"the chord", altRunes("r"), "alt", true},
		{"the bare letter is the profile lock", keyRunes("r"), "alt", false},
		{"a different letter", altRunes("q"), "alt", false},
		{"the wrong modifier", altRunes("r"), "ctrl", false},
	}
	for _, c := range cases {
		if got := IsSessionLock(keys, c.msg, c.modifier); got != c.want {
			t.Errorf("%s: IsSessionLock(%q, %q) = %v, want %v", c.name, c.msg.String(), c.modifier, got, c.want)
		}
	}

	// It follows a rebound letter, independently of the profile lock.
	rebound := config.InputConfig{Lock: "r", SessionLock: "x"}
	if !IsSessionLock(rebound, altRunes("x"), "alt") {
		t.Error("session_lock = \"x\" should answer to alt+x")
	}
	if IsSessionLock(rebound, altRunes("r"), "alt") {
		t.Error("alt+r must not lock once session_lock has been moved")
	}
}

func TestSessionLock_ChordLocksTheSession(t *testing.T) {
	m := update(t, lockTestModel(t), altRunes("r"))
	if m.mode != lockedMode {
		t.Fatalf("alt+r should lock the session, mode = %v", m.mode)
	}
	if m.lock.progress != 0 {
		t.Errorf("a fresh lock starts the slider empty, got %d", m.lock.progress)
	}
}

// The moment you want this is the moment you are walking away — a filter box
// left open must not swallow it. It is a chord, so nothing was typing it.
func TestSessionLock_ReachableWhileATextFieldIsOpen(t *testing.T) {
	m := lockTestModel(t)
	m.mode = pathMode
	m.path.Searching = true
	if !m.capturingText() {
		t.Fatal("fixture should be capturing text")
	}

	m = update(t, m, altRunes("r"))
	if m.mode != lockedMode {
		t.Fatalf("alt+r must lock even mid-search, mode = %v", m.mode)
	}
}

// Glassroot blocks the profile lock, the inventory and path mode — all of which
// expose config. This hides the screen and grants nothing, so a guest keeps it.
func TestSessionLock_AllowedInGlassroot(t *testing.T) {
	m := lockTestModel(t)
	m.GlassrootMode = true

	m = update(t, m, altRunes("r"))
	if m.mode != lockedMode {
		t.Fatalf("glassroot must still allow the session lock, mode = %v", m.mode)
	}
}

// The idle timer is what auto_lock_enabled governs. A key the user pressed is
// not idleness.
func TestSessionLock_WorksWithAutoLockDisabled(t *testing.T) {
	m := lockTestModel(t)
	off := false
	m.Config.AutoLockEnabled = &off

	m = update(t, m, altRunes("r"))
	if m.mode != lockedMode {
		t.Fatalf("a manual lock does not depend on the idle timer, mode = %v", m.mode)
	}
}

// Pumping out returns you where you were, not to the grid.
func TestSessionLock_ReturnsToTheModeItLockedFrom(t *testing.T) {
	m := lockTestModel(t)
	m.mode = inventoryMode

	m = update(t, m, altRunes("r"))
	if m.mode != lockedMode {
		t.Fatalf("mode = %v, want lockedMode", m.mode)
	}
	if got := m.exitLockedMode().mode; got != inventoryMode {
		t.Errorf("unlocking returned to %v, want the inventory it locked from", got)
	}
}

// The two locks live one modifier apart; they must not bleed into each other.
func TestSessionLock_BareLetterStillTogglesTheProfileLock(t *testing.T) {
	m := update(t, lockTestModel(t), keyRunes("r"))
	if m.mode == lockedMode {
		t.Fatal("bare r is the profile lock, not the session lock")
	}
}
