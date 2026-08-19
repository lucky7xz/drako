package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/lucky7xz/drako/internal/profiles"
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

	if err := ApplySpec(cfg, []string{"work"}, false); err != nil {
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

// A deck may legitimately list more profiles than the leader chords address,
// but only the confirmed path is allowed to apply it.
func TestApplySpec_OverCapNeedsConsent(t *testing.T) {
	names := make([]string, profiles.MaxEquipped+3)
	for i := range names {
		names[i] = fmt.Sprintf("deck%02d", i)
	}

	t.Run("refused without consent", func(t *testing.T) {
		cfg := t.TempDir()
		inv := filepath.Join(cfg, "inventory")
		for _, n := range names {
			writeProfile(t, inv, n)
		}

		if err := ApplySpec(cfg, names, false); err == nil {
			t.Fatal("expected an over-cap spec to be refused")
		}
		for _, n := range names {
			if exists(t, cfg, n+".profile.toml") {
				t.Fatalf("a refused spec must move nothing, but %s was equipped", n)
			}
		}
	})

	t.Run("applied with consent", func(t *testing.T) {
		cfg := t.TempDir()
		inv := filepath.Join(cfg, "inventory")
		for _, n := range names {
			writeProfile(t, inv, n)
		}

		if err := ApplySpec(cfg, names, true); err != nil {
			t.Fatalf("ApplySpec over cap: %v", err)
		}
		for _, n := range names {
			if !exists(t, cfg, n+".profile.toml") {
				t.Errorf("%s must be equipped", n)
			}
		}
	})
}
