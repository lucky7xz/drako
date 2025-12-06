package core

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/lucky7xz/drako/internal/config"
)

func TestFindCommandByName(t *testing.T) {
	cfg := config.Config{
		Commands: []config.Command{
			{Name: "A", Command: "echo A"},
			{Name: "B", Items: []config.CommandItem{{Name: "B1", Command: "echo B1"}}},
		},
	}
	tests := []struct {
		name     string
		in       string
		wantOk   bool
		wantTop  bool
		wantItem bool
	}{
		{"top-level hit", "A", true, true, false},
		{"nested item hit", "B1", true, true, true},
		{"miss", "X", false, false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, it, ok := FindCommandByName(cfg, tc.in)
			if ok != tc.wantOk {
				t.Fatalf("ok=%v, want %v", ok, tc.wantOk)
			}
			if p != nil != tc.wantTop {
				t.Fatalf("top-level presence=%v, want %v", p != nil, tc.wantTop)
			}
			if it != nil != tc.wantItem {
				t.Fatalf("item presence=%v, want %v", it != nil, tc.wantItem)
			}
		})
	}
}

func TestRunCommand_ConfigMatchButEmptyCommand(t *testing.T) {
	// Save original seams
	oldPause, oldLook, oldCmd := pauseFn, lookPathFn, commandFn
	defer func() { pauseFn, lookPathFn, commandFn = oldPause, oldLook, oldCmd }()

	// Stub seams
	var paused bool
	gotArgs := []string{}
	pauseFn = func(string) { paused = true }
	lookPathFn = func(s string) (string, error) { return "", fmt.Errorf("should not be called") }
	commandFn = func(name string, args ...string) *exec.Cmd {
		gotArgs = append([]string{name}, args...)
		return exec.Command("echo")
	}

	cfg := config.Config{
		Commands: []config.Command{{Name: "test", Command: ""}},
	}
	RunCommand(cfg, "test")

	// Should pause but not execute anything
	if !paused {
		t.Fatal("expected pause call")
	}
	if len(gotArgs) > 0 {
		t.Fatalf("expected no command execution, got %v", gotArgs)
	}
}

func TestRunCommand_PathFallback(t *testing.T) {
	// Save original seams
	oldPause, oldLook, oldCmd := pauseFn, lookPathFn, commandFn
	defer func() { pauseFn, lookPathFn, commandFn = oldPause, oldLook, oldCmd }()

	// Stub seams
	var gotArgs []string
	gotArgs = nil // Reset capture
	pauseFn = func(string) {}
	lookPathFn = func(s string) (string, error) { return "/bin/echo", nil }
	commandFn = func(name string, args ...string) *exec.Cmd {
		gotArgs = append([]string{name}, args...)
		return exec.Command("echo")
	}

	cfg := config.Config{} // No matching configured command
	RunCommand(cfg, "echo")

	if len(gotArgs) == 0 || gotArgs[0] != "/bin/echo" {
		t.Fatalf("expected to build cmd with looked-up path, got %v", gotArgs)
	}
}
