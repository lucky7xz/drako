package lair

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucky7xz/drako/internal/equip"
)

func TestInstallPutsEachKindInItsPlace(t *testing.T) {
	_, dir := lairAt(t)

	if err := InstallProfile(dir, equip.Equipped, "a.profile.toml", strings.NewReader("a")); err != nil {
		t.Fatal(err)
	}
	if err := InstallProfile(dir, equip.Inventory, "b.profile.toml", strings.NewReader("b")); err != nil {
		t.Fatal(err)
	}
	if err := InstallSpec(dir, "dev.spec.toml", strings.NewReader("s")); err != nil {
		t.Fatal(err)
	}
	if err := InstallAsset(dir, "a", filepath.Join("scripts", "run.sh"), strings.NewReader("r")); err != nil {
		t.Fatal(err)
	}
	assertContent(t, filepath.Join(dir, "a.profile.toml"), "a")
	assertContent(t, filepath.Join(dir, "inventory", "b.profile.toml"), "b")
	assertContent(t, filepath.Join(dir, "specs", "dev.spec.toml"), "s")
	assertContent(t, filepath.Join(dir, "assets", "a", "scripts", "run.sh"), "r")

	// Installing again replaces, and leaves no temp file behind.
	if err := InstallProfile(dir, equip.Equipped, "a.profile.toml", strings.NewReader("a2")); err != nil {
		t.Fatal(err)
	}
	assertContent(t, filepath.Join(dir, "a.profile.toml"), "a2")
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

// The name handed over decides nothing about the folder: a name with a path
// in it is refused before the directory is even asked.
func TestInstallRefusesNamesThatAreNotPlain(t *testing.T) {
	_, dir := lairAt(t)
	outside := filepath.Join(filepath.Dir(dir), "escaped.txt")

	bad := []error{
		InstallProfile(dir, equip.Equipped, "../escaped.txt", strings.NewReader("x")),
		InstallProfile(dir, equip.Equipped, filepath.Join("inventory", "x.profile.toml"), strings.NewReader("x")),
		InstallProfile(dir, equip.Trash, "x.profile.toml", strings.NewReader("x")),
		InstallSpec(dir, "../escaped.txt", strings.NewReader("x")),
		InstallAsset(dir, "..", "x", strings.NewReader("x")),
		InstallAsset(dir, "a", filepath.Join("..", "..", "escaped.txt"), strings.NewReader("x")),
		InstallAsset(dir, "a", outside, strings.NewReader("x")),
	}
	for i, err := range bad {
		if err == nil {
			t.Errorf("case %d: accepted", i)
		}
	}
	assertMissing(t, outside)
}

// A summon cannot write through a linked inventory, specs or assets folder.
func TestInstallRefusesALinkedSubfolder(t *testing.T) {
	_, dir := lairAt(t)
	targets := map[string]string{}
	for _, name := range []string{"inventory", "specs", "assets"} {
		targets[name] = linkOut(t, dir, name, "", true)
	}

	errs := map[string]error{
		"inventory": InstallProfile(dir, equip.Inventory, "b.profile.toml", strings.NewReader("b")),
		"specs":     InstallSpec(dir, "dev.spec.toml", strings.NewReader("s")),
		"assets":    InstallAsset(dir, "a", "run.sh", strings.NewReader("r")),
	}
	for name, err := range errs {
		if !errors.Is(err, ErrLinkedFolder) || !strings.HasPrefix(err.Error(), name+" ") {
			t.Errorf("%s: %v, want ErrLinkedFolder naming it", name, err)
		}
		if left, _ := os.ReadDir(targets[name]); len(left) != 0 {
			t.Errorf("%s: written through the link: %v", name, left)
		}
	}
	if err := MakeInventory(dir); !errors.Is(err, ErrLinkedFolder) {
		t.Errorf("MakeInventory on a linked inventory: %v, want ErrLinkedFolder", err)
	}
}
