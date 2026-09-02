package multiplex

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The full tab: split the root right, then each column down, so the panes end
// up in reading order — ① ② across the top, ③ ④ beneath.
func TestSplitTree_FillsATabInReadingOrder(t *testing.T) {
	cases := []struct {
		panes int
		want  []split
	}{
		{1, nil},
		{2, []split{{0, "right"}}},
		{3, []split{{0, "right"}, {0, "down"}}},
		{4, []split{{0, "right"}, {0, "down"}, {1, "down"}}},
	}
	for _, c := range cases {
		if got := splitTree(c.panes); !slices.Equal(got, c.want) {
			t.Errorf("splitTree(%d) = %v, want %v", c.panes, got, c.want)
		}
	}
}

// Each layout is the previous one plus a split, so growing a tab never
// rearranges the panes already in it.
func TestSplitTree_EachSizeExtendsTheLast(t *testing.T) {
	for n := 2; n <= PanesPerTab; n++ {
		prev, cur := splitTree(n-1), splitTree(n)
		if !slices.Equal(cur[:len(prev)], prev) {
			t.Errorf("splitTree(%d) = %v does not extend splitTree(%d) = %v", n, cur, n-1, prev)
		}
	}
}

// Split i creates pane i+1, so a tab of n panes needs n-1 splits and every
// parent must be a pane that already exists when its split runs.
func TestSplitTree_OnlySplitsPanesThatExistYet(t *testing.T) {
	for n := 1; n <= PanesPerTab; n++ {
		tree := splitTree(n)
		if len(tree) != n-1 {
			t.Fatalf("splitTree(%d) has %d splits, want %d", n, len(tree), n-1)
		}
		for i, s := range tree {
			// Panes 0..i exist before this split; it creates pane i+1.
			if s.parent < 0 || s.parent > i {
				t.Errorf("splitTree(%d) split %d targets pane %d, which does not exist yet", n, i, s.parent)
			}
			if s.dir != "right" && s.dir != "down" {
				t.Errorf("splitTree(%d) split %d direction %q is not one herdr takes", n, i, s.dir)
			}
		}
	}
}

// A tab can never hold more than the ceiling, so splitTree is never asked to.
func TestCheckTabs_RejectsATabOverTheCeiling(t *testing.T) {
	if err := checkTabs([]int{PanesPerTab + 1}, PanesPerTab+1); err == nil {
		t.Errorf("a tab of %d exceeds the %d-pane ceiling, want an error", PanesPerTab+1, PanesPerTab)
	}
	if err := checkTabs([]int{PanesPerTab}, PanesPerTab); err != nil {
		t.Errorf("a full tab is fine, got %v", err)
	}
}

// fakeHerdr stubs the herdr invocation seam, recording every call and minting
// pane ids the way the server would.
func fakeHerdr(t *testing.T) *[][]string {
	t.Helper()
	old := herdrExec
	t.Cleanup(func() { herdrExec = old })
	var got [][]string
	n := 0
	herdrExec = func(argv []string) ([]byte, error) {
		got = append(got, argv)
		switch {
		case argv[1] == "tab" && argv[2] == "create":
			n++
			return fmt.Appendf(nil, `{"result":{"tab":{"tab_id":"w1:t%d"},"root_pane":{"pane_id":"w1:p%d"}}}`, n, n), nil
		case argv[1] == "pane" && argv[2] == "split":
			n++
			return fmt.Appendf(nil, `{"result":{"pane":{"pane_id":"w1:p%d"}}}`, n), nil
		}
		return []byte(`{"result":{}}`), nil
	}
	return &got
}

func launchHerdr(t *testing.T, n int, tabs []int) ([][]string, []string) {
	t.Helper()
	got := fakeHerdr(t)
	dir := filepath.Join(t.TempDir(), "batch")
	if err := Launch(NewHerdr(), "drako-t", cells(n), tabs, dir, nil); err != nil {
		t.Fatal(err)
	}
	return *got, scriptPaths(cells(n), dir)
}

// Every pane in a tab is created before anything runs in it — a split carves
// up a shell, never a pane already busy with a cell.
func TestHerdr_CreatesEveryPaneBeforeRunningInThem(t *testing.T) {
	got, paths := launchHerdr(t, 3, []int{2, 1})
	want := [][]string{
		{"herdr", "tab", "create", "--label", "cell 1·cell 2", "--focus"},
		{"herdr", "pane", "split", "--pane", "w1:p1", "--direction", "right", "--no-focus"},
		{"herdr", "pane", "run", "w1:p1", "exec", paths[0]},
		{"herdr", "pane", "run", "w1:p2", "exec", paths[1]},
		{"herdr", "tab", "create", "--label", "cell 3", "--no-focus"},
		{"herdr", "pane", "run", "w1:p3", "exec", paths[2]},
	}
	if len(got) != len(want) {
		t.Fatalf("ran %d calls, want %d:\n%v", len(got), len(want), got)
	}
	for i := range want {
		if !slices.Equal(got[i], want[i]) {
			t.Errorf("call %d =\n  %v\nwant\n  %v", i, got[i], want[i])
		}
	}
}

// A full tab splits per splitTree, and every split names an id the server
// actually handed back.
func TestHerdr_SplitsAFullTabByTheTree(t *testing.T) {
	got, _ := launchHerdr(t, 4, []int{4})
	var splits [][]string
	for _, c := range got {
		if c[2] == "split" {
			splits = append(splits, c)
		}
	}
	want := [][]string{
		{"herdr", "pane", "split", "--pane", "w1:p1", "--direction", "right", "--no-focus"},
		{"herdr", "pane", "split", "--pane", "w1:p1", "--direction", "down", "--no-focus"},
		{"herdr", "pane", "split", "--pane", "w1:p2", "--direction", "down", "--no-focus"},
	}
	for i := range want {
		if i >= len(splits) || !slices.Equal(splits[i], want[i]) {
			t.Fatalf("splits =\n  %v\nwant\n  %v", splits, want)
		}
	}
}

// exec replaces the pane's shell, so the pane closes when its cell finishes —
// the same as tmux, and what auto_close_execution promises.
func TestHerdr_SendsExecSoAPaneClosesWithItsCell(t *testing.T) {
	got, paths := launchHerdr(t, 1, []int{1})
	for _, c := range got {
		if c[2] != "run" {
			continue
		}
		if !slices.Equal(c[4:], []string{"exec", paths[0]}) {
			t.Errorf("pane run must exec the script, got %v", c)
		}
		return
	}
	t.Fatal("no pane run call")
}

// The batch is worth looking at, but only once: later tabs are built behind it.
func TestHerdr_OnlyTheFirstTabTakesFocus(t *testing.T) {
	got, _ := launchHerdr(t, 3, []int{1, 1, 1})
	var focus []string
	for _, c := range got {
		if c[1] == "tab" {
			focus = append(focus, c[len(c)-1])
		}
	}
	if !slices.Equal(focus, []string{"--focus", "--no-focus", "--no-focus"}) {
		t.Errorf("tab focus flags = %v", focus)
	}
}

// herdr is only ever used from inside herdr, so it never hands over a
// terminal — and its scripts must outlive the call that started the panes.
func TestHerdr_NeverAttaches(t *testing.T) {
	fakeHerdr(t)
	dir := filepath.Join(t.TempDir(), "batch")
	if err := Launch(NewHerdr(), "drako-t", cells(2), []int{2}, dir, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("herdr panes are still starting; their scripts must stay: %v", err)
	}
}

func TestHerdr_ReportsAFailedCall(t *testing.T) {
	old := herdrExec
	t.Cleanup(func() { herdrExec = old })
	herdrExec = func(argv []string) ([]byte, error) {
		return nil, errors.New("pane w1:p9 not found")
	}
	err := Launch(NewHerdr(), "drako-t", cells(1), []int{1}, filepath.Join(t.TempDir(), "b"), nil)
	if err == nil {
		t.Fatal("a failing call must surface")
	}
	for _, want := range []string{"herdr", "not found"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q", err, want)
		}
	}
}

// tab create names its root pane, pane split names the new one; one reader
// handles both, and says so plainly when a reply names neither.
func TestPaneID_ReadsBothReplyShapes(t *testing.T) {
	if id, err := paneID([]byte(`{"result":{"root_pane":{"pane_id":"w1:p4"}}}`)); err != nil || id != "w1:p4" {
		t.Errorf("tab create reply = %q, %v", id, err)
	}
	if id, err := paneID([]byte(`{"result":{"pane":{"pane_id":"w2:p7"}}}`)); err != nil || id != "w2:p7" {
		t.Errorf("pane split reply = %q, %v", id, err)
	}
	for _, bad := range []string{`{"result":{}}`, `not json`, `{"result":{"pane":{"pane_id":""}}}`} {
		if _, err := paneID([]byte(bad)); err == nil {
			t.Errorf("%s names no pane, want an error", bad)
		}
	}
}
