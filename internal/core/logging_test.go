package core

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRotateLogIfNeeded pins the rotator's behavior: it moves a log to its
// archive only once the log exceeds the cap, overwrites any previous archive,
// and never fails when there is nothing to rotate.
func TestRotateLogIfNeeded(t *testing.T) {
	const maxBytes = 10

	t.Run("under cap: nothing happens", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "drako.log")
		archive := filepath.Join(dir, "drako.log.old")
		writeFile(t, path, "small")

		RotateLogIfNeeded(path, archive, maxBytes)

		assertContent(t, path, "small") // live log untouched
		assertMissing(t, archive)       // no backup created
	})

	t.Run("over cap: log moves to archive", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "drako.log")
		archive := filepath.Join(dir, "drako.log.old")
		writeFile(t, path, "this content is well over ten bytes")

		RotateLogIfNeeded(path, archive, maxBytes)

		assertMissing(t, path) // live log rotated away
		assertContent(t, archive, "this content is well over ten bytes")
	})

	t.Run("missing log: no-op, no panic", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "drako.log")
		archive := filepath.Join(dir, "drako.log.old")

		RotateLogIfNeeded(path, archive, maxBytes) // path does not exist

		assertMissing(t, path)
		assertMissing(t, archive)
	})

	t.Run("existing archive is overwritten", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "drako.log")
		archive := filepath.Join(dir, "drako.log.old")
		writeFile(t, path, "fresh content over the cap limit")
		writeFile(t, archive, "stale backup")

		RotateLogIfNeeded(path, archive, maxBytes)

		assertMissing(t, path)
		assertContent(t, archive, "fresh content over the cap limit") // old backup replaced
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != want {
		t.Errorf("%s = %q, want %q", filepath.Base(path), got, want)
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected %s to not exist, stat err = %v", filepath.Base(path), err)
	}
}
