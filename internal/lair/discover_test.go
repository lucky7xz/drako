package lair

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckProfileFile(t *testing.T) {
	dir := t.TempDir()

	good := filepath.Join(dir, "g.profile.toml")
	os.WriteFile(good, []byte("x=1\ny=1\n[[commands]]\nname=\"a\"\ncol=\"A\"\nrow=0\n"), 0o644)
	if err := CheckProfileFile(dir, "g.profile.toml"); err != nil {
		t.Errorf("valid file rejected: %v", err)
	}

	bad := filepath.Join(dir, "b.profile.toml")
	os.WriteFile(bad, []byte("not [ toml"), 0o644)
	if err := CheckProfileFile(dir, "b.profile.toml"); err == nil {
		t.Error("invalid TOML accepted")
	}

	empty := filepath.Join(dir, "e.profile.toml")
	os.WriteFile(empty, []byte(""), 0o644)
	if err := CheckProfileFile(dir, "e.profile.toml"); err == nil {
		t.Error("empty profile accepted")
	}

	if err := CheckProfileFile(dir, "missing.profile.toml"); err == nil {
		t.Error("missing file accepted")
	}
}
