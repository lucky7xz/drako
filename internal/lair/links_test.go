package lair

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/lucky7xz/drako/internal/equip"
)

const linkedDeck = "x = 1\ny = 1\n[[commands]]\nname = \"a\"\ncol = \"A\"\nrow = 0\ncommand = \"true\"\n"

// linkOut makes name inside dir a link to a fresh file or folder outside it,
// the way stow links things out of a dotfiles repository, and returns where
// it points.
func linkOut(t *testing.T, dir, name, content string, folder bool) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("creating links needs privileges on Windows")
	}
	outside := filepath.Join(t.TempDir(), filepath.Base(name))
	if folder {
		if err := os.Mkdir(outside, 0o755); err != nil {
			t.Fatal(err)
		}
	} else {
		writeFile(t, outside, content)
	}
	if err := os.Symlink(outside, filepath.Join(dir, name)); err != nil {
		t.Fatal(err)
	}
	return outside
}

// A file the user linked into the folder is read: settings, themes and
// profiles may live in a dotfiles repository.
func TestLinkedFilesAreRead(t *testing.T) {
	l, dir := lairAt(t)
	linkOut(t, dir, "config.toml", "theme = \"nord\"\n", false)
	linkOut(t, dir, "work.profile.toml", linkedDeck, false)

	data, err := l.read("config.toml")
	if err != nil || string(data) != "theme = \"nord\"\n" {
		t.Errorf("linked config.toml: %q, %v", data, err)
	}
	if _, err := l.stat("config.toml"); err != nil {
		t.Errorf("a linked file must count as present: %v", err)
	}
	found, broken := l.discoverProfiles([]string{"linux_generic"})
	if len(found) != 1 || found[0].Name != "work" || len(broken) != 0 {
		t.Errorf("linked profile not loaded: %+v, broken %+v", found, broken)
	}
}

// Reading follows a linked file and nothing else: not "..", and not a path
// that goes through a linked subfolder.
func TestReadsDoNotLeaveAnyOtherWay(t *testing.T) {
	l, dir := lairAt(t)
	writeFile(t, filepath.Join(filepath.Dir(dir), "secret.txt"), "secret")
	if data, err := l.read("../secret.txt"); err == nil {
		t.Errorf("read stepped out with ..: %q", data)
	}

	outside := linkOut(t, dir, "inventory", "", true)
	writeFile(t, filepath.Join(outside, "x.profile.toml"), linkedDeck)
	_, err := l.read(filepath.Join("inventory", "x.profile.toml"))
	if !errors.Is(err, ErrLinkedFolder) || !strings.HasPrefix(err.Error(), "inventory ") {
		t.Errorf("read through a linked subfolder: %v, want ErrLinkedFolder naming it", err)
	}
}

// Writing replaces a linked pivot.toml with a file of drako's own; what the
// link pointed at stays as it was.
func TestWritingNeverGoesThroughALink(t *testing.T) {
	_, dir := lairAt(t)
	outside := linkOut(t, dir, "pivot.toml", "locked = \"old\"\n", false)

	if err := WritePivotLocked(dir, "work"); err != nil {
		t.Fatal(err)
	}
	assertContent(t, outside, "locked = \"old\"\n")
	if info, err := os.Lstat(filepath.Join(dir, "pivot.toml")); err != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Errorf("pivot.toml should now be a plain file in the folder: %v", err)
	}
	pf, err := ReadPivotProfile(dir)
	if err != nil || pf.Locked != "work" {
		t.Errorf("pivot after write: %+v, %v", pf, err)
	}
	// No temp file left behind.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

// A linked subfolder is not written through: bootstrap restores everything
// else, says which folder it skipped, and leaves the link's target empty.
func TestBootstrapSkipsALinkedSubfolder(t *testing.T) {
	_, dir := lairAt(t)
	outside := linkOut(t, dir, "inventory", "", true)

	restored, err := RestoreBootstrap(dir)
	if !errors.Is(err, ErrLinkedFolder) {
		t.Fatalf("err = %v, want ErrLinkedFolder", err)
	}
	if !slices.Contains(restored, "config.toml") || !slices.Contains(restored, "core.profile.toml") || !slices.Contains(restored, "themes.toml") {
		t.Errorf("the rest should still be restored, got %v", restored)
	}
	for _, r := range restored {
		if strings.HasPrefix(r, "inventory/") {
			t.Errorf("restored %s through the link", r)
		}
	}
	if left, _ := os.ReadDir(outside); len(left) != 0 {
		t.Errorf("the link's target was written to: %v", left)
	}
}

// Stashing and trashing write into a subfolder. When that subfolder is a
// link leaving the directory, nothing moves and the error names it.
func TestMovesRefuseALinkedSubfolder(t *testing.T) {
	_, dir := lairAt(t)
	writeFile(t, filepath.Join(dir, "work.profile.toml"), linkedDeck)
	inventory := linkOut(t, dir, "inventory", "", true)
	trash := linkOut(t, dir, "trash", "", true)

	err := Move(dir, "work.profile.toml", equip.Equipped, equip.Inventory)
	if !errors.Is(err, ErrLinkedFolder) || !strings.HasPrefix(err.Error(), "inventory ") {
		t.Errorf("stash into a linked inventory: %v, want ErrLinkedFolder naming it", err)
	}
	err = MoveToTrash(dir, "work.profile.toml")
	if !errors.Is(err, ErrLinkedFolder) || !strings.HasPrefix(err.Error(), "trash ") {
		t.Errorf("trash into a linked trash: %v, want ErrLinkedFolder naming it", err)
	}

	assertContent(t, filepath.Join(dir, "work.profile.toml"), linkedDeck) // still equipped
	for _, target := range []string{inventory, trash} {
		if left, _ := os.ReadDir(target); len(left) != 0 {
			t.Errorf("%s was written to through the link: %v", target, left)
		}
	}
	if _, err := List(dir, equip.Inventory); !errors.Is(err, ErrLinkedFolder) {
		t.Errorf("listing a linked inventory: %v, want ErrLinkedFolder", err)
	}
}

// The layout check names every subfolder that is a link leaving drako's
// folder; a real folder and a link that stays inside are fine.
func TestCheckLayout(t *testing.T) {
	_, dir := lairAt(t)
	if err := CheckLayout(dir); err != nil {
		t.Fatalf("plain folder: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "inv-real"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("inv-real", filepath.Join(dir, "inventory")); err != nil {
		t.Fatal(err)
	}
	if err := CheckLayout(dir); err != nil {
		t.Fatalf("a link that stays inside: %v", err)
	}

	linkOut(t, dir, "specs", "", true)
	err := CheckLayout(dir)
	if err == nil || !strings.HasPrefix(err.Error(), "specs is a linked folder, and drako does not support linked folders.\n") {
		t.Fatalf("one linked folder: %v", err)
	}
	linkOut(t, dir, "trash", "", true)
	err = CheckLayout(dir)
	if err == nil || !strings.HasPrefix(err.Error(), "specs and trash are linked folders") {
		t.Fatalf("two linked folders: %v", err)
	}
	if !strings.Contains(err.Error(), `See "Linking Your Config" in the README.`) {
		t.Errorf("no pointer to the README: %v", err)
	}
}

// A folder that does not exist yet (first run) has nothing to check.
func TestCheckLayoutMissingFolder(t *testing.T) {
	if err := CheckLayout(filepath.Join(t.TempDir(), "drako")); err != nil {
		t.Errorf("missing folder: %v", err)
	}
}
