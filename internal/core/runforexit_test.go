package core

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/lucky7xz/drako/internal/config"
)

// The test binary doubles as drako run: RunForExit may replace the process
// with the command, so tests run it in a child (startRun) that lands here.
func TestMain(m *testing.M) {
	if command, ok := os.LookupEnv("DRAKO_TEST_RUN"); ok {
		code, err := RunForExit(shCfg, os.Getenv("DRAKO_TEST_NAME"), command, os.Getenv("DRAKO_TEST_PROFILE"))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(125)
		}
		os.Exit(code)
	}
	os.Exit(m.Run())
}

// runExitDir isolates the config dir so history.log lands in a temp dir.
func runExitDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)
	return filepath.Join(tmp, "drako")
}

var shCfg = config.Config{DefaultShell: "sh"}

// startRun starts a child test process that runs command through RunForExit.
func startRun(t *testing.T, name, command, profile string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), "DRAKO_TEST_RUN="+command, "DRAKO_TEST_NAME="+name, "DRAKO_TEST_PROFILE="+profile)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	return cmd
}

// exitStatus waits for the child and reports its end the way a shell would:
// the exit code, or 128+N when signal N killed it.
func exitStatus(t *testing.T, cmd *exec.Cmd) int {
	t.Helper()
	var exitErr *exec.ExitError
	if err := cmd.Wait(); err != nil && !errors.As(err, &exitErr) {
		t.Fatal(err)
	}
	if ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return 128 + int(ws.Signal())
	}
	return cmd.ProcessState.ExitCode()
}

func runStatus(t *testing.T, name, command, profile string) int {
	t.Helper()
	return exitStatus(t, startRun(t, name, command, profile))
}

func TestRunForExitPropagatesExitCode(t *testing.T) {
	runExitDir(t)
	for cmd, want := range map[string]int{"true": 0, "exit 3": 3, "exit 255": 255, "false": 1} {
		if got := runStatus(t, "c", cmd, "core"); got != want {
			t.Errorf("%q: exit status %d, want %d", cmd, got, want)
		}
	}
}

// A command killed by a signal ends drako run the same way, so a shell
// reports 128+N.
func TestRunForExitSignalIs128PlusN(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no POSIX signals on Windows")
	}
	runExitDir(t)
	for cmd, want := range map[string]int{"kill -9 $$": 137, "kill -TERM $$": 143} {
		if got := runStatus(t, "sig", cmd, "core"); got != want {
			t.Errorf("%q: exit status %d, want %d", cmd, got, want)
		}
	}
}

// A signal sent to drako run reaches the command, as if drako were not there:
// INT too, which scripts and agents send to the one process they started.
func TestRunForExitSignalsReachTheCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no POSIX signals on Windows")
	}
	// Catching INT here starts the child with INT at its default, even when
	// this test runs with INT ignored, as a background job does.
	caught := make(chan os.Signal, 1)
	signal.Notify(caught, syscall.SIGINT)
	defer signal.Stop(caught)

	started := filepath.Join(t.TempDir(), "started")
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		runExitDir(t)
		os.Remove(started)
		cmd := startRun(t, "sig", "touch "+started+"; exec sleep 5", "core")
		for i := 0; ; i++ {
			if _, err := os.Stat(started); err == nil {
				break
			}
			if i == 500 {
				cmd.Process.Kill()
				t.Fatal("the command never started")
			}
			time.Sleep(10 * time.Millisecond)
		}
		cmd.Process.Signal(sig)
		if got, want := exitStatus(t, cmd), 128+int(sig); got != want {
			t.Errorf("%v: exit status %d, want %d", sig, got, want)
		}
	}
}

func TestRunForExitEmptyCommandErrors(t *testing.T) {
	runExitDir(t)
	for _, cmd := range []string{"", "   "} {
		if got := runStatus(t, "ghost", cmd, "core"); got != 125 {
			t.Errorf("command %q: exit status %d, want 125 (an error, not a run)", cmd, got)
		}
	}
}

func TestRunForExitRunsTheGivenCommand(t *testing.T) {
	dir := runExitDir(t)
	out := filepath.Join(dir, "..", "ran.txt")
	if got := runStatus(t, "writer", "printf hi > '"+out+"'", "core"); got != 0 {
		t.Fatalf("exit status %d, want 0", got)
	}
	if b, err := os.ReadFile(out); err != nil || string(b) != "hi" {
		t.Errorf("command did not run as given: %q, %v", b, err)
	}
}

func TestRunForExitSetsProfileEnv(t *testing.T) {
	runExitDir(t)
	if got := runStatus(t, "env", `test "$DRAKO_PROFILE" = "work"`, "work"); got != 0 {
		t.Errorf("child must see DRAKO_PROFILE=work: exit status %d", got)
	}
}

func TestRunForExitInheritsTheEnvironment(t *testing.T) {
	runExitDir(t)
	t.Setenv("DRAKO_TEST_INHERITED", "yes")
	if got := runStatus(t, "env", `test "$DRAKO_TEST_INHERITED" = "yes"`, "core"); got != 0 {
		t.Errorf("the child must inherit drako's environment: exit status %d", got)
	}
}

func TestRunForExitLogsHistory(t *testing.T) {
	dir := runExitDir(t)
	if got := runStatus(t, "🔄 Logged Cell", "true", "core"); got != 0 {
		t.Fatalf("exit status %d, want 0", got)
	}
	b, err := os.ReadFile(filepath.Join(dir, "history.log"))
	if err != nil {
		t.Fatalf("history.log not written: %v", err)
	}
	if !strings.Contains(string(b), "🔄 Logged Cell") {
		t.Errorf("history.log lacks the cell name:\n%s", b)
	}
}
