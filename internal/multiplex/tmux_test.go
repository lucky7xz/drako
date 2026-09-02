package multiplex

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

type fakeStep struct {
	argv   []string
	attach bool
	env    []string
}

// captureSteps stubs the tmux invocation seam and records what would have run.
func captureSteps(t *testing.T) *[]fakeStep {
	t.Helper()
	old := execStep
	t.Cleanup(func() { execStep = old })
	var got []fakeStep
	execStep = func(argv []string, attach bool, env []string) error {
		got = append(got, fakeStep{argv, attach, env})
		return nil
	}
	return &got
}

func TestTmux_RunsEveryPlannedStepInOrder(t *testing.T) {
	got := captureSteps(t)

	if err := Launch(NewTmux(true), "drako-t", cells(2), []int{2},
		filepath.Join(t.TempDir(), "batch"), []string{"A=1"}); err != nil {
		t.Fatal(err)
	}

	want := []string{"new-window", "split-window", "select-layout"}
	if len(*got) != len(want) {
		t.Fatalf("ran %d steps, want %v", len(*got), want)
	}
	for i, st := range *got {
		if st.argv[1] != want[i] {
			t.Errorf("step %d = %q, want %q", i, st.argv[1], want[i])
		}
		if st.attach {
			t.Error("a nested launch never attaches — drako already owns a pane")
		}
	}
}

func TestTmux_AttachStepGetsTheTerminalAndTheEnv(t *testing.T) {
	got := captureSteps(t)

	if err := Launch(NewTmux(false), "drako-t", cells(1), []int{1},
		filepath.Join(t.TempDir(), "batch"), []string{"DRAKO_PROFILE=work"}); err != nil {
		t.Fatal(err)
	}

	last := (*got)[len(*got)-1]
	if !last.attach || last.argv[1] != "attach-session" {
		t.Fatalf("the last step must attach, got %+v", last)
	}
	if len(last.env) != 1 || last.env[0] != "DRAKO_PROFILE=work" {
		t.Errorf("the attach step must carry the sanitized env, got %v", last.env)
	}
}

// A step that fails names the tmux verb that failed, not just "exit 1".
func TestTmux_ReportsWhichVerbFailed(t *testing.T) {
	old := execStep
	t.Cleanup(func() { execStep = old })
	execStep = func(argv []string, attach bool, env []string) error {
		if argv[1] == "split-window" {
			return errors.New("no space for a new pane")
		}
		return nil
	}

	err := Launch(NewTmux(true), "drako-t", cells(2), []int{2},
		filepath.Join(t.TempDir(), "batch"), nil)
	if err == nil {
		t.Fatal("a failing step must surface")
	}
	for _, want := range []string{"tmux", "split-window", "no space"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q", err, want)
		}
	}
}
