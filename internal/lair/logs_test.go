package lair

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// lairAt opens a Lair on a fresh temp directory.
func lairAt(t *testing.T) (*Lair, string) {
	t.Helper()
	dir := t.TempDir()
	l, err := Create(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	return l, dir
}

// TestRotate pins the rotator's behavior: it moves a log to its archive only
// once the log exceeds the cap, overwrites any previous archive, and never
// fails when there is nothing to rotate.
func TestRotate(t *testing.T) {
	const maxBytes = 10

	t.Run("under cap: nothing happens", func(t *testing.T) {
		l, dir := lairAt(t)
		path := filepath.Join(dir, "drako.log")
		archive := filepath.Join(dir, "drako.log.old")
		writeFile(t, path, "small")

		l.rotate("drako.log", "drako.log.old", maxBytes)

		assertContent(t, path, "small") // live log untouched
		assertMissing(t, archive)       // no backup created
	})

	t.Run("over cap: log moves to archive", func(t *testing.T) {
		l, dir := lairAt(t)
		path := filepath.Join(dir, "drako.log")
		archive := filepath.Join(dir, "drako.log.old")
		writeFile(t, path, "this content is well over ten bytes")

		l.rotate("drako.log", "drako.log.old", maxBytes)

		assertMissing(t, path) // live log rotated away
		assertContent(t, archive, "this content is well over ten bytes")
	})

	t.Run("missing log: no-op, no panic", func(t *testing.T) {
		l, dir := lairAt(t)
		path := filepath.Join(dir, "drako.log")
		archive := filepath.Join(dir, "drako.log.old")

		l.rotate("drako.log", "drako.log.old", maxBytes) // the log does not exist

		assertMissing(t, path)
		assertMissing(t, archive)
	})

	t.Run("existing archive is overwritten", func(t *testing.T) {
		l, dir := lairAt(t)
		path := filepath.Join(dir, "drako.log")
		archive := filepath.Join(dir, "drako.log.old")
		writeFile(t, path, "fresh content over the cap limit")
		writeFile(t, archive, "stale backup")

		l.rotate("drako.log", "drako.log.old", maxBytes)

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

func TestLogExecutionAppendsHistory(t *testing.T) {
	l, dir := lairAt(t)

	l.LogExecution("my cell", "batch: echo hi")

	data, err := os.ReadFile(filepath.Join(dir, "history.log"))
	if err != nil {
		t.Fatalf("history.log not written: %v", err)
	}
	line := string(data)
	if !strings.Contains(line, "my cell") || !strings.Contains(line, "(exec: batch: echo hi)") {
		t.Errorf("history entry malformed: %q", line)
	}
}

// history.log holds full command lines, so only its owner may read it, even
// a file created before it was made private.
func TestLogExecutionKeepsHistoryPrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no Unix permissions")
	}
	l, dir := lairAt(t)
	path := filepath.Join(dir, "history.log")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	l.LogExecution("cell", "curl -H 'Authorization: secret'")

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("history.log mode = %o, want 600", mode)
	}
}

// The TUI and the CLI share OpenLog, so a CLI-only user still gets rotation.
func TestOpenLogRotatesPastOneMB(t *testing.T) {
	l, dir := lairAt(t)
	writeFile(t, filepath.Join(dir, "drako.log"), strings.Repeat("x", 1024*1024+1))

	f, err := l.OpenLog()
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	if info, err := os.Stat(filepath.Join(dir, "drako.log")); err != nil || info.Size() != 0 {
		t.Errorf("want a fresh drako.log, got %v, %v", info, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "drako.log.old")); err != nil {
		t.Errorf("want the old log archived: %v", err)
	}
}
