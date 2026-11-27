package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/lair"
	"github.com/lucky7xz/drako/internal/profile"
	"github.com/lucky7xz/drako/internal/settings"
)

// altRunes is the chord bubbletea reports for Alt held with a letter.
func altRunes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s), Alt: true}
}

func lockTestModel(t *testing.T) Model {
	t.Helper()
	cfg := settings.Config{X: 1, Y: 1}
	cfg.ApplyDefaults()
	return Model{Config: cfg, styles: BuildStyles(cfg, ""), mode: gridMode}
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

func TestSessionLockChord(t *testing.T) {
	keys := settings.InputConfig{Lock: "r", SessionLock: "r"}
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
	// The lock is one of the keys that work everywhere, typing included.
	locks := func(keys settings.InputConfig, msg tea.KeyMsg, modifier string) bool {
		cfg := settings.Config{Keys: keys, NumbModifier: modifier}
		return cfg.Screen(settings.Typing).Act(msg.String()) == settings.LockSession
	}
	for _, c := range cases {
		if got := locks(keys, c.msg, c.modifier); got != c.want {
			t.Errorf("%s: %q with modifier %q locks = %v, want %v", c.name, c.msg.String(), c.modifier, got, c.want)
		}
	}

	// It follows a rebound letter, independently of the profile lock.
	rebound := settings.InputConfig{Lock: "r", SessionLock: "x"}
	if !locks(rebound, altRunes("x"), "alt") {
		t.Error("session_lock = \"x\" should answer to alt+x")
	}
	if locks(rebound, altRunes("r"), "alt") {
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

// A background reload finding a broken profile must not kick the lock
// screen away — it queues the error and waits for a real unlock instead.
func TestApplyReloadedBundle_BrokenProfileStaysLockedUntilUnlock(t *testing.T) {
	m := lockTestModel(t)
	m.mode = lockedMode
	m.profile.acknowledged = map[string]bool{}

	broken := lair.ProfileParseError{Name: "work", Path: "/tmp/work.profile.toml", Err: "boom"}
	m = m.applyReloadedBundle(lair.ConfigBundle{Broken: []lair.ProfileParseError{broken}})

	if m.mode != lockedMode {
		t.Fatalf("a broken-profile reload must not leave lockedMode, got %v", m.mode)
	}
	if len(m.profile.pendingErrors) != 1 {
		t.Fatalf("the error should still be queued, got %d pending", len(m.profile.pendingErrors))
	}

	m = pumpUnlock(t, m)
	if m.mode != infoMode || m.activeDetail == nil {
		t.Fatalf("unlocking should present the queued error, mode=%v detail=%v", m.mode, m.activeDetail)
	}
}

// pumpUnlock drives a full alternating pump sequence through Update(), the
// same path a real unlock takes (so presentDeferredIfSafe runs too).
func pumpUnlock(t *testing.T, m Model) Model {
	t.Helper()
	if m.lock.pumpGoal <= 0 {
		m.lock.pumpGoal = defaultLockPumpGoal
	}
	for i := 0; i < m.lock.pumpGoal; i++ {
		key := "h"
		if i%2 == 1 {
			key = "l"
		}
		m = update(t, m, keyRunes(key))
	}
	return m
}

// Even a clean reload (nothing broken or dropped) must not reset the mode
// to gridMode while locked.
func TestApplyReloadedBundle_CleanReloadStaysLocked(t *testing.T) {
	m := lockTestModel(t)
	m.mode = lockedMode

	m = m.applyReloadedBundle(lair.ConfigBundle{})
	if m.mode != lockedMode {
		t.Fatalf("a clean reload must not leave lockedMode, got %v", m.mode)
	}
}

// capturingText (an inventory delete-confirmation, here) is the second
// protected mode — a background reload must defer to it too, with no
// explicit drain wiring anywhere near the confirmation's own esc/enter keys.
func TestApplyReloadedBundle_DefersWhileCapturingText(t *testing.T) {
	m := lockTestModel(t)
	m.mode = inventoryMode
	m.inventory.pending = &pendingDelete{rel: "x.profile.toml", name: "x"}
	m.profile.acknowledged = map[string]bool{}

	if !m.capturingText() {
		t.Fatal("fixture should be capturing text")
	}

	broken := lair.ProfileParseError{Name: "work", Path: "/tmp/work.profile.toml", Err: "boom"}
	m = m.applyReloadedBundle(lair.ConfigBundle{Broken: []lair.ProfileParseError{broken}})
	if m.mode != inventoryMode || m.inventory.pending == nil {
		t.Fatalf("a broken-profile reload must not interrupt the delete prompt, mode=%v pending=%v", m.mode, m.inventory.pending)
	}

	// Esc cancels the confirmation — an ordinary keypress, no knowledge of
	// the deferred error anywhere near this code path.
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != infoMode || m.activeDetail == nil {
		t.Fatalf("closing the prompt should present the deferred error, mode=%v detail=%v", m.mode, m.activeDetail)
	}
}

// The same race that breaks the lock screen can hit afterEdit: a queued
// lockCheckMsg resolving while $EDITOR was suspended, landing right before
// editorFinishedMsg. afterEdit must not bounce lockedMode to inventoryMode.
func TestAfterEditWhileLocked_StaysLocked(t *testing.T) {
	m := lockTestModel(t)
	m.mode = lockedMode

	next, _ := m.afterEdit(editorFinishedMsg{err: errors.New("editor failed to launch")})
	got := next.(Model)
	if got.mode != lockedMode {
		t.Fatalf("afterEdit must not leave lockedMode, got %v", got.mode)
	}
}

// The two locks live one modifier apart; they must not bleed into each other.
func TestSessionLock_BareLetterStillTogglesTheProfileLock(t *testing.T) {
	m := update(t, lockTestModel(t), keyRunes("r"))
	if m.mode == lockedMode {
		t.Fatal("bare r is the profile lock, not the session lock")
	}
}

// A reload while locked replaces the grid; unlocking must not return to a
// dropdown still listing the old folder's items.
func TestUnlockAfterReloadLeavesTheStaleDropdown(t *testing.T) {
	folder := func(item, cmd string) settings.Config {
		cfg := settings.Config{X: 1, Y: 1, Commands: []profile.Command{
			{Name: "tools", Col: "a", Items: []profile.CommandItem{{Name: item, Command: cmd}}},
		}}
		cfg.ApplyDefaults()
		return cfg
	}
	old := folder("status", "git status")
	m := Model{Config: old, styles: BuildStyles(old, ""), mode: gridMode, gridNav: gridNav{grid: profile.BuildGrid(old.X, old.Y, old.Commands)}}
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // open "tools"
	if m.mode != dropdownMode {
		t.Fatalf("setup: mode = %v, want dropdownMode", m.mode)
	}
	m = m.enterLockedMode()

	m = m.applyReloadedBundle(lair.ConfigBundle{Config: folder("wipe", "rm -rf ./build")})
	m = pumpUnlock(t, m)

	if m.mode == dropdownMode {
		t.Errorf("unlock reopened a dropdown listing %q from the replaced config", m.dropdown.items[0].Name)
	}
}

// Stashing the deck in use reloads twice: Apply's own reload, then the
// watcher's for the same file moves. The first says the deck is gone, by
// comparing before with after; the second has nothing to compare and says
// nothing, so the line stands.
func TestGoneLine_SaidOnceHoweverTheDeckWasReached(t *testing.T) {
	dir := freshConfigDir(t)
	deck := []byte("x = 1\ny = 1\n[[commands]]\nname = \"a\"\ncol = \"a\"\nrow = 0\ncommand = \"true\"\n")
	for _, name := range []string{"aaa", "zzz"} {
		if err := os.WriteFile(filepath.Join(dir, name+".profile.toml"), deck, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// On aaa only because it is first: nothing asked for it.
	m := InitialModel(setupFromEnv(false), Session{})
	if got := m.profile.deckName(); got != "aaa" {
		t.Fatalf("started on %q, want aaa", got)
	}
	if err := os.Rename(filepath.Join(dir, "aaa.profile.toml"), filepath.Join(dir, "inventory", "aaa.profile.toml")); err != nil {
		t.Fatal(err)
	}

	next, _ := m.Update(reloadProfilesMsg{})
	m = next.(Model)
	if m.mode != gridMode || !strings.HasPrefix(m.profile.statusMessage, "aaa is no longer equipped · now on ") {
		t.Fatalf("mode %v, status %q: want the grid and the gone line", m.mode, m.profile.statusMessage)
	}
	said := m.profile.statusMessage
	next, _ = m.Update(ConfigChangedMsg{Path: filepath.Join(dir, "pivot.toml")})
	if got := next.(Model).profile.statusMessage; got != said {
		t.Fatalf("the second reload changed the line to %q", got)
	}
}

// A deck that is pinned and then stashed: the line says the pin went too.
func TestGoneLine_NamesAClearedPin(t *testing.T) {
	bundle := lair.ConfigBundle{Profiles: []lair.ProfileInfo{{Name: "core"}}}
	if got, want := goneNote("work", "work", bundle), "work is no longer equipped · now on core · anchor lifted"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := goneNote("core", "", bundle); got != "" {
		t.Errorf("a deck still equipped got the line %q", got)
	}
	bundle.Broken = []lair.ProfileParseError{{Name: "work"}}
	if got := goneNote("work", "", bundle); got != "" {
		t.Errorf("a broken deck is not gone; its own notice speaks, got %q", got)
	}
}

// A cell's explain popup describes a grid the reload replaced, so it closes.
func TestExplainPopup_ClosesOnReload(t *testing.T) {
	m := lockTestModel(t)
	m.previousMode = gridMode
	m.activeDetail = newCommandDetail("a", "", "true", "", nil, "/")
	m.mode = infoMode

	m = m.applyReloadedBundle(lair.ConfigBundle{})
	if m.mode != gridMode {
		t.Fatalf("explain should close on a reload, mode=%v", m.mode)
	}
}

// The lock check locks the screen once the timeout passes without a key; not
// before, and never with auto-lock turned off. Either way it checks again.
func TestAutoLockAfterInactivity(t *testing.T) {
	off := false
	for _, c := range []struct {
		name    string
		idle    time.Duration
		enabled *bool
		want    bool
	}{
		{"idle past the timeout", 6 * time.Minute, nil, true},
		{"idle under the timeout", 4 * time.Minute, nil, false},
		{"auto-lock off", time.Hour, &off, false},
	} {
		m := lockTestModel(t)
		m.Config.AutoLockEnabled = c.enabled
		m.lock.timeoutMins = 5
		m.lock.lastActivity = time.Now().Add(-c.idle)
		next, cmd := m.Update(lockCheckMsg{})
		if locked := next.(Model).mode == lockedMode; locked != c.want {
			t.Errorf("%s: locked = %v, want %v", c.name, locked, c.want)
		}
		if cmd == nil {
			t.Errorf("%s: no next check scheduled", c.name)
		}
	}
}
