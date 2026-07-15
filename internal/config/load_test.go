package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadTestDir builds a config dir with a valid config.toml and the named
// profiles, redirecting XDG_CONFIG_HOME/HOME there. Returns the drako dir.
func loadTestDir(t *testing.T, profileNames ...string) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("DRAKO_PROFILE", "")

	dir := filepath.Join(tmp, "drako")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("theme = \"default\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	profile := "x = 1\ny = 1\n[[commands]]\nname = \"noop\"\ncommand = \"true\"\nrow = 0\ncol = \"a\"\n"
	for _, name := range profileNames {
		if err := os.WriteFile(filepath.Join(dir, name+".profile.toml"), []byte(profile), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// mustLoadConfig runs LoadConfig and fails the test on an environment error,
// which no test environment should produce.
func mustLoadConfig(t *testing.T, override *string) ConfigBundle {
	t.Helper()
	bundle, err := LoadConfig(override)
	if err != nil {
		t.Fatal(err)
	}
	return bundle
}

// Base-config loading: settings map onto the base Config, and every failure
// path lands on a rescue config instead of killing the process.

func TestLoadBaseConfig_ValidSettings(t *testing.T) {
	dir := t.TempDir()
	content := "default_shell = \"fish\"\ntheme = \"default\"\nprofile = \"work\"\n"
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	base, broken := loadBaseConfig(dir)
	if len(broken) != 0 {
		t.Fatalf("unexpected broken entries: %+v", broken)
	}
	if base.DefaultShell != "fish" || base.Profile != "work" {
		t.Errorf("settings not mapped: shell=%q profile=%q", base.DefaultShell, base.Profile)
	}
	if base.Commands == nil || len(base.Commands) != 0 {
		t.Errorf("base commands must be empty (profiles own commands), got %v", base.Commands)
	}
}

func TestLoadBaseConfig_GarbageFallsBackToRescue(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("this is { not toml"), 0o644); err != nil {
		t.Fatal(err)
	}

	base, broken := loadBaseConfig(dir)
	if len(broken) != 1 || !strings.Contains(broken[0].Err, "SYNTAX ERROR") {
		t.Fatalf("want one SYNTAX ERROR entry, got %+v", broken)
	}
	if len(base.Commands) == 0 {
		t.Error("rescue config should carry its helper commands")
	}
}

func TestLoadBaseConfig_MissingFileWritesRescue(t *testing.T) {
	dir := t.TempDir()

	base, broken := loadBaseConfig(dir)
	if len(broken) != 0 {
		t.Fatalf("missing file is a fresh start, not an error: %+v", broken)
	}
	if len(base.Commands) == 0 {
		t.Error("rescue config should carry its helper commands")
	}
	if _, err := os.Stat(filepath.Join(dir, "config.toml")); err != nil {
		t.Errorf("rescue config should be written to disk: %v", err)
	}
}

func TestLoadBaseConfig_ExpandsEnvVars(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DRAKO_TEST_SHELL", "zsh")
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("default_shell = \"$DRAKO_TEST_SHELL\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	base, broken := loadBaseConfig(dir)
	if len(broken) != 0 {
		t.Fatalf("unexpected broken entries: %+v", broken)
	}
	if base.DefaultShell != "zsh" {
		t.Errorf("env vars must expand before decode, got shell=%q", base.DefaultShell)
	}
}

// Characterization pins for the selection behavior LoadConfig has today.

func TestLoadConfig_EnvVarSelectsProfile(t *testing.T) {
	loadTestDir(t, "alpha", "work")
	t.Setenv("DRAKO_PROFILE", "work")

	bundle := mustLoadConfig(t, nil)
	if got := bundle.Profiles[bundle.ActiveIndex].Name; got != "work" {
		t.Errorf("DRAKO_PROFILE should select work, got %q", got)
	}
}

func TestLoadConfig_OverrideBeatsEnvAndPivot(t *testing.T) {
	dir := loadTestDir(t, "alpha", "work")
	t.Setenv("DRAKO_PROFILE", "work")
	if err := WritePivotLocked(dir, "work"); err != nil {
		t.Fatal(err)
	}

	override := "alpha"
	bundle := mustLoadConfig(t, &override)
	if got := bundle.Profiles[bundle.ActiveIndex].Name; got != "alpha" {
		t.Errorf("override should beat env and pivot, got %q", got)
	}
}

func TestReloadConfig_SessionBeatsEnvAndCfg(t *testing.T) {
	loadTestDir(t, "alpha", "beta", "work")
	t.Setenv("DRAKO_PROFILE", "work")

	bundle, err := ReloadConfig("beta")
	if err != nil {
		t.Fatal(err)
	}
	if got := bundle.Profiles[bundle.ActiveIndex].Name; got != "beta" {
		t.Errorf("session profile should beat env and config.toml, got %q", got)
	}
}

func TestReloadConfig_PivotBeatsSession(t *testing.T) {
	dir := loadTestDir(t, "alpha", "work")
	if err := WritePivotLocked(dir, "work"); err != nil {
		t.Fatal(err)
	}

	bundle, err := ReloadConfig("alpha")
	if err != nil {
		t.Fatal(err)
	}
	if got := bundle.Profiles[bundle.ActiveIndex].Name; got != "work" {
		t.Errorf("pivot lock should snap back over the session profile, got %q", got)
	}
}

func TestReloadConfig_EmptySessionFallsBackToEnv(t *testing.T) {
	loadTestDir(t, "alpha", "work")
	t.Setenv("DRAKO_PROFILE", "work")

	bundle, err := ReloadConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if got := bundle.Profiles[bundle.ActiveIndex].Name; got != "work" {
		t.Errorf("no session yet: the env var handshake still applies, got %q", got)
	}
}

func TestLoadConfig_PivotOrderReorders(t *testing.T) {
	dir := loadTestDir(t, "alpha", "beta", "work")
	if err := WritePivotEquippedOrder(dir, []string{"work"}); err != nil {
		t.Fatal(err)
	}

	bundle := mustLoadConfig(t, nil)
	var got []string
	for _, p := range bundle.Profiles {
		got = append(got, p.Name)
	}
	want := []string{"work", "alpha", "beta"} // ordered first, leftovers name-sorted
	if len(got) != len(want) {
		t.Fatalf("profiles = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("profiles = %v, want %v", got, want)
		}
	}
}

// isRescueGrid reports whether cfg is the compiled-in rescue grid, identified
// by its distinctive "Exit Rescue Mode" command.
func isRescueGrid(cfg Config) bool {
	for _, c := range cfg.Commands {
		if c.Name == "Exit Rescue Mode" {
			return true
		}
	}
	return false
}

// Deleting the profile a pivot lock points at drops drako into the rescue grid.
// The bundle must name the vanished profile (DroppedProfile) so the UI can
// explain the drop instead of dumping the user into rescue silently, and the
// now-stale lock must be cleared.
func TestReloadConfig_DeletedLockedProfileReportsDropped(t *testing.T) {
	dir := loadTestDir(t, "work", "alpha")
	if err := WritePivotLocked(dir, "work"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "work.profile.toml")); err != nil {
		t.Fatal(err)
	}

	bundle, err := ReloadConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if bundle.DroppedProfile != "work" {
		t.Errorf("DroppedProfile = %q, want %q", bundle.DroppedProfile, "work")
	}
	if !isRescueGrid(bundle.Config) {
		t.Error("expected the rescue grid after the locked profile was deleted")
	}
	if bundle.LockedName != "" {
		t.Errorf("stale lock should be cleared, LockedName = %q", bundle.LockedName)
	}
}

// Moving the UNLOCKED *active* profile to inventory (session still names it)
// also drops to rescue — DroppedProfile must report it so the UI can explain.
func TestReloadConfig_UnlockedActiveGoneReportsDropped(t *testing.T) {
	dir := loadTestDir(t, "work", "alpha")
	// Session is on "work"; stash it (move out of the equipped dir).
	if err := os.Remove(filepath.Join(dir, "work.profile.toml")); err != nil {
		t.Fatal(err)
	}

	bundle, err := ReloadConfig("work")
	if err != nil {
		t.Fatal(err)
	}
	if bundle.DroppedProfile != "work" {
		t.Errorf("DroppedProfile = %q, want %q", bundle.DroppedProfile, "work")
	}
	if !isRescueGrid(bundle.Config) {
		t.Error("expected rescue after the active (session) profile was removed")
	}
}

// Removing a non-active profile while another remains, with no request pointing
// at it, falls back to the first available profile — no rescue, DroppedProfile
// stays empty.
func TestReloadConfig_GracefulFallbackNoDropped(t *testing.T) {
	dir := loadTestDir(t, "work", "alpha")
	if err := os.Remove(filepath.Join(dir, "work.profile.toml")); err != nil {
		t.Fatal(err)
	}

	bundle, err := ReloadConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if bundle.DroppedProfile != "" {
		t.Errorf("DroppedProfile = %q, want empty for a graceful fallback", bundle.DroppedProfile)
	}
	if isRescueGrid(bundle.Config) {
		t.Error("a remaining profile with no active request should not land in rescue")
	}
}
