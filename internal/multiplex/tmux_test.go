package multiplex

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type fakeStep struct {
	argv   []string
	attach bool
}

// captureSteps stubs the tmux invocation seam and records what would have run.
func captureSteps(t *testing.T) *[]fakeStep {
	t.Helper()
	old := execStep
	t.Cleanup(func() { execStep = old })
	var got []fakeStep
	execStep = func(argv []string, attach bool) error {
		got = append(got, fakeStep{argv, attach})
		return nil
	}
	return &got
}

func TestTmux_RunsEveryPlannedStepInOrder(t *testing.T) {
	got := captureSteps(t)

	if err := Launch(NewTmux(true), "drako-t", cells(2), []int{2},
		filepath.Join(t.TempDir(), "batch")); err != nil {
		t.Fatal(err)
	}

	// One tab is one tmux call, its commands chained so nothing can come between.
	if len(*got) != 1 {
		t.Fatalf("ran %d tmux calls, want 1 for the tab", len(*got))
	}
	st := (*got)[0]
	if d := describeStep(st.argv); d != "new-window; split-window; select-layout" {
		t.Errorf("the tab ran %q", d)
	}
	if st.attach {
		t.Error("a nested launch never attaches — drako already owns a pane")
	}
}

func TestTmux_LastStepAttaches(t *testing.T) {
	got := captureSteps(t)

	if err := Launch(NewTmux(false), "drako-t", cells(1), []int{1},
		filepath.Join(t.TempDir(), "batch")); err != nil {
		t.Fatal(err)
	}

	last := (*got)[len(*got)-1]
	if !last.attach || last.argv[1] != "attach-session" {
		t.Fatalf("the last step must attach, got %+v", last)
	}
}

// A step that fails names the tmux verb that failed, not just "exit 1".
func TestTmux_ReportsWhichVerbFailed(t *testing.T) {
	old := execStep
	t.Cleanup(func() { execStep = old })
	execStep = func(argv []string, attach bool) error {
		if slices.Contains(argv, "split-window") {
			return errors.New("no space for a new pane")
		}
		return nil
	}

	err := Launch(NewTmux(true), "drako-t", cells(2), []int{2},
		filepath.Join(t.TempDir(), "batch"))
	if err == nil {
		t.Fatal("a failing step must surface")
	}
	for _, want := range []string{"tmux", "split-window", "no space"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q", err, want)
		}
	}
}
