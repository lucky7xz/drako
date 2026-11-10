package core

import (
	"errors"
	"os/exec"
	"runtime"
	"slices"
	"syscall"
	"testing"

	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/launch"
)

func TestCommand_CarriesArgvAndEnv(t *testing.T) {
	plan := launch.Plan{Argv: []string{"sh", "-c", "echo hi"}}
	env := []string{"PATH=/usr/bin", "DRAKO_PROFILE=work"}
	cmd := command(plan, env)
	if !slices.Equal(cmd.Args, plan.Argv) {
		t.Errorf("args=%v, want the plan's %v", cmd.Args, plan.Argv)
	}
	if len(cmd.Env) != 2 || cmd.Env[1] != "DRAKO_PROFILE=work" {
		t.Errorf("the command env must be attached before any run mode, got %v", cmd.Env)
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

	for _, sel := range []launch.Selection{{Cell: "test"}, {Cell: "blank"}, {Cell: "gone"}} {
		paused, gotArgs = false, nil
		cfg := config.Config{
			Commands: []config.Command{{Name: "test", Command: ""}, {Name: "blank", Command: "   "}},
		}
		RunCommand(launch.For(cfg, sel, ""))

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
