package lair

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinkTarget(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "drako")
	dots := filepath.Join(base, "dots")
	for _, d := range []string{filepath.Join(dir, "inventory"), dots} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	orig := filepath.Join(dots, "work.profile.toml")
	for _, f := range []string{orig, filepath.Join(dir, "plain.profile.toml")} {
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	links := map[string]string{
		"rel.profile.toml":           "../dots/work.profile.toml", // as stow writes it
		"abs.profile.toml":           orig,
		"inventory/inv.profile.toml": "../../dots/work.profile.toml",
	}
	for name, target := range links {
		if err := os.Symlink(target, filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	for name := range links {
		if got := LinkTarget(dir, name); got != orig {
			t.Errorf("LinkTarget(%s) = %q, want %q", name, got, orig)
		}
	}
	for _, name := range []string{"plain.profile.toml", "gone.profile.toml", "../dots/work.profile.toml"} {
		if got := LinkTarget(dir, name); got != "" {
			t.Errorf("LinkTarget(%s) = %q, want none", name, got)
		}
	}
	if got := CountLinks(dir); got != 3 {
		t.Errorf("CountLinks = %d, want 3", got)
	}

	if got := DirLinkTarget(dir); got != "" {
		t.Errorf("an ordinary folder: DirLinkTarget = %q, want none", got)
	}
	linked := filepath.Join(base, "linked")
	if err := os.Symlink("drako", linked); err != nil {
		t.Fatal(err)
	}
	if got := DirLinkTarget(linked); got != dir {
		t.Errorf("DirLinkTarget = %q, want %q", got, dir)
	}
}
