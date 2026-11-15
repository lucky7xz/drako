package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucky7xz/drako/internal/lair"
)

func luaScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "s.lua")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func runLua(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	stderr = captureStderr(t, func() {
		stdout = captureStdout(t, func() {
			code = HandleRunLuaCommand(append([]string{"drako", "run-lua"}, args...))
		})
	})
	return code, stdout, stderr
}

// The arguments reach the script as lua(1) hands them over: the arg table,
// with the script at 0, and the chunk's "...".
func TestRunLuaPassesArguments(t *testing.T) {
	path := luaScript(t, `print(arg[0] ~= nil, arg[1], arg[2], select("#", ...), ...)`)
	code, out, errOut := runLua(t, path, "-y", "two words")
	if code != 0 || errOut != "" {
		t.Fatalf("code %d, stderr %q", code, errOut)
	}
	if want := "true\t-y\ttwo words\t2\t-y\ttwo words\n"; out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

// The interpreter is stock: the standard libraries are open.
func TestRunLuaHasTheStandardLibraries(t *testing.T) {
	path := luaScript(t, `assert(os.getenv and io.open and string.rep and table.concat and math.floor)`)
	if code, _, errOut := runLua(t, path); code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

// A first line starting with # is skipped, so a script may carry a shebang.
func TestRunLuaSkipsAShebang(t *testing.T) {
	path := luaScript(t, "#!/usr/bin/env drako run-lua\nprint('ok')\n")
	if code, out, _ := runLua(t, path); code != 0 || out != "ok\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestRunLuaFailures(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.lua")
	for name, c := range map[string]struct {
		args []string
		want string
	}{
		"no script":     {nil, "drako run-lua: no script given. Usage: drako run-lua <script.lua> [args...]\n"},
		"missing file":  {[]string{missing}, "drako run-lua: cannot read " + missing + ": no such file or directory\n"},
		"runtime error": {[]string{luaScript(t, "local x\nx()\n")}, "s.lua:2: attempt to call"},
		// Lua 5.1 has no "//": the message names the file and the line.
		"syntax error": {[]string{luaScript(t, "print(7 // 2)\n")}, "s.lua line:1"},
	} {
		code, out, errOut := runLua(t, c.args...)
		if code != 1 {
			t.Errorf("%s: exit %d, want 1", name, code)
		}
		if out != "" {
			t.Errorf("%s: wrote to stdout: %q", name, out)
		}
		if !strings.HasPrefix(errOut, "drako run-lua: ") || !strings.Contains(errOut, c.want) || !strings.HasSuffix(errOut, "\n") || strings.HasSuffix(errOut, "\n\n") {
			t.Errorf("%s: stderr %q, want it to contain %q and end in one newline", name, errOut, c.want)
		}
	}
}

// run-lua needs no config folder, so it is not held to drako's layout check.
func TestRunLuaIsNotAFolderCommand(t *testing.T) {
	if worksInFolder([]string{"drako", "run-lua", "x.lua"}) {
		t.Error("run-lua must work without drako's folder")
	}
}

// The shipped example (luadeck_example/demo.lua) writes profiles drako
// accepts: one per flag, both with none, and never over a file that exists.
func TestLuadeckExample(t *testing.T) {
	dir := t.TempDir()
	if _, err := lair.RestoreBootstrap(dir); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	grid := filepath.Join("inventory", "demo-grid.profile.toml")
	hello := filepath.Join("inventory", "demo-hello.profile.toml")
	script := filepath.Join("luadeck_example", "demo.lua")

	if code, out, errOut := runLua(t, script, "--grid"); code != 0 || !strings.Contains(out, "wrote") {
		t.Fatalf("--grid: code %d, out %q, stderr %q", code, out, errOut)
	}
	if _, err := os.Stat(hello); err == nil {
		t.Fatal("--grid also wrote demo-hello")
	}

	// A file that is there is kept as it is.
	if err := os.WriteFile(grid, []byte("X = 1\nY = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := runLua(t, script)
	if code != 0 || !strings.Contains(out, "kept") || !strings.Contains(out, "wrote") {
		t.Fatalf("no flags: code %d, out %q, stderr %q", code, out, errOut)
	}
	if b, _ := os.ReadFile(grid); string(b) != "X = 1\nY = 1\n" {
		t.Error("an existing file was overwritten")
	}

	// It finds the inventory from its own place, not from the current folder.
	if err := os.Remove(hello); err != nil {
		t.Fatal(err)
	}
	t.Chdir("luadeck_example")
	if code, _, errOut := runLua(t, "demo.lua", "--hello"); code != 0 {
		t.Fatalf("from inside its folder: code %d, stderr %q", code, errOut)
	}
	t.Chdir(t.TempDir())
	if err := os.Remove(filepath.Join(dir, hello)); err != nil {
		t.Fatalf("run from inside its folder, it did not write demo-hello: %v", err)
	}
	if code, _, errOut := runLua(t, filepath.Join(dir, script), "--hello"); code != 0 {
		t.Fatalf("by full path from elsewhere: code %d, stderr %q", code, errOut)
	}
	t.Chdir(dir)

	// What it writes passes drako check.
	if err := os.Remove(grid); err != nil {
		t.Fatal(err)
	}
	runLua(t, script, "--grid")
	var checked int
	captureStdout(t, func() { checked = HandleCheckCommand([]string{"drako", "check", grid, hello}) })
	if checked != 0 {
		t.Errorf("drako check on the generated profiles exits %d", checked)
	}
}
