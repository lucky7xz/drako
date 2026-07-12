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
		"inventory/ssh-utils.profile.toml",
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
	sshUtils := filepath.Join(dir, "inventory", "ssh-utils.profile.toml")
	if err := os.Remove(sshUtils); err != nil {
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
	if len(restored) != 1 || restored[0] != "inventory/ssh-utils.profile.toml" {
		t.Fatalf("expected only the deleted file restored, got %v", restored)
	}
	if _, err := os.Stat(sshUtils); err != nil {
		t.Errorf("deleted deck not restored: %v", err)
	}
	got, _ := os.ReadFile(corePath)
	if string(got) != string(sentinel) {
		t.Error("existing core.profile.toml was overwritten — must never happen")
	}
}
