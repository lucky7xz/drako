package multiplex

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func cells(n int) []Command {
	out := make([]Command, n)
	for i := range out {
		out[i] = Command{
			Name:   fmt.Sprintf("cell %d", i+1),
			Script: fmt.Sprintf("echo %d", i+1),
			Shell:  "bash",
		}
	}
	return out
}

// scriptPaths mirrors what Launch writes, so plan tests see the real filenames.
func scriptPaths(cmds []Command, dir string) []string {
	out := make([]string, len(cmds))
	for i, c := range cmds {
		out[i] = filepath.Join(dir, scriptName(i, c))
	}
	return out
}

func planFor(t *testing.T, cmds []Command, tabs []int, inside bool) Session {
	t.Helper()
	s, err := Plan("drako-1", cmds, tabs, scriptPaths(cmds, "/tmp/x"), inside)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// stepVerbs extracts the tmux subcommand of every step for shape assertions.
func stepVerbs(s Session) []string {
	verbs := make([]string, len(s.Steps))
	for i, step := range s.Steps {
		if len(step) < 2 || step[0] != "tmux" {
			verbs[i] = "INVALID:" + strings.Join(step, " ")
			continue
		}
		verbs[i] = step[1]
	}
	return verbs
}

func wantVerbs(t *testing.T, s Session, want ...string) {
	t.Helper()
	got := stepVerbs(s)
	if len(got) != len(want) {
		t.Fatalf("steps = %v, want verbs %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("step %d = %q, want %q (all: %v)", i, got[i], want[i], got)
		}
	}
}

func TestPlan_SingleCommand(t *testing.T) {
	s := planFor(t, cells(1), []int{1}, false)
	wantVerbs(t, s, "new-session", "attach-session")
	if !s.Attach {
		t.Error("outside tmux, the last step hands the terminal over")
	}
	if !strings.Contains(strings.Join(s.Steps[0], " "), "/tmp/x/") {
		t.Errorf("new-session must run the script by its full path: %v", s.Steps[0])
	}
}

// One tab holding every cell: the session's first window, then a split per
// extra cell, arranged tiled.
func TestPlan_OneTabOfFourPanes(t *testing.T) {
	s := planFor(t, cells(4), []int{4}, false)
	wantVerbs(t, s, "new-session", "split-window", "split-window", "split-window",
		"select-layout", "attach-session")
	last := s.Steps[4]
	if last[len(last)-1] != "tiled" {
		t.Errorf("a multi-pane tab must be arranged tiled, got %v", last)
	}
}

// Five cells used to explode into five windows. The vector decides now, so
// [4 1] is two tabs — four side by side, then one.
func TestPlan_TwoTabsFromFiveCells(t *testing.T) {
	s := planFor(t, cells(5), []int{4, 1}, false)
	wantVerbs(t, s, "new-session", "split-window", "split-window", "split-window",
		"select-layout", "new-window", "attach-session")
	// A one-pane tab needs no layout step.
	if strings.Contains(strings.Join(s.Steps[5], " "), "tiled") {
		t.Errorf("a single-pane tab must not be arranged: %v", s.Steps[5])
	}
}

// A tab holds several cells, so it is named after all of them — the identity a
// pane cannot carry.
func TestPlan_TabsAreNamedAfterTheirCells(t *testing.T) {
	s := planFor(t, cells(5), []int{4, 1}, false)
	if first := strings.Join(s.Steps[0], " "); !strings.Contains(first, "cell 1·cell 2·cell 3·cell 4") {
		t.Errorf("the first tab must be named after its four cells: %v", s.Steps[0])
	}
	if second := strings.Join(s.Steps[5], " "); !strings.Contains(second, "cell 5") {
		t.Errorf("the second tab must be named after its cell: %v", s.Steps[5])
	}
}

// Inside tmux the batch gets its own windows; drako's pane is never split.
func TestPlan_InsideTmuxMakesItsOwnTabs(t *testing.T) {
	s := planFor(t, cells(3), []int{3}, true)
	wantVerbs(t, s, "new-window", "split-window", "split-window", "select-layout")
	if s.Attach {
		t.Error("nested launch must not attach (drako already owns a pane)")
	}
	// No -t: the steps join the session drako is already in.
	for _, step := range s.Steps {
		if slices.Contains(step, "-t") {
			t.Errorf("a nested step must not target a session by name: %v", step)
		}
	}
}

func TestPlan_InsideTmuxTwoTabs(t *testing.T) {
	s := planFor(t, cells(6), []int{4, 2}, true)
	wantVerbs(t, s, "new-window", "split-window", "split-window", "split-window",
		"select-layout", "new-window", "split-window", "select-layout")
}

// The vector and the cells must agree, or panes would be dropped or invented.
func TestPlan_RejectsATabVectorThatMissesCells(t *testing.T) {
	cmds := cells(4)
	for _, tabs := range [][]int{nil, {3}, {5}, {2, 3}, {4, 0, 1}} {
		if _, err := Plan("s", cmds, tabs, scriptPaths(cmds, "/tmp/x"), false); err == nil {
			t.Errorf("tabs %v does not lay out 4 cells, want an error", tabs)
		}
	}
	// Any vector that does add up is accepted, however it splits them.
	for _, tabs := range [][]int{{4}, {2, 2}, {1, 1, 1, 1}} {
		if _, err := Plan("s", cmds, tabs, scriptPaths(cmds, "/tmp/x"), false); err != nil {
			t.Errorf("tabs %v lays out 4 cells, got %v", tabs, err)
		}
	}
}

// Plan is handed the scripts rather than deriving them, so a caller that
// passes the wrong number is a bug worth catching rather than an index panic.
func TestPlan_RejectsAScriptPerCellMismatch(t *testing.T) {
	if _, err := Plan("s", cells(3), []int{3}, []string{"/tmp/x/a.sh"}, false); err == nil {
		t.Error("three cells with one script must be rejected")
	}
}
