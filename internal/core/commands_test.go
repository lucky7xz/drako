package core

import (
	"errors"
	"os/exec"
	"runtime"
	"syscall"
	"testing"

	"github.com/lucky7xz/drako/internal/config"
)

func TestSelectionResolve(t *testing.T) {
	cfg := config.Config{
		Commands: []config.Command{
			{Name: "A", Command: "echo A"},
			{Name: "B", Items: []config.CommandItem{{Name: "Same", Command: "echo B1"}}},
			{Name: "C", Items: []config.CommandItem{{Name: "Same", Command: "echo C1"}}},
			{Name: "Same", Command: "echo cell"},
		},
	}
	tests := []struct {
		name string
		sel  Selection
		want string // the resolved command, "" for no match
	}{
		{"cell", Selection{Cell: "A"}, "echo A"},
		{"item in its own folder", Selection{Cell: "C", Item: 1}, "echo C1"},
		{"cell sharing an item's name", Selection{Cell: "Same"}, "echo cell"},
		{"item past the folder's end", Selection{Cell: "B", Item: 2}, ""},
		{"no such cell", Selection{Cell: "X"}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cell, item, ok := tc.sel.Resolve(cfg)
			got := ""
			if ok {
				got = cell.Command
				if item != nil {
					got = item.Command
				}
			}
			if got != tc.want {
				t.Errorf("Resolve(%+v) = %q, want %q", tc.sel, got, tc.want)
			}
		})
	}
}

func TestBuildShellCmd(t *testing.T) {
	tests := []struct {
		shell    string
		wantArgs []string
	}{
		{"bash", []string{"bash", "-lc", "echo hi"}},
		{"sh", []string{"sh", "-c", "echo hi"}},
		{"zsh", []string{"zsh", "-lc", "echo hi"}},
		{"fish", []string{"fish", "-c", "echo hi"}},
		{"pwsh", []string{"pwsh", "-NoLogo", "-NoProfile", "-Command", "echo hi"}},
		{"powershell", []string{"powershell", "-NoLogo", "-NoProfile", "-Command", "echo hi"}},
		{"powershell.exe", []string{"powershell", "-NoLogo", "-NoProfile", "-Command", "echo hi"}},
		{"cmd", []string{"cmd", "/C", "echo hi"}},
	}
	for _, tc := range tests {
		t.Run(tc.shell, func(t *testing.T) {
			got := buildShellCmd(tc.shell, "echo hi")
			if len(got.Args) != len(tc.wantArgs) {
				t.Fatalf("args=%v, want %v", got.Args, tc.wantArgs)
			}
			for i := range tc.wantArgs {
				if got.Args[i] != tc.wantArgs[i] {
					t.Fatalf("args=%v, want %v", got.Args, tc.wantArgs)
				}
			}
		})
	}
}

func TestFallbackShell(t *testing.T) {
	if got := fallbackShell("windows"); got != "powershell" {
		t.Fatalf("windows fallback=%q, want powershell", got)
	}
	for _, goos := range []string{"linux", "darwin", "freebsd"} {
		if got := fallbackShell(goos); got != "bash" {
			t.Fatalf("%s fallback=%q, want bash", goos, got)
		}
	}
}

func TestRunCommand_ConfigMatchButEmptyCommand(t *testing.T) {
	oldPause, oldCmd := pauseFn, commandFn
	defer func() { pauseFn, commandFn = oldPause, oldCmd }()

	var paused bool
	gotArgs := []string{}
	pauseFn = func(string) { paused = true }
	commandFn = func(name string, args ...string) *exec.Cmd {
		gotArgs = append([]string{name}, args...)
		return exec.Command("echo")
	}

	for _, sel := range []Selection{{Cell: "test"}, {Cell: "blank"}, {Cell: "gone"}} {
		paused, gotArgs = false, nil
		cfg := config.Config{
			Commands: []config.Command{{Name: "test", Command: ""}, {Name: "blank", Command: "   "}},
		}
		RunCommand(cfg, sel, "")

		// Says so and pauses, but runs nothing.
		if !paused {
			t.Errorf("%s: expected pause call", sel.Cell)
		}
		if len(gotArgs) > 0 {
			t.Errorf("%s: expected no command execution, got %v", sel.Cell, gotArgs)
		}
	}
}

// Ctrl-C reaches drako and the command alike; drako must outlive it.
func TestRunRelayedOutlivesInterrupt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no kill on Windows")
	}
	err := runRelayed(exec.Command("sh", "-c", "kill -INT $PPID; sleep 0.2; exit 3"))
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 3 {
		t.Fatalf("want the command's exit 3, got %v", err)
	}
}

func TestRunRelayedPassesTerm(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no kill on Windows")
	}
	err := runRelayed(exec.Command("sh", "-c", "kill -TERM $PPID; exec sleep 5"))
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("want the command killed by TERM, got %v", err)
	}
	if ws, ok := exitErr.Sys().(syscall.WaitStatus); !ok || !ws.Signaled() || ws.Signal() != syscall.SIGTERM {
		t.Errorf("want death by SIGTERM, got %v", err)
	}
}
