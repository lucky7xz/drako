package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestRunForExitHonorsEnvWhitelist(t *testing.T) {
	runExitDir(t)
	t.Setenv("DRAKO_TEST_SECRET", "leak")
	cfg := config.Config{DefaultShell: "sh", EnvWhitelist: []string{"PATH", "HOME"}}
	code, err := RunForExit(cfg, "env", `test -z "$DRAKO_TEST_SECRET"`, "core")
	if err != nil || code != 0 {
		t.Errorf("whitelist must keep DRAKO_TEST_SECRET out of the child: (%d, %v)", code, err)
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
