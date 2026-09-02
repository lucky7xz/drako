package multiplex

import (
	"strings"
	"testing"
)

// The command is wrapped for the configured shell with POSIX single-quote
// escaping — it is never interpolated into a multiplexer argument, which is
// the classic batch footgun.
func TestBuildScript_QuotesTheCommand(t *testing.T) {
	content := buildScript(Command{
		Name:   "tricky",
		Script: `echo 'hi' && say "there" | grep $HOME`,
		Shell:  "bash",
	})
	if !strings.Contains(content, `bash -lc 'echo '\''hi'\'' && say "there" | grep $HOME'`) {
		t.Errorf("command not safely quoted:\n%s", content)
	}
	if !strings.HasPrefix(content, "#!/bin/sh\n") {
		t.Errorf("script must be plain sh, got:\n%s", content)
	}
}

func TestBuildScript_KeepOpenPausesBeforeThePaneCloses(t *testing.T) {
	open := buildScript(Command{Name: "open", Script: "ls", Shell: "bash", KeepOpen: true})
	if !strings.Contains(open, "read ") {
		t.Errorf("KeepOpen script must pause for Enter:\n%s", open)
	}
	closed := buildScript(Command{Name: "closed", Script: "ls", Shell: "bash"})
	if strings.Contains(closed, "read ") {
		t.Errorf("auto-close script must exit with its command:\n%s", closed)
	}
}

func TestBuildScript_EnvExportedOnTopOfInherited(t *testing.T) {
	content := buildScript(Command{
		Name: "cell", Script: "ls", Shell: "bash",
		Env: []string{"DRAKO_PROFILE=work"},
	})
	if !strings.Contains(content, "export DRAKO_PROFILE='work'\n") {
		t.Errorf("env entry must be exported before the command:\n%s", content)
	}
	if strings.Contains(content, "env -i") {
		t.Errorf("without Isolate the inherited environment stays:\n%s", content)
	}
}

func TestBuildScript_IsolateReplacesTheEnvironment(t *testing.T) {
	content := buildScript(Command{
		Name: "cell", Script: "ls", Shell: "bash",
		Env: []string{"PATH=/bin", "DRAKO_PROFILE=work"}, Isolate: true,
	})
	if !strings.Contains(content, "env -i PATH='/bin' DRAKO_PROFILE='work' bash -lc 'ls'") {
		t.Errorf("isolated cell must run under env -i with exactly Env:\n%s", content)
	}
	if strings.Contains(content, "export ") {
		t.Errorf("isolated cell needs no exports:\n%s", content)
	}
}

func TestBuildScript_EnvValuesQuoted(t *testing.T) {
	content := buildScript(Command{
		Name: "cell", Script: "ls", Shell: "bash",
		Env: []string{`DRAKO_PROFILE=it's here`},
	})
	if !strings.Contains(content, `export DRAKO_PROFILE='it'\''s here'`) {
		t.Errorf("env values must be single-quote escaped:\n%s", content)
	}
}

// The filename is handed to a multiplexer as a bare word, so it must survive
// both a path and a shell line.
func TestScriptName_IsPathAndShellSafe(t *testing.T) {
	name := scriptName(0, Command{Name: "🧹 Weird / Name ⋮", Script: "ls", Shell: "bash"})
	if strings.ContainsAny(name, "/ \t'\"") {
		t.Errorf("script filename must be path- and quote-safe: %q", name)
	}
	if !strings.HasPrefix(name, "01-") {
		t.Errorf("script filenames are ordered by cell: %q", name)
	}
}
