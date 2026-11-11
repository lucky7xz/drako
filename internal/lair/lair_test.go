package lair

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Open never creates the directory: a CLI command on a machine drako has not
// run on must leave no trace. Create is the first run.
func TestOpenNeedsTheDirectoryCreateMakesIt(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "drako")

	if _, err := Open(dir); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Open on a missing directory: %v, want fs.ErrNotExist", err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("Open must not create the directory")
	}

	l, err := Create(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if l.Dir() != dir {
		t.Errorf("Dir() = %q, want %q", l.Dir(), dir)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("Create must make the directory: %v", err)
	}
}

// The promise of the package: a write cannot leave the directory, even when
// a link inside it points somewhere else.
func TestWritesDoNotFollowALinkOut(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating links needs privileges on Windows")
	}
	outside := filepath.Join(t.TempDir(), "outside.txt")
	writeFile(t, outside, "untouched")

	for _, name := range []string{"history.log", "drako.log"} {
		l, dir := lairAt(t)
		if err := os.Symlink(outside, filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}

		l.LogExecution("cell", "echo hi") // best-effort: must fail quietly
		if f, err := l.OpenLog(); err == nil {
			f.WriteString("leak")
			f.Close()
			if name == "drako.log" {
				t.Error("OpenLog followed a link out of the directory")
			}
		}
		assertContent(t, outside, "untouched")
	}
}

// A log opened through the Lair outlives it: the CLI closes the Lair at once
// and keeps logging to the file.
func TestOpenLogOutlivesTheLair(t *testing.T) {
	l, dir := lairAt(t)
	f, err := l.OpenLog()
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	l.Close()

	if _, err := f.WriteString("still here\n"); err != nil {
		t.Fatalf("write after Close: %v", err)
	}
	assertContent(t, filepath.Join(dir, "drako.log"), "still here\n")
}
