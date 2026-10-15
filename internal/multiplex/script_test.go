package multiplex

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The command is wrapped for the configured shell with POSIX single-quote
// escaping — it is never interpolated into a multiplexer argument, which is
// the classic batch footgun.
func TestBuildScript_QuotesTheCommand(t *testing.T) {
	content := buildScript(Command{
		Name: "tricky",
		Argv: []string{"bash", "-lc", `echo 'hi' && say "there" | grep $HOME`},
	})
	if !strings.Contains(content, `bash -lc 'echo '\''hi'\'' && say "there" | grep $HOME'`) {
		t.Errorf("command not safely quoted:\n%s", content)
	}
	if !strings.HasPrefix(content, "#!/bin/sh\n") {
		t.Errorf("script must be plain sh, got:\n%s", content)
	}
}

func TestBuildScript_KeepOpenPausesBeforeThePaneCloses(t *testing.T) {
	open := buildScript(Command{Name: "open", Argv: []string{"bash", "-lc", "ls"}, KeepOpen: true})
	if !strings.Contains(open, "read ") {
		t.Errorf("KeepOpen script must pause for Enter:\n%s", open)
	}
	closed := buildScript(Command{Name: "closed", Argv: []string{"bash", "-lc", "ls"}})
	if strings.Contains(closed, "read ") {
		t.Errorf("auto-close script must exit with its command:\n%s", closed)
	}
}

// The deck is the one variable a cell adds; the pane keeps its own environment.
func TestBuildScript_ExportsTheProfile(t *testing.T) {
	ls := []string{"bash", "-lc", "ls"}
	if got := buildScript(Command{Name: "cell", Argv: ls, Profile: "work"}); !strings.Contains(got, "export DRAKO_PROFILE='work'\n") {
		t.Errorf("the profile must be exported before the command:\n%s", got)
	}
	if got := buildScript(Command{Name: "cell", Argv: ls}); strings.Contains(got, "DRAKO_PROFILE") {
		t.Errorf("without a profile nothing is exported:\n%s", got)
	}
}

// A deck name is file-name text: it must reach the cell as data, never as shell.
func TestBuildScript_ProfileIsDataNotShell(t *testing.T) {
	dir := t.TempDir()
	pwned := filepath.Join(dir, "pwned")
	profile := "it's $(touch " + pwned + ")"
	c := Command{Name: "x", Argv: []string{"sh", "-c", `printf '%s' "$DRAKO_PROFILE"`}, Profile: profile}
	script := filepath.Join(dir, "cell.sh")
	if err := os.WriteFile(script, []byte(buildScript(c)), 0o700); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("sh", script).CombinedOutput()
	if err != nil {
		t.Fatalf("the cell failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(pwned); err == nil {
		t.Error("the deck name ran as shell code")
	}
	if string(out) != profile {
		t.Errorf("the cell saw DRAKO_PROFILE = %q, want %q", out, profile)
	}
}

// The filename is handed to a multiplexer as a bare word, so it must survive
// both a path and a shell line.
func TestScriptName_IsPathAndShellSafe(t *testing.T) {
	name := scriptName(0, Command{Name: "🧹 Weird / Name ⋮", Argv: []string{"bash", "-lc", "ls"}})
	if strings.ContainsAny(name, "/ \t'\"") {
		t.Errorf("script filename must be path- and quote-safe: %q", name)
	}
	if !strings.HasPrefix(name, "01-") {
		t.Errorf("script filenames are ordered by cell: %q", name)
	}
}

// A cell runs exactly the command line a single run would (core.ShellArgv): no
// login flag added for sh or fish, and pwsh is not swapped for bash.
func TestBuildScript_RunsTheGivenCommandLine(t *testing.T) {
	for _, argv := range [][]string{
		{"sh", "-c", "ls"},
		{"fish", "-c", "ls"},
		{"pwsh", "-NoLogo", "-NoProfile", "-Command", "ls"},
	} {
		want := strings.Join(argv[:len(argv)-1], " ") + " 'ls'\n"
		if got := buildScript(Command{Name: "c", Argv: argv}); !strings.Contains(got, want) {
			t.Errorf("%s: script runs\n%s\nwant a line ending %q", argv[0], got, want)
		}
	}
}
