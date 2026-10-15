package core

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/lucky7xz/drako/internal/config"
)

// runExitDir isolates the config dir so history.log lands in a temp dir.
func runExitDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)
	return filepath.Join(tmp, "drako")
}

var shCfg = config.Config{DefaultShell: "sh"}

func TestRunForExitPropagatesExitCode(t *testing.T) {
	runExitDir(t)
	for cmd, want := range map[string]int{"true": 0, "exit 3": 3, "exit 255": 255, "false": 1} {
		code, err := RunForExit(shCfg, "c", cmd, "core")
		if err != nil || code != want {
			t.Errorf("%q: got (%d, %v), want (%d, nil)", cmd, code, err, want)
		}
	}
}

// A command killed by a signal reports 128+N, as a shell would, rather than
// Go's -1 (which os.Exit turns into 255).
func TestRunForExitSignalIs128PlusN(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no POSIX signals on Windows")
	}
	runExitDir(t)
	for cmd, want := range map[string]int{"kill -9 $$": 137, "kill -TERM $$": 143} {
		code, err := RunForExit(shCfg, "sig", cmd, "core")
		if err != nil || code != want {
			t.Errorf("%q: got (%d, %v), want (%d, nil)", cmd, code, err, want)
		}
	}
}

// Signals sent to drako while it waits: TERM reaches the command, INT is left
// to the terminal, and drako stays to report the command's own exit code.
func TestRunForExitRelaysSignals(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no POSIX signals on Windows")
	}
	dir := t.TempDir()
	started := filepath.Join(dir, "started")
	tests := []struct {
		sig  syscall.Signal
		cmd  string
		want int
	}{
		{syscall.SIGTERM, "touch " + started + "; exec sleep 5", 143},
		{syscall.SIGINT, "touch " + started + "; sleep 1; exit 7", 7},
	}
	for _, tt := range tests {
		runExitDir(t)
		os.Remove(started)
		go func() {
			for i := 0; i < 200; i++ {
				if _, err := os.Stat(started); err == nil {
					self, _ := os.FindProcess(os.Getpid())
					self.Signal(tt.sig)
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()
		code, err := RunForExit(shCfg, "sig", tt.cmd, "core")
		if err != nil || code != tt.want {
			t.Errorf("%v: got (%d, %v), want (%d, nil)", tt.sig, code, err, tt.want)
		}
	}
}

func TestRunForExitEmptyCommandErrors(t *testing.T) {
	runExitDir(t)
	for _, cmd := range []string{"", "   "} {
		if _, err := RunForExit(shCfg, "ghost", cmd, "core"); err == nil {
			t.Errorf("command %q must error, not run", cmd)
		}
	}
}

func TestRunForExitRunsTheGivenCommand(t *testing.T) {
	dir := runExitDir(t)
	out := filepath.Join(dir, "..", "ran.txt")
	if _, err := RunForExit(shCfg, "writer", "printf hi > '"+out+"'", "core"); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(out); err != nil || string(b) != "hi" {
		t.Errorf("command did not run as given: %q, %v", b, err)
	}
}

func TestRunForExitSetsProfileEnv(t *testing.T) {
	runExitDir(t)
	code, err := RunForExit(shCfg, "env", `test "$DRAKO_PROFILE" = "work"`, "work")
	if err != nil || code != 0 {
		t.Errorf("child must see DRAKO_PROFILE=work: (%d, %v)", code, err)
	}
}

func TestRunForExitInheritsTheEnvironment(t *testing.T) {
	runExitDir(t)
	t.Setenv("DRAKO_TEST_INHERITED", "yes")
	code, err := RunForExit(shCfg, "env", `test "$DRAKO_TEST_INHERITED" = "yes"`, "core")
	if err != nil || code != 0 {
		t.Errorf("the child must inherit drako's environment: (%d, %v)", code, err)
	}
}

func TestRunForExitLogsHistory(t *testing.T) {
	dir := runExitDir(t)
	if _, err := RunForExit(shCfg, "🔄 Logged Cell", "true", "core"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "history.log"))
	if err != nil {
		t.Fatalf("history.log not written: %v", err)
	}
	if !strings.Contains(string(b), "🔄 Logged Cell") {
		t.Errorf("history.log lacks the cell name:\n%s", b)
	}
}
