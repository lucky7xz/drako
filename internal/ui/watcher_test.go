package ui

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func watchTestDir(t *testing.T) (string, <-chan ConfigChangedMsg) {
	t.Helper()
	dir := t.TempDir()
	got := make(chan ConfigChangedMsg, 16)
	stop, err := WatchConfig(dir, func(msg ConfigChangedMsg) { got <- msg })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(stop)
	return dir, got
}

func TestWatchConfigReportsTomlChangesOnly(t *testing.T) {
	dir, got := watchTestDir(t)
	if err := os.MkdirAll(filepath.Join(dir, "inventory"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"notes.txt", "inventory/x.profile.toml", "a.profile.toml"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x = 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case msg := <-got:
		if filepath.Base(msg.Path) != "a.profile.toml" {
			t.Errorf("reported %s, want only the root .toml", msg.Path)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no change reported for a.profile.toml")
	}
}

// Linux-only probe: one inotify instance per watcher, released by stop.
func TestWatchConfigStopReleasesTheWatcher(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("counts inotify instances via /proc")
	}
	count := func() int {
		n := 0
		ents, _ := os.ReadDir("/proc/self/fd")
		for _, e := range ents {
			if l, _ := os.Readlink(filepath.Join("/proc/self/fd", e.Name())); l == "anon_inode:inotify" {
				n++
			}
		}
		return n
	}
	before := count()
	stop, err := WatchConfig(t.TempDir(), func(ConfigChangedMsg) {})
	if err != nil {
		t.Fatal(err)
	}
	stop()
	if after := count(); after != before {
		t.Errorf("inotify instances: %d before, %d after stop", before, after)
	}
}

// An in-place rewrite (truncate, then write) is reported once, after the file
// is whole: a reload on the truncate would read an empty, "broken" profile.
func TestWatchConfigWaitsForWritesToSettle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.profile.toml")
	if err := os.WriteFile(path, []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	seen := make(chan string, 16)
	stop, err := WatchConfig(dir, func(ConfigChangedMsg) {
		b, _ := os.ReadFile(path)
		seen <- string(b)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond) // the editor, between truncate and write
	f.WriteString("x = 2\n")
	f.Close()

	select {
	case got := <-seen:
		if got != "x = 2\n" {
			t.Errorf("reload read %q mid-write, want the finished file", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("rewrite never reported")
	}
	select {
	case extra := <-seen:
		t.Errorf("one rewrite reported twice (second read %q)", extra)
	case <-time.After(400 * time.Millisecond):
	}
}
