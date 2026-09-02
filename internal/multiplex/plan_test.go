package multiplex

import (
	"fmt"
	"slices"
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
	s, err := Plan("drako-1", cells(1), []int{1}, false, "/tmp/x")
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

// One tab holding every cell: the session's first window, then a split per
// extra cell, arranged tiled.
func TestPlan_OneTabOfFourPanes(t *testing.T) {
	s, err := Plan("drako-1", cells(4), []int{4}, false, "/tmp/x")
	if err != nil {
		t.Fatal(err)
	}
	wantVerbs(t, s, "new-session", "split-window", "split-window", "split-window",
		"select-layout", "attach-session")
	last := s.Steps[4]
	if last[len(last)-1] != "tiled" {
		t.Errorf("a multi-pane tab must be arranged tiled, got %v", last)
	}
}

// Five cells used to explode into five windows. The vector decides now, so
// [4 1] is two tabs — four side by side, then one.
func TestPlan_TwoTabsFromFiveCells(t *testing.T) {
	s, err := Plan("drako-1", cells(5), []int{4, 1}, false, "/tmp/x")
	if err != nil {
		t.Fatal(err)
	}
	wantVerbs(t, s, "new-session", "split-window", "split-window", "split-window",
		"select-layout", "new-window", "attach-session")
	// A one-pane tab needs no layout step.
	if strings.Contains(strings.Join(s.Steps[5], " "), "tiled") {
		t.Errorf("a single-pane tab must not be arranged: %v", s.Steps[5])
	}
}

// A tab holds several cells, so it is named after all of them — the identity a
// pane cannot carry.
func TestPlan_TabsAreNamedAfterTheirCells(t *testing.T) {
	s, err := Plan("drako-1", cells(5), []int{4, 1}, false, "/tmp/x")
	if err != nil {
		t.Fatal(err)
	}
	first := strings.Join(s.Steps[0], " ")
	if !strings.Contains(first, "cell 1·cell 2·cell 3·cell 4") {
		t.Errorf("the first tab must be named after its four cells: %v", s.Steps[0])
	}
	if second := strings.Join(s.Steps[5], " "); !strings.Contains(second, "cell 5") {
		t.Errorf("the second tab must be named after its cell: %v", s.Steps[5])
	}
}

// Inside tmux the batch gets its own windows; drako's pane is never split.
func TestPlan_InsideTmuxMakesItsOwnTabs(t *testing.T) {
	s, err := Plan("drako-1", cells(3), []int{3}, true, "/tmp/x")
	if err != nil {
		t.Fatal(err)
	}
	wantVerbs(t, s, "new-window", "split-window", "split-window", "select-layout")
	if s.Attach {
		t.Error("nested launch must not attach (drako already owns a pane)")
	}
	// No -t: the steps join the session drako is already in.
	for _, step := range s.Steps {
		if slices.Contains(step, "-t") {
			t.Errorf("a nested step must not target a session by name: %v", step)
		}
	}
}

func TestPlan_InsideTmuxTwoTabs(t *testing.T) {
	s, err := Plan("drako-1", cells(6), []int{4, 2}, true, "/tmp/x")
	if err != nil {
		t.Fatal(err)
	}
	wantVerbs(t, s, "new-window", "split-window", "split-window", "split-window",
		"select-layout", "new-window", "split-window", "select-layout")
}

func TestPlan_CapAndEmpty(t *testing.T) {
	if _, err := Plan("s", cells(10), []int{4, 4, 2}, false, "/tmp/x"); err == nil {
		t.Error("more than 9 commands must be rejected")
	}
	if _, err := Plan("s", nil, nil, false, "/tmp/x"); err == nil {
		t.Error("zero commands must be rejected")
	}
}

// The vector and the cells must agree, or panes would be dropped or invented.
func TestPlan_RejectsATabVectorThatMissesCells(t *testing.T) {
	for _, tabs := range [][]int{nil, {3}, {5}, {2, 3}, {4, 0, 1}} {
		if _, err := Plan("s", cells(4), tabs, false, "/tmp/x"); err == nil {
			t.Errorf("tabs %v does not lay out 4 cells, want an error", tabs)
		}
	}
	// Any vector that does add up is accepted, however it splits them.
	for _, tabs := range [][]int{{4}, {2, 2}, {1, 1, 1, 1}} {
		if _, err := Plan("s", cells(4), tabs, false, "/tmp/x"); err != nil {
			t.Errorf("tabs %v lays out 4 cells, got %v", tabs, err)
		}
	}
}

func TestPlan_ScriptQuoting(t *testing.T) {
	cmds := []Command{{
		Name:   "tricky",
		Script: `echo 'hi' && say "there" | grep $HOME`,
		Shell:  "bash",
	}}
	s, err := Plan("drako-1", cmds, []int{len(cmds)}, false, "/tmp/x")
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
	s, err := Plan("drako-1", cmds, []int{len(cmds)}, false, "/tmp/x")
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

func scriptFor(t *testing.T, c Command) string {
	t.Helper()
	s, err := Plan("drako-1", []Command{c}, []int{1}, false, "/tmp/x")
	if err != nil {
		t.Fatal(err)
	}
	for _, content := range s.Scripts {
		return content
	}
	t.Fatal("no script produced")
	return ""
}

func TestPlan_EnvExportedOnTopOfInherited(t *testing.T) {
	content := scriptFor(t, Command{
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

func TestPlan_IsolateReplacesEnvironment(t *testing.T) {
	content := scriptFor(t, Command{
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

func TestPlan_EnvValuesQuoted(t *testing.T) {
	content := scriptFor(t, Command{
		Name: "cell", Script: "ls", Shell: "bash",
		Env: []string{`DRAKO_PROFILE=it's here`},
	})
	if !strings.Contains(content, `export DRAKO_PROFILE='it'\''s here'`) {
		t.Errorf("env values must be single-quote escaped:\n%s", content)
	}
}

func TestPlan_SanitizedFilenames(t *testing.T) {
	cmds := []Command{{Name: "🧹 Weird / Name ⋮", Script: "ls", Shell: "bash"}}
	s, err := Plan("drako-1", cmds, []int{len(cmds)}, false, "/tmp/x")
	if err != nil {
		t.Fatal(err)
	}
	for name := range s.Scripts {
		if strings.ContainsAny(name, "/ \t'\"") {
			t.Errorf("script filename must be path- and quote-safe: %q", name)
		}
	}
}
