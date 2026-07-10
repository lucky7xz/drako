package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucky7xz/drako/internal/config"
)

func switchTestModel(t *testing.T) Model {
	t.Helper()
	dir := t.TempDir()
	var infos []config.ProfileInfo
	for _, name := range []string{"alpha", "beta"} {
		path := filepath.Join(dir, name+".profile.toml")
		if err := os.WriteFile(path, []byte("x = 1\ny = 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		infos = append(infos, config.ProfileInfo{Name: name, Path: path, Profile: config.ProfileFile{X: 1, Y: 1}})
	}
	base := config.Config{X: 1, Y: 1}
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
