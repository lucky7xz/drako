package ui

import (
	"os"
	"path/filepath"
	"strings"
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

	first := InitialModel(false, Session{})
	if first.mode != infoMode {
		t.Fatalf("setup: lap 1 should show the broken profile, mode = %v", first.mode)
	}
	next, _ := first.Update(tea.KeyMsg{Type: tea.KeyEnter}) // dismiss
	session := next.(Model).Session()

	second := InitialModel(false, session)
	if second.mode != gridMode {
		t.Errorf("lap 2 re-showed the dismissed error: mode = %v, title %q", second.mode, second.activeDetail.Title)
	}
}

// A profile that broke again after being fixed is news: its error shows again.
func TestRebrokenProfileShowsAgainNextLap(t *testing.T) {
	lapConfig(t)
	carried := Session{acknowledged: map[string]bool{"/elsewhere/old.profile.toml": true}}

	m := InitialModel(false, carried)
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

// Stashing the session's profile explains itself once, accurately: drako moved
// to another equipped deck. Later reloads must not repeat the note.
func TestDroppedSessionProfileIsExplainedOnce(t *testing.T) {
	dir := twoDecks(t)
	m := InitialModel(false, Session{Profile: "work"})
	if m.ActiveProfileName() != "work" {
		t.Fatalf("setup: active = %q, want work", m.ActiveProfileName())
	}
	m.profile.sessionProfile = "work" // as an explicit switch leaves it
	if err := os.Rename(filepath.Join(dir, "work.profile.toml"), filepath.Join(dir, "inventory", "work.profile.toml")); err != nil {
		t.Fatal(err)
	}

	next, _ := m.Update(ConfigChangedMsg{})
	m = next.(Model)
	if m.mode != infoMode || m.activeDetail == nil {
		t.Fatalf("the first reload should explain the drop, mode = %v", m.mode)
	}
	if d := m.activeDetail.Description; strings.Contains(d, "Rescue Mode") || !strings.Contains(d, "switched to alpha") {
		t.Errorf("note should say drako switched to alpha, says: %q", d)
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // dismiss
	next, _ = next.(Model).Update(ConfigChangedMsg{})  // any later change
	if m = next.(Model); m.mode != gridMode {
		t.Errorf("a later reload repeated the note: mode = %v, title %q", m.mode, m.activeDetail.Title)
	}
}

// config.toml can name a deck that was later stashed; nothing rewrites it, so
// every reload and every lap requests it again. One explanation is enough.
func TestDroppedConfigProfileIsExplainedOnce(t *testing.T) {
	dir := twoDecks(t)
	first := InitialModel(false, Session{}) // bootstraps config.toml
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

	m := InitialModel(false, Session{})
	if m.mode != infoMode {
		t.Fatalf("setup: the drop should be explained once, mode = %v", m.mode)
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next, _ = next.(Model).Update(ConfigChangedMsg{})
	if m = next.(Model); m.mode != gridMode {
		t.Errorf("a reload repeated the note, mode = %v", m.mode)
	}
	lap := InitialModel(false, m.Session())
	if lap.mode != gridMode {
		t.Errorf("the next lap repeated the note, mode = %v", lap.mode)
	}
}

// afterEdit applies the same glassroot policy as every other reload: zero
// equipped profiles would expose the rescue grid, so the session ends instead.
func TestAfterEditInGlassrootRejectsAnEmptyBundle(t *testing.T) {
	dir := twoDecks(t)
	m := InitialModel(true, Session{})
	if m.Quitting {
		t.Fatal("setup: two good decks should start a glassroot session")
	}
	equipped, _ := filepath.Glob(filepath.Join(dir, "*.profile.toml"))
	for _, path := range equipped { // bootstrap added core next to alpha and work
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	}

	next, _ := m.afterEdit(editorFinishedMsg{path: filepath.Join(dir, "alpha.profile.toml")})
	if got := next.(Model); !got.Quitting || got.ExitCode != 1 {
		t.Errorf("glassroot kept running on the rescue grid (quitting=%v, first cell %q)", got.Quitting, got.Config.Commands[0].Name)
	}
}

// Two broken decks land on rescue; a lap keeps rescue instead of switching back
// into the deck the load just rejected.
func TestRescueSurvivesALap(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("DRAKO_PROFILE", "beta")
	dir := filepath.Join(xdg, "drako")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"alpha", "beta"} {
		deck := "x = 1\ny = 1\n[[commands]]\nname = \"far\"\ncommand = \"echo\"\nrow = 5\ncol = \"a\"\n"
		if err := os.WriteFile(filepath.Join(dir, n+".profile.toml"), []byte(deck), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	first := InitialModel(false, Session{})
	second := InitialModel(false, first.Session())
	for lap, m := range []Model{first, second} {
		if !m.profile.rescue || m.ActiveProfileName() != "" {
			t.Errorf("lap %d: want rescue with no active deck, got rescue=%v active=%q", lap+1, m.profile.rescue, m.ActiveProfileName())
		}
	}
}
