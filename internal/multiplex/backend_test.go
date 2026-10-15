package multiplex

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
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
	calls   int
	err     error
}

func (f *fakeBackend) Name() string { return "fake" }

func (f *fakeBackend) Launch(session string, cmds []Command, tabs []int, paths []string) error {
	f.session, f.cmds, f.tabs, f.paths = session, cmds, tabs, paths
	f.calls++
	return f.err
}

func TestLaunch_WritesAScriptPerCellAndHandsOverThePaths(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "batch")
	f := &fakeBackend{}

	if err := Launch(f, "drako-t", cells(3), []int{2, 1}, dir); err != nil {
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
		if want := cells(3)[i].Argv[2]; !strings.Contains(string(body), want) {
			t.Errorf("script %d does not run %q:\n%s", i, want, body)
		}
	}
}

// Panes may still be starting when a launch returns, so Launch leaves the
// scripts of a launch that worked to delete themselves.
func TestLaunch_LeavesTheScriptsToThePanes(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "batch")
	if err := Launch(&fakeBackend{}, "s", cells(1), []int{1}, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Error("the scripts must stay for the panes still starting")
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
		if err := Launch(f, "s", c.cmds, c.tabs, dir); err == nil {
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
	err := Launch(&fakeBackend{err: boom}, "s", cells(1), []int{1}, dir)
	if !errors.Is(err, boom) {
		t.Errorf("err = %v, want it to wrap the backend's own error", err)
	}
}

// A launch that failed may have written scripts holding command text; they must
// not be left in the temp dir.
func TestLaunch_RemovesTheScriptsWhenTheBackendFails(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "batch")
	if err := Launch(&fakeBackend{err: errors.New("attach failed")}, "s", cells(2), []int{2}, dir); err == nil {
		t.Fatal("the backend error must surface")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("scripts left behind after a failed launch (%v)", err)
	}
}

// runningBackend starts every script the way a pane would.
type runningBackend struct{}

func (runningBackend) Name() string { return "running" }

func (runningBackend) Launch(session string, cmds []Command, tabs []int, paths []string) error {
	for _, p := range paths {
		if out, err := exec.Command("sh", p).CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %v: %s", p, err, out)
		}
	}
	return nil
}

// A script deletes itself once sh has it open, and the last one its directory,
// so no batch leaves command text in the temp dir.
func TestLaunch_ScriptsCleanUpAfterThemselves(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "batch")
	cmds := []Command{{Name: "a", Argv: []string{"sh", "-c", "true"}}, {Name: "b", Argv: []string{"sh", "-c", "true"}}}
	if err := Launch(runningBackend{}, "s", cmds, []int{2}, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("the scripts stayed after every pane ran (%v)", err)
	}
}
