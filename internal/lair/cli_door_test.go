package lair

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// DeleteLogs removes the logs that are there, names them, and removes a
// linked log's link, never its target.
func TestDeleteLogs(t *testing.T) {
	_, dir := lairAt(t)
	writeFile(t, filepath.Join(dir, LogFileName), "log")
	outside := linkOut(t, dir, HistoryFileName, "history", false)
	deleted, err := DeleteLogs(dir)
	if err != nil || !slices.Equal(deleted, []string{LogFileName, HistoryFileName}) {
		t.Fatalf("DeleteLogs = %v, %v", deleted, err)
	}
	for _, n := range []string{LogFileName, HistoryFileName} {
		if _, err := os.Lstat(filepath.Join(dir, n)); err == nil {
			t.Errorf("%s still there", n)
		}
	}
	assertContent(t, outside, "history")
	if deleted, err := DeleteLogs(filepath.Join(dir, "missing")); err != nil || deleted != nil {
		t.Errorf("missing folder: %v, %v", deleted, err)
	}
}

// Destroy removes the folder; a missing one is an error that says so.
func TestDestroy(t *testing.T) {
	_, dir := lairAt(t)
	writeFile(t, filepath.Join(dir, "x.profile.toml"), "x")
	if err := Destroy(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(dir); !os.IsNotExist(err) {
		t.Errorf("folder still there: %v", err)
	}
	if err := Destroy(dir); err == nil {
		t.Error("destroying a missing folder succeeded")
	}
}

// A folder that is a link loses the link; what it points to stays.
func TestDestroyLinkedFolder(t *testing.T) {
	real := t.TempDir()
	writeFile(t, filepath.Join(real, "x.profile.toml"), "x")
	link := filepath.Join(t.TempDir(), "drako")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("cannot create links here")
	}
	if err := Destroy(link); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Errorf("link still there: %v", err)
	}
	assertContent(t, filepath.Join(real, "x.profile.toml"), "x")
}

// ReadFile follows a linked file like the loader; SameFile sees through it.
func TestReadFileAndSameFile(t *testing.T) {
	_, dir := lairAt(t)
	outside := linkOut(t, dir, "w.profile.toml", "linked", false)
	if got, err := ReadFile(dir, "w.profile.toml"); err != nil || string(got) != "linked" {
		t.Errorf("ReadFile = %q, %v", got, err)
	}
	src, err := os.Stat(outside)
	if err != nil {
		t.Fatal(err)
	}
	if !SameFile(dir, "w.profile.toml", src) {
		t.Error("a deck linked to the source is the same file")
	}
	writeFile(t, filepath.Join(dir, "other.profile.toml"), "linked")
	if SameFile(dir, "other.profile.toml", src) {
		t.Error("a copy is not the same file")
	}
}

// DeckFiles lists what check lints, refused names included.
func TestDeckFiles(t *testing.T) {
	_, dir := lairAt(t)
	if err := os.MkdirAll(filepath.Join(dir, InventoryDirName), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{ConfigFileName, "a.profile.toml", "inventory/b.profile.toml", "x\x1b.profile.toml", "notes.txt"} {
		writeFile(t, filepath.Join(dir, n), "x")
	}
	got, err := DeckFiles(dir)
	want := []string{ConfigFileName, "a.profile.toml", "x\x1b.profile.toml", "inventory/b.profile.toml"}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("DeckFiles = %q, %v; want %q", got, err, want)
	}
}
