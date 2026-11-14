package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// lapConfig points drako at a fresh config dir holding one good and one
// broken profile.
func lapConfig(t *testing.T) {
	t.Helper()
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	dir := filepath.Join(xdg, "drako")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	good := "x = 1\ny = 1\n[[commands]]\nname = \"hi\"\ncommand = \"echo hi\"\nrow = 0\ncol = \"a\"\n"
	if err := os.WriteFile(filepath.Join(dir, "good.profile.toml"), []byte(good), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "broken.profile.toml"), []byte("x = [oops\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// An error the user dismissed stays dismissed when a command run relaunches
// the TUI, instead of popping up again (and dropping into rescue) every lap.
func TestDismissedErrorStaysDismissedAcrossLaps(t *testing.T) {
	lapConfig(t)

	first := InitialModel(setupFromEnv(false), Session{})
	if first.mode != infoMode {
		t.Fatalf("setup: lap 1 should show the broken profile, mode = %v", first.mode)
	}
	next, _ := first.Update(tea.KeyMsg{Type: tea.KeyEnter}) // dismiss
	session := next.(Model).Session()

	second := InitialModel(setupFromEnv(false), session)
	if second.mode != gridMode {
		t.Errorf("lap 2 re-showed the dismissed error: mode = %v, title %q", second.mode, second.activeDetail.Title)
	}
}

// A profile that broke again after being fixed is news: its error shows again.
func TestRebrokenProfileShowsAgainNextLap(t *testing.T) {
	lapConfig(t)
	carried := Session{acknowledged: map[string]bool{"/elsewhere/old.profile.toml": true}}

	m := InitialModel(setupFromEnv(false), carried)
	if m.mode != infoMode {
		t.Errorf("an error not acknowledged before must show, mode = %v", m.mode)
	}
	if m.profile.acknowledged["/elsewhere/old.profile.toml"] {
		t.Error("an acknowledgment for a file no longer broken must be dropped")
	}
}

// twoDecks: equipped profiles alpha and work, nothing broken.
func twoDecks(t *testing.T) string {
	t.Helper()
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	dir := filepath.Join(xdg, "drako")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"alpha", "work"} {
		deck := "x = 1\ny = 1\n[[commands]]\nname = \"" + name + "-cell\"\ncommand = \"true\"\nrow = 0\ncol = \"a\"\n"
		if err := os.WriteFile(filepath.Join(dir, name+".profile.toml"), []byte(deck), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// Stashing the session's profile says so once, accurately: drako moved to
// another equipped deck. Later reloads must not say it again.
func TestGoneSessionProfileIsSaidOnce(t *testing.T) {
	dir := twoDecks(t)
	m := InitialModel(setupFromEnv(false), Session{Profile: "work"})
	if m.ActiveProfileName() != "work" {
		t.Fatalf("setup: active = %q, want work", m.ActiveProfileName())
	}
	m.profile.sessionProfile = "work" // as an explicit switch leaves it
	if err := os.Rename(filepath.Join(dir, "work.profile.toml"), filepath.Join(dir, "inventory", "work.profile.toml")); err != nil {
		t.Fatal(err)
	}

	next, _ := m.Update(ConfigChangedMsg{})
	m = next.(Model)
	if got, want := m.profile.statusMessage, "work is no longer equipped · now on alpha"; m.mode != gridMode || got != want {
		t.Fatalf("mode %v, status %q; want the grid and %q", m.mode, got, want)
	}

	m.profile.statusMessage = ""           // its time is up
	next, _ = m.Update(ConfigChangedMsg{}) // any later change
	if m = next.(Model); m.profile.statusMessage != "" {
		t.Errorf("a later reload said it again: %q", m.profile.statusMessage)
	}
}

// config.toml can name a deck that was later stashed; nothing rewrites it, so
// every load requests it again. A fresh start says so; a reload and a lap of
// the same session, which were never on that deck, do not.
func TestUnequippedConfigProfileIsSaidAtTheStart(t *testing.T) {
	dir := twoDecks(t)
	first := InitialModel(setupFromEnv(false), Session{}) // bootstraps config.toml
	_ = first
	cfgPath := filepath.Join(dir, "config.toml")
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, append([]byte("profile = \"work\"\n"), raw...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "work.profile.toml"), filepath.Join(dir, "inventory", "work.profile.toml")); err != nil {
		t.Fatal(err)
	}

	m := InitialModel(setupFromEnv(false), Session{})
	if got, want := m.profile.statusMessage, "work is not equipped · now on alpha"; m.mode != gridMode || got != want {
		t.Fatalf("mode %v, status %q; want the grid and %q", m.mode, got, want)
	}
	m.profile.statusMessage = "" // its time is up
	next, _ := m.Update(ConfigChangedMsg{})
	if m = next.(Model); m.profile.statusMessage != "" {
		t.Errorf("a reload said it again: %q", m.profile.statusMessage)
	}
	if lap := InitialModel(setupFromEnv(false), m.Session()); lap.profile.statusMessage != "" {
		t.Errorf("the next lap said it again: %q", lap.profile.statusMessage)
	}
}

// afterEdit applies the same glassroot policy as every other reload: zero
// equipped profiles would expose the rescue grid, so the session ends instead.
func TestAfterEditInGlassrootRejectsAnEmptyBundle(t *testing.T) {
	dir := twoDecks(t)
	m := InitialModel(setupFromEnv(true), Session{})
	if m.quitting {
		t.Fatal("setup: two good decks should start a glassroot session")
	}
	equipped, _ := filepath.Glob(filepath.Join(dir, "*.profile.toml"))
	for _, path := range equipped { // bootstrap added core next to alpha and work
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	}

	next, _ := m.afterEdit(editorFinishedMsg{path: filepath.Join(dir, "alpha.profile.toml")})
	if got := next.(Model); !got.quitting || got.exitCode != 1 {
		t.Errorf("glassroot kept running on the rescue grid (quitting=%v, first cell %q)", got.quitting, got.Config.Commands[0].Name)
	}
}
