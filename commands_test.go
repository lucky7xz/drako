package main

import (
	"fmt"
	"os/exec"
	"testing"
)

func TestFindCommandByName(t *testing.T) {
	cfg := Config{
		Commands: []Command{
			{Name: "A", Command: "echo A"},
			{Name: "B", Items: []CommandItem{{Name: "B1", Command: "echo B1"}}},
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
			p, it, ok := findCommandByName(cfg, tc.in)
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

	cfg := Config{
		Commands: []Command{{Name: "test", Command: ""}},
	}
	runCommand(cfg, "test")

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

	cfg := Config{} // No matching configured command
	runCommand(cfg, "echo")

	if len(gotArgs) == 0 || gotArgs[0] != "/bin/echo" {
		t.Fatalf("expected to build cmd with looked-up path, got %v", gotArgs)
	}
}

func TestExpandCommandTokens(t *testing.T) {
	cfg := Config{DR4koPath: "/x"}
	tests := map[string]string{
		"":                      "",
		"no_tokens":            "no_tokens",
		"cd {dR4ko_path}":      "cd /x",
		"{dR4ko_path} {dR4ko_path}": "/x /x",
	}
	for in, want := range tests {
		t.Run(in, func(t *testing.T) {
			if got := expandCommandTokens(in, cfg); got != want {
				t.Fatalf("want %q, got %q", want, got)
			}
		})
	}
}