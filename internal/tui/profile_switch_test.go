package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/lair"
	"github.com/lucky7xz/drako/internal/profile"
	"github.com/lucky7xz/drako/internal/settings"
)

func switchTestModel(t *testing.T) Model {
	t.Helper()
	dir := t.TempDir()
	var infos []lair.ProfileInfo
	for _, name := range []string{"alpha", "beta"} {
		path := filepath.Join(dir, name+".profile.toml")
		if err := os.WriteFile(path, []byte("x = 1\ny = 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		infos = append(infos, lair.ProfileInfo{Name: name, Path: path, Profile: profile.File{X: 1, Y: 1}})
	}
	base := settings.Config{X: 1, Y: 1}
	base.ApplyDefaults()
	return Model{Config: base, profile: profileState{base: base, profiles: infos}}
}

func TestSwitchProfileTracksSessionWithoutEnvMutation(t *testing.T) {
	t.Setenv("DRAKO_PROFILE", "sentinel")
	m := switchTestModel(t)

	updated, ok := m.switchToProfileIndex(1)
	if !ok {
		t.Fatal("switch to beta failed")
	}
	if got := updated.ActiveProfileName(); got != "beta" {
		t.Errorf("ActiveProfileName = %q, want beta", got)
	}
	if updated.profile.sessionProfile != "beta" {
		t.Errorf("sessionProfile = %q, want beta (reload handshake)", updated.profile.sessionProfile)
	}
	if os.Getenv("DRAKO_PROFILE") != "sentinel" {
		t.Error("switching profiles must not mutate the process environment")
	}
}

func TestActiveProfileName_NoProfiles(t *testing.T) {
	var m Model
	if got := m.ActiveProfileName(); got != "" {
		t.Errorf("ActiveProfileName with no profiles = %q, want empty", got)
	}
}

// leader+digit past the last profile is a no-op, like modifier+digit;
// switchToProfileIndex's wraparound is for cycling only.
func TestLeaderDigitPastLastProfileIsNoop(t *testing.T) {
	m := switchTestModel(t)
	m.profile.activeIndex = 1

	next, _ := m.handleLeaderContinuation(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})

	if got := next.(Model).ActiveProfileName(); got != "beta" {
		t.Errorf("leader+3 with 2 profiles switched to %q, want to stay on beta", got)
	}
}

// o and p cycle through the equipped profiles, wrapping at both ends and
// skipping a profile whose file is gone.
func TestProfileCycleWrapsAndSkipsMissing(t *testing.T) {
	m := switchTestModel(t) // alpha, beta
	gamma := filepath.Join(t.TempDir(), "gamma.profile.toml")
	if err := os.WriteFile(gamma, []byte("x = 1\ny = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.profile.profiles = append(m.profile.profiles, lair.ProfileInfo{Name: "gamma", Path: gamma, Profile: profile.File{X: 1, Y: 1}})
	m.mode = gridMode
	m.styles = BuildStyles(m.Config, "")

	for _, step := range []struct{ key, want string }{
		{"p", "beta"}, {"p", "gamma"}, {"p", "alpha"}, // past the last: the first
		{"o", "gamma"}, // before the first: the last
	} {
		m = update(t, m, keyRunes(step.key))
		if got := m.ActiveProfileName(); got != step.want {
			t.Fatalf("%s: on %q, want %q", step.key, got, step.want)
		}
	}

	if err := os.Remove(m.profile.profiles[1].Path); err != nil { // beta's file
		t.Fatal(err)
	}
	m = update(t, m, keyRunes("p")) // gamma → alpha
	m = update(t, m, keyRunes("p")) // alpha → gamma: beta is skipped
	if got := m.ActiveProfileName(); got != "gamma" {
		t.Errorf("p past a missing profile: on %q, want gamma", got)
	}
}
