package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSpec(t *testing.T, dir, file, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, file), []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", file, err)
	}
}

func TestLoadSpec_ReadsProfiles(t *testing.T) {
	dir := t.TempDir()
	writeSpec(t, dir, "dev.spec.toml", "profiles = [\"git\", \"rust\"]\n")

	got, err := LoadSpec(filepath.Join(dir, "dev.spec.toml"))
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if len(got) != 2 || got[0] != "git" || got[1] != "rust" {
		t.Errorf("got %v, want [git rust]", got)
	}
}

func TestLoadSpec_MalformedTOML(t *testing.T) {
	dir := t.TempDir()
	writeSpec(t, dir, "bad.spec.toml", "profiles = [\"git\"\n")

	if _, err := LoadSpec(filepath.Join(dir, "bad.spec.toml")); err == nil {
		t.Fatal("malformed TOML should be an error")
	}
}

func TestLoadSpec_Missing(t *testing.T) {
	if _, err := LoadSpec(filepath.Join(t.TempDir(), "nope.spec.toml")); err == nil {
		t.Fatal("a missing spec should be an error")
	}
}

func TestLoadSpec_EmptyProfilesIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	writeSpec(t, dir, "empty.spec.toml", "profiles = []\n")

	got, err := LoadSpec(filepath.Join(dir, "empty.spec.toml"))
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

// One unreadable spec must not cost the caller every other spec in the
// directory — the same contract DiscoverProfilesWithErrors keeps.
func TestDiscoverSpecs_CarriesPerFileErrors(t *testing.T) {
	dir := t.TempDir()
	writeSpec(t, dir, "dev.spec.toml", "profiles = [\"git\", \"rust\"]\n")
	writeSpec(t, dir, "bad.spec.toml", "profiles = [\"git\"\n")
	writeSpec(t, dir, "notaspec.txt", "ignored\n")

	specs, err := DiscoverSpecs(dir)
	if err != nil {
		t.Fatalf("DiscoverSpecs: %v", err)
	}
	if len(specs) != 2 {
		t.Fatalf("got %d specs, want 2: %+v", len(specs), specs)
	}

	// Sorted by name: bad, dev.
	if specs[0].Name != "bad" || specs[0].File != "bad.spec.toml" {
		t.Errorf("first spec = %+v", specs[0])
	}
	if specs[0].Err == nil {
		t.Error("malformed spec should carry an Err")
	}
	if specs[1].Name != "dev" || specs[1].Err != nil {
		t.Errorf("second spec = %+v", specs[1])
	}
	if strings.Join(specs[1].Profiles, ",") != "git,rust" {
		t.Errorf("profiles = %v", specs[1].Profiles)
	}
}

func TestDiscoverSpecs_MissingDirIsEmpty(t *testing.T) {
	specs, err := DiscoverSpecs(filepath.Join(t.TempDir(), "absent"))
	if err != nil {
		t.Fatalf("a missing specs dir should not be an error: %v", err)
	}
	if len(specs) != 0 {
		t.Errorf("got %d specs, want 0", len(specs))
	}
}
