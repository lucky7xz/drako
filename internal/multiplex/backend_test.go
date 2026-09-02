package multiplex

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// fakeBackend records what Launch was handed instead of touching a multiplexer.
type fakeBackend struct {
	session string
	cmds    []Command
	tabs    []int
	paths   []string
	env     []string
	calls   int
	attach  bool
	err     error
}

func (f *fakeBackend) Name() string { return "fake" }

func (f *fakeBackend) Launch(session string, cmds []Command, tabs []int, paths, env []string) (bool, error) {
	f.session, f.cmds, f.tabs, f.paths, f.env = session, cmds, tabs, paths, env
	f.calls++
	return f.attach, f.err
}

func TestLaunch_WritesAScriptPerCellAndHandsOverThePaths(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "batch")
	f := &fakeBackend{}

	if err := Launch(f, "drako-t", cells(3), []int{2, 1}, dir, []string{"A=1"}); err != nil {
		t.Fatal(err)
	}
	if f.calls != 1 {
		t.Fatalf("backend launched %d times, want 1", f.calls)
	}
	if len(f.paths) != 3 {
		t.Fatalf("paths = %v, want one per cell", f.paths)
	}
	if !slices.Equal(f.tabs, []int{2, 1}) || f.session != "drako-t" {
		t.Errorf("backend got session %q tabs %v", f.session, f.tabs)
	}

	// Every path names a real, executable file holding that cell's command.
	for i, p := range f.paths {
		if filepath.Dir(p) != dir {
			t.Errorf("path %d = %q, want it inside %q", i, p, dir)
		}
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("script %d not written: %v", i, err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Errorf("script %d mode = %v, want 0700 — it holds the user's command text", i, info.Mode().Perm())
		}
		body, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if want := cells(3)[i].Script; !strings.Contains(string(body), want) {
			t.Errorf("script %d does not run %q:\n%s", i, want, body)
		}
	}
}

// An attached session is over when Launch returns, so its scripts go; a
// detached one is still spawning panes that hold them open.
func TestLaunch_RemovesScriptsOnlyAfterAnAttachedSession(t *testing.T) {
	attached := filepath.Join(t.TempDir(), "batch")
	if err := Launch(&fakeBackend{attach: true}, "s", cells(1), []int{1}, attached, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(attached); !os.IsNotExist(err) {
		t.Error("an attached session's scripts should be cleaned up")
	}

	detached := filepath.Join(t.TempDir(), "batch")
	if err := Launch(&fakeBackend{attach: false}, "s", cells(1), []int{1}, detached, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(detached); err != nil {
		t.Error("a detached launch must leave its scripts for the panes still starting")
	}
}

func TestLaunch_RejectsABatchTheBackendCouldNotRun(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "batch")
	cases := []struct {
		name string
		cmds []Command
		tabs []int
	}{
		{"nothing marked", nil, nil},
		{"over the cap", cells(10), []int{4, 4, 2}},
		{"tabs that miss cells", cells(4), []int{3}},
	}
	for _, c := range cases {
		f := &fakeBackend{}
		if err := Launch(f, "s", c.cmds, c.tabs, dir, nil); err == nil {
			t.Errorf("%s: want an error", c.name)
		}
		if f.calls != 0 {
			t.Errorf("%s: the backend must not be reached", c.name)
		}
	}
}

func TestLaunch_ReportsWhatTheBackendFailedOn(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "batch")
	boom := errors.New("no such pane")
	err := Launch(&fakeBackend{err: boom}, "s", cells(1), []int{1}, dir, nil)
	if !errors.Is(err, boom) {
		t.Errorf("err = %v, want it to wrap the backend's own error", err)
	}
}
