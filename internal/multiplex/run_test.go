package multiplex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeStep struct {
	argv   []string
	attach bool
	env    []string
}

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

func TestRun_WritesScriptsAndExecutesSteps(t *testing.T) {
	got := captureSteps(t)
	dir := filepath.Join(t.TempDir(), "batch")

	s, err := Plan("drako-t", cells(2), []int{2}, true, dir) // nested: no attach, no cleanup
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(s, dir, []string{"A=1"}); err != nil {
		t.Fatal(err)
	}

	// Scripts on disk, executable, correct content.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 scripts on disk, got %d", len(entries))
	}
	info, _ := entries[0].Info()
	if info.Mode().Perm()&0o100 == 0 {
		t.Errorf("script must be executable, mode %v", info.Mode())
	}
	data, _ := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if !strings.HasPrefix(string(data), "#!/bin/sh\n") {
		t.Errorf("script content missing, got %q", data)
	}

	// Steps executed in order, none attaching (nested).
	if len(*got) != len(s.Steps) {
		t.Fatalf("executed %d steps, want %d", len(*got), len(s.Steps))
	}
	for _, st := range *got {
		if st.attach {
			t.Error("nested launch must not attach")
		}
	}
}

func TestRun_AttachStepGetsTerminalAndEnv(t *testing.T) {
	got := captureSteps(t)
	dir := filepath.Join(t.TempDir(), "batch")

	s, err := Plan("drako-t", cells(1), []int{1}, false, dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(s, dir, []string{"DRAKO_PROFILE=work"}); err != nil {
		t.Fatal(err)
	}

	last := (*got)[len(*got)-1]
	if !last.attach || last.argv[1] != "attach-session" {
		t.Fatalf("last step must attach, got %+v", last)
	}
	if len(last.env) != 1 || last.env[0] != "DRAKO_PROFILE=work" {
		t.Errorf("attach step must carry the sanitized env, got %v", last.env)
	}

	// Attached sessions clean their script dir up after control returns.
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("script dir should be removed after an attached session ends")
	}
}

func TestRun_NestedLeavesScriptsAlone(t *testing.T) {
	captureSteps(t)
	dir := filepath.Join(t.TempDir(), "batch")

	s, err := Plan("drako-t", cells(2), []int{2}, true, dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(s, dir, nil); err != nil {
		t.Fatal(err)
	}
	// Nested panes spawn asynchronously; deleting now would race them.
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("nested launch must leave the script dir in place: %v", err)
	}
}
