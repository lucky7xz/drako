package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestRestoreBootstrap_FullSetIntoEmptyDir(t *testing.T) {
	dir := t.TempDir()
	restored, err := RestoreBootstrap(dir)
	if err != nil {
		t.Fatal(err)
	}

	// The whole embedded set lands, reported by relative path.
	for _, want := range []string{
		"core.profile.toml",
		"inventory/ssh-101.profile.toml",
		"themes.toml",
		"config.toml",
		"specs/example.spec.toml",
	} {
		if !slices.Contains(restored, want) {
			t.Errorf("restored list missing %q; got %v", want, restored)
		}
		if _, err := os.Stat(filepath.Join(dir, want)); err != nil {
			t.Errorf("%s not written to disk: %v", want, err)
		}
	}
}

func TestRestoreBootstrap_Idempotent(t *testing.T) {
	dir := t.TempDir()
	if _, err := RestoreBootstrap(dir); err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBootstrap(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored) != 0 {
		t.Errorf("second run should restore nothing, got %v", restored)
	}
}

func TestRestoreBootstrap_PartialNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	if _, err := RestoreBootstrap(dir); err != nil {
		t.Fatal(err)
	}

	// Delete one file; tamper with another to prove it's never rewritten.
	sshDeck := filepath.Join(dir, "inventory", "ssh-101.profile.toml")
	if err := os.Remove(sshDeck); err != nil {
		t.Fatal(err)
	}
	corePath := filepath.Join(dir, "core.profile.toml")
	sentinel := []byte("# tampered\n")
	if err := os.WriteFile(corePath, sentinel, 0o644); err != nil {
		t.Fatal(err)
	}

	restored, err := RestoreBootstrap(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored) != 1 || restored[0] != "inventory/ssh-101.profile.toml" {
		t.Fatalf("expected only the deleted file restored, got %v", restored)
	}
	if _, err := os.Stat(sshDeck); err != nil {
		t.Errorf("deleted deck not restored: %v", err)
	}
	got, _ := os.ReadFile(corePath)
	if string(got) != string(sentinel) {
		t.Error("existing core.profile.toml was overwritten — must never happen")
	}
}

// An equipped inventory deck (moved to the config root) must not be duplicated
// back into inventory/ when a *different* deck is missing.
func TestRestoreBootstrap_EquippedDeckNotDuplicated(t *testing.T) {
	dir := t.TempDir()
	if _, err := RestoreBootstrap(dir); err != nil {
		t.Fatal(err)
	}

	// Equip ssh-101: inventory/ssh-101.profile.toml -> <root>/ssh-101.profile.toml.
	stashed := filepath.Join(dir, "inventory", "ssh-101.profile.toml")
	equipped := filepath.Join(dir, "ssh-101.profile.toml")
	if err := os.Rename(stashed, equipped); err != nil {
		t.Fatal(err)
	}
	// One other deck goes genuinely missing.
	missing := filepath.Join(dir, "inventory", "jukebox.profile.toml")
	if err := os.Remove(missing); err != nil {
		t.Fatal(err)
	}

	restored, err := RestoreBootstrap(dir)
	if err != nil {
		t.Fatal(err)
	}

	// The equipped deck is left alone: no duplicate in inventory/, original kept.
	if _, err := os.Stat(stashed); err == nil {
		t.Error("equipped ssh-101 was duplicated back into inventory/")
	}
	if _, err := os.Stat(equipped); err != nil {
		t.Errorf("equipped ssh-101 disappeared: %v", err)
	}
	// The genuinely-missing deck is restored to its canonical inventory/ spot.
	if _, err := os.Stat(missing); err != nil {
		t.Errorf("missing jukebox deck was not restored: %v", err)
	}
	if slices.Contains(restored, "inventory/ssh-101.profile.toml") {
		t.Errorf("restored list should not include the equipped deck; got %v", restored)
	}
	if !slices.Contains(restored, "inventory/jukebox.profile.toml") {
		t.Errorf("restored list missing the deck that was actually gone; got %v", restored)
	}
}

// A stashed core deck (moved to inventory/) must not be re-created at the root.
func TestRestoreCoreProfile_StashedNotDuplicated(t *testing.T) {
	dir := t.TempDir()
	if _, err := RestoreBootstrap(dir); err != nil {
		t.Fatal(err)
	}

	// Stash core: <root>/core.profile.toml -> inventory/core.profile.toml.
	root := filepath.Join(dir, "core.profile.toml")
	stashed := filepath.Join(dir, "inventory", "core.profile.toml")
	if err := os.MkdirAll(filepath.Dir(stashed), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(root, stashed); err != nil {
		t.Fatal(err)
	}

	created, err := RestoreCoreProfile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Error("stashed core was duplicated back at the config root")
	}
	if _, err := os.Stat(root); err == nil {
		t.Error("core.profile.toml re-created at root despite being stashed")
	}
}
