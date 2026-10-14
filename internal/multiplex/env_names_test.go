package multiplex

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A whitelist like "*" can pass names no shell would write raw (bash exports
// functions as BASH_FUNC_x%%). They must reach env as data, never as syntax.
func TestBuildScript_EnvNamesAreNotShellSyntax(t *testing.T) {
	dir := t.TempDir()
	pwned := filepath.Join(dir, "pwned")
	c := Command{
		Name: "x", Isolate: true,
		Argv: []string{"sh", "-c", `printf '%s\n' "$OK"`},
		Env:  []string{"X;touch " + pwned + "=1", "OK=fine"},
	}
	script := filepath.Join(dir, "cell.sh")
	if err := os.WriteFile(script, []byte(buildScript(c)), 0o700); err != nil {
		t.Fatal(err)
	}
	out, _ := exec.Command("sh", script).CombinedOutput()
	if _, err := os.Stat(pwned); err == nil {
		t.Fatal("an env var name was executed as shell code")
	}
	if !strings.Contains(string(out), "fine") {
		t.Errorf("the cell lost its whitelisted env: %q", out)
	}
}
