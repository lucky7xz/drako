package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// helpers ---------------------------------------------------------------

func writeProfile(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".profile.toml"), []byte("x=1\ny=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func exists(t *testing.T, parts ...string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(parts...))
	return err == nil
}

// StripAllProfiles ------------------------------------------------------

func TestStripAllProfiles_MovesEverything(t *testing.T) {
	cfg := t.TempDir()
	inv := filepath.Join(cfg, "inventory")
	writeProfile(t, cfg, "core")
	writeProfile(t, cfg, "git")
	writeProfile(t, cfg, "work")

	if err := StripAllProfiles(cfg); err != nil {
		t.Fatalf("StripAllProfiles: %v", err)
	}

	if exists(t, cfg, "core.profile.toml") || exists(t, cfg, "git.profile.toml") || exists(t, cfg, "work.profile.toml") {
		t.Error("all profiles must leave the equipped dir")
	}
	if !exists(t, inv, "core.profile.toml") || !exists(t, inv, "git.profile.toml") || !exists(t, inv, "work.profile.toml") {
		t.Error("all profiles must land in inventory")
	}
}

// StashSpec -------------------------------------------------------------

func TestStashSpec_MovesOnlyTargets(t *testing.T) {
	cfg := t.TempDir()
	inv := filepath.Join(cfg, "inventory")
	writeProfile(t, cfg, "core")
	writeProfile(t, cfg, "git")
	writeProfile(t, cfg, "work")

	if err := StashSpec(cfg, []string{"git"}); err != nil {
		t.Fatalf("StashSpec: %v", err)
	}

	if !exists(t, inv, "git.profile.toml") {
		t.Error("git must be stashed to inventory")
	}
	if !exists(t, cfg, "work.profile.toml") || !exists(t, cfg, "core.profile.toml") {
		t.Error("non-targets must stay equipped")
	}
}

// ApplySpec -------------------------------------------------------------

func TestApplySpec_EquipsTargetsStashesRest(t *testing.T) {
	cfg := t.TempDir()
	inv := filepath.Join(cfg, "inventory")
	writeProfile(t, cfg, "core")
	writeProfile(t, cfg, "git")  // equipped, not in target -> should stash
	writeProfile(t, inv, "work") // inventory, in target -> should equip

	if err := ApplySpec(cfg, []string{"work"}); err != nil {
		t.Fatalf("ApplySpec: %v", err)
	}

	if !exists(t, cfg, "work.profile.toml") {
		t.Error("work must be equipped from inventory")
	}
	if !exists(t, inv, "git.profile.toml") {
		t.Error("git must be stashed (not in target)")
	}
	if !exists(t, inv, "core.profile.toml") {
		t.Error("core must be stashed like any other deck (not in target)")
	}
}
