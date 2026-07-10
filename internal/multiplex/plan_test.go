package multiplex

import (
	"fmt"
	"strings"
	"testing"
)

func cells(n int) []Command {
	out := make([]Command, n)
	for i := range out {
		out[i] = Command{
			Name:   fmt.Sprintf("cell %d", i+1),
			Script: fmt.Sprintf("echo %d", i+1),
			Shell:  "bash",
		}
	}
	return out
}

// stepVerbs extracts the tmux subcommand of every step for shape assertions.
func stepVerbs(s Session) []string {
	verbs := make([]string, len(s.Steps))
	for i, step := range s.Steps {
		if len(step) < 2 || step[0] != "tmux" {
			verbs[i] = "INVALID:" + strings.Join(step, " ")
			continue
		}
		verbs[i] = step[1]
	}
	return verbs
}

func wantVerbs(t *testing.T, s Session, want ...string) {
	t.Helper()
	got := stepVerbs(s)
	if len(got) != len(want) {
		t.Fatalf("steps = %v, want verbs %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("step %d = %q, want %q (all: %v)", i, got[i], want[i], got)
		}
	}
}

func TestPlan_SingleCommand(t *testing.T) {
	s, err := Plan("drako-1", cells(1), false, "/tmp/x")
	if err != nil {
		t.Fatal(err)
	}
	wantVerbs(t, s, "new-session", "attach-session")
	if !s.Attach {
		t.Error("outside tmux, the last step hands the terminal over")
	}
	if len(s.Scripts) != 1 {
		t.Fatalf("want one script, got %v", s.Scripts)
	}
	if !strings.Contains(strings.Join(s.Steps[0], " "), "/tmp/x/") {
		t.Errorf("new-session must run the script by its full path: %v", s.Steps[0])
	}
}

func TestPlan_FourCommandsTiledPanes(t *testing.T) {
	s, err := Plan("drako-1", cells(4), false, "/tmp/x")
	if err != nil {
		t.Fatal(err)
	}
	wantVerbs(t, s, "new-session", "split-window", "split-window", "split-window",
		"select-layout", "attach-session")
	last := s.Steps[4]
	if last[len(last)-1] != "tiled" {
		t.Errorf("pane path must arrange tiled, got %v", last)
	}
}

func TestPlan_FiveCommandsNamedWindows(t *testing.T) {
	s, err := Plan("drako-1", cells(5), false, "/tmp/x")
	if err != nil {
		t.Fatal(err)
	}
	wantVerbs(t, s, "new-session", "new-window", "new-window", "new-window",
		"new-window", "attach-session")
	// Windows carry the cell name so the tab bar is readable.
	found := false
	for _, tok := range s.Steps[1] {
		if tok == "cell 2" {
			found = true
		}
	}
	if !found {
		t.Errorf("new-window should be named after its cell: %v", s.Steps[1])
	}
}

func TestPlan_InsideTmuxNested(t *testing.T) {
	s, err := Plan("drako-1", cells(3), true, "/tmp/x")
	if err != nil {
		t.Fatal(err)
	}
	// Current session: every command splits, no new-session, no attach.
	wantVerbs(t, s, "split-window", "split-window", "split-window", "select-layout")
	if s.Attach {
		t.Error("nested launch must not attach (drako already owns a pane)")
	}
}

func TestPlan_InsideTmuxManyWindows(t *testing.T) {
	s, err := Plan("drako-1", cells(6), true, "/tmp/x")
	if err != nil {
		t.Fatal(err)
	}
	wantVerbs(t, s, "new-window", "new-window", "new-window", "new-window",
		"new-window", "new-window")
}

func TestPlan_CapAndEmpty(t *testing.T) {
	if _, err := Plan("s", cells(10), false, "/tmp/x"); err == nil {
		t.Error("more than 9 commands must be rejected")
	}
	if _, err := Plan("s", nil, false, "/tmp/x"); err == nil {
		t.Error("zero commands must be rejected")
	}
}

func TestPlan_ScriptQuoting(t *testing.T) {
	cmds := []Command{{
		Name:   "tricky",
		Script: `echo 'hi' && say "there" | grep $HOME`,
		Shell:  "bash",
	}}
	s, err := Plan("drako-1", cmds, false, "/tmp/x")
	if err != nil {
		t.Fatal(err)
	}
	var content string
	for _, c := range s.Scripts {
		content = c
	}
	// The command is wrapped for the configured shell with POSIX single-quote
	// escaping — never interpolated into a tmux argument.
	if !strings.Contains(content, `bash -lc 'echo '\''hi'\'' && say "there" | grep $HOME'`) {
		t.Errorf("command not safely quoted:\n%s", content)
	}
	if !strings.HasPrefix(content, "#!/bin/sh\n") {
		t.Errorf("script must be plain sh, got:\n%s", content)
	}
}

func TestPlan_KeepOpenAppendsPause(t *testing.T) {
	cmds := []Command{
		{Name: "open", Script: "ls", Shell: "bash", KeepOpen: true},
		{Name: "closed", Script: "ls", Shell: "bash"},
	}
	s, err := Plan("drako-1", cmds, false, "/tmp/x")
	if err != nil {
		t.Fatal(err)
	}
	openContent, closedContent := "", ""
	for name, c := range s.Scripts {
		if strings.Contains(name, "open") && !strings.Contains(name, "closed") {
			openContent = c
		}
		if strings.Contains(name, "closed") {
			closedContent = c
		}
	}
	if !strings.Contains(openContent, "read ") {
		t.Errorf("KeepOpen script must pause for Enter:\n%s", openContent)
	}
	if strings.Contains(closedContent, "read ") {
		t.Errorf("auto-close script must exit with its command:\n%s", closedContent)
	}
}

func TestPlan_SanitizedFilenames(t *testing.T) {
	cmds := []Command{{Name: "🧹 Weird / Name ⋮", Script: "ls", Shell: "bash"}}
	s, err := Plan("drako-1", cmds, false, "/tmp/x")
	if err != nil {
		t.Fatal(err)
	}
	for name := range s.Scripts {
		if strings.ContainsAny(name, "/ \t'\"") {
			t.Errorf("script filename must be path- and quote-safe: %q", name)
		}
	}
}
