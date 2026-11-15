package proc

import (
	"errors"
	"os/exec"
	"runtime"
	"slices"
	"syscall"
	"testing"
	"time"

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

// A plan with nothing to run is refused, never handed to the system.
func TestNothingToRunIsRefused(t *testing.T) {
	plan := launch.Shell("sh", "blank", "   ", "")
	if err := Run(plan); !errors.Is(err, errNothingToRun) {
		t.Errorf("Run = %v, want errNothingToRun", err)
	}
	if out, err := Capture(plan); !errors.Is(err, errNothingToRun) || out != nil {
		t.Errorf("Capture = %q, %v, want nothing and errNothingToRun", out, err)
	}
}

// debug_execution: both streams come back, in order, and nothing is lost when
// the command fails.
func TestCapture(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs sh")
	}
	out, err := Capture(launch.Shell("sh", "c", "echo out; echo err >&2; exit 3", "work"))
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 3 {
		t.Errorf("want the command's exit 3, got %v", err)
	}
	if string(out) != "out\nerr\n" {
		t.Errorf("captured %q, want both streams", out)
	}
}

func TestToolCommand(t *testing.T) {
	cmd := Git.Command("clone", "--", "url", "dir")
	if want := []string{"git", "clone", "--", "url", "dir"}; !slices.Equal(cmd.Args, want) {
		t.Errorf("args=%v, want %v", cmd.Args, want)
	}
	if (Tool{}).Found() {
		t.Error("the zero Tool names no program and must not be found")
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

// TERM sent to drako is passed on to the command, and the result says drako
// is to end too.
func TestRunRelayedPassesTermAndEnds(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no kill on Windows")
	}
	// The command would run 5 s and exit 0; only a passed-on TERM ends it early.
	start := time.Now()
	err := runRelayed(exec.Command("sh", "-c", "kill -TERM $PPID; exec sleep 5"))
	var end *ToldToEnd
	if !errors.As(err, &end) || end.Code != 128+int(syscall.SIGTERM) {
		t.Fatalf("want ToldToEnd with code 143, got %v", err)
	}
	if time.Since(start) > 3*time.Second {
		t.Error("the command was not ended by the passed-on TERM")
	}
}

func TestRunRelayedHangUpEnds(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no kill on Windows")
	}
	err := runRelayed(exec.Command("sh", "-c", "kill -HUP $PPID; exec sleep 5"))
	var end *ToldToEnd
	if !errors.As(err, &end) || end.Code != 128+int(syscall.SIGHUP) {
		t.Fatalf("want ToldToEnd with code 129, got %v", err)
	}
}
