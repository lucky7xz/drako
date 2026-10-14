package multiplex

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// fakeShell is a pane's login shell: for readyAfter polls it drops or queues
// typed lines, then runs them in order; lines behind an exec reach the cell.
type fakeShell struct {
	queue    []string
	polls    int
	running  string
	output   string
	typedIn  int
	sentExec int
	probes   int
}

var probeRe = regexp.MustCompile(`printf 'DRAKO-%s\\n' '([^']*)'`)

func shellHerdr(t *testing.T, readyAfter int, drops bool) map[string]*fakeShell {
	t.Helper()
	oldExec, oldSleep := herdrExec, herdrSleep
	t.Cleanup(func() { herdrExec, herdrSleep = oldExec, oldSleep })
	herdrSleep = func(time.Duration) {}

	panes := map[string]*fakeShell{}
	n := 0
	step := func(p *fakeShell) {
		p.polls++
		for p.polls >= readyAfter && len(p.queue) > 0 {
			line := p.queue[0]
			p.queue = p.queue[1:]
			if p.running != "" {
				p.typedIn++
				continue
			}
			if m := probeRe.FindStringSubmatch(line); m != nil {
				p.output += "DRAKO-" + m[1] + "\n"
			} else if s, ok := strings.CutPrefix(line, "exec sh "); ok {
				p.running = strings.Trim(s, "'")
			}
		}
	}
	herdrExec = func(argv []string) ([]byte, error) {
		switch argv[2] {
		case "create", "split":
			n++
			id := fmt.Sprintf("w1:p%d", n)
			panes[id] = &fakeShell{}
			field := "pane"
			if argv[1] == "tab" {
				field = "root_pane"
			}
			return fmt.Appendf(nil, `{"result":{"%s":{"pane_id":"%s"}}}`, field, id), nil
		case "run":
			p := panes[argv[3]]
			line := strings.Join(argv[4:], " ")
			if strings.HasPrefix(line, "exec ") {
				p.sentExec++
			} else {
				p.probes++
			}
			if p.running != "" {
				p.typedIn++
			} else if !(drops && p.polls < readyAfter) {
				p.queue = append(p.queue, line)
			}
			return []byte(`{"result":{}}`), nil
		case "wait-output":
			id := argv[len(argv)-1]
			p := panes[id]
			for range 20 { // the --timeout window, in poll ticks
				step(p)
				for i, a := range argv {
					if a == "--match" && strings.Contains(p.output, argv[i+1]) {
						return []byte(`{"result":{}}`), nil
					}
				}
			}
			return nil, fmt.Errorf("timed out")
		case "process-info":
			p := panes[argv[len(argv)-1]]
			step(p)
			if p.running != "" {
				return fmt.Appendf(nil, `{"result":{"process_info":{"foreground_processes":[{"cmdline":"/bin/sh %s"}]}}}`, p.running), nil
			}
			return []byte(`{"result":{"process_info":{"foreground_processes":[{"cmdline":"/bin/bash"}]}}}`), nil
		}
		return []byte(`{"result":{}}`), nil
	}
	return panes
}

// A slow login profile that queues typeahead (most shells do) must not end
// with a resent start command typed into the running cell.
func TestHerdr_SlowShellGetsNothingTypedIntoTheCell(t *testing.T) {
	panes := shellHerdr(t, 25, false) // ~1.25s of profile, input kept
	if err := Launch(NewHerdr(), "drako-t", cells(2), []int{2}, filepath.Join(t.TempDir(), "batch")); err != nil {
		t.Fatal(err)
	}
	for id, p := range panes {
		if p.running == "" || p.typedIn != 0 {
			t.Errorf("pane %s: running %q, %d line(s) typed into the cell", id, p.running, p.typedIn)
		}
	}
}

// A profile that drops typed lines (a slow MOTD, as on Bazzite) still gets its
// cell started, with exec sent exactly once.
func TestHerdr_ShellThatDropsTypingStillStartsTheCell(t *testing.T) {
	panes := shellHerdr(t, 25, true)
	if err := Launch(NewHerdr(), "drako-t", cells(2), []int{2}, filepath.Join(t.TempDir(), "batch")); err != nil {
		t.Fatal(err)
	}
	for id, p := range panes {
		if p.running == "" || p.sentExec != 1 || p.typedIn != 0 {
			t.Errorf("pane %s: running %q, exec sent %d times, %d typed into the cell", id, p.running, p.sentExec, p.typedIn)
		}
	}
}

// A shell that is already reading costs one probe and one exec, nothing more.
func TestHerdr_ReadyShellGetsOneProbeAndOneExec(t *testing.T) {
	panes := shellHerdr(t, 0, false)
	if err := Launch(NewHerdr(), "drako-t", cells(2), []int{2}, filepath.Join(t.TempDir(), "batch")); err != nil {
		t.Fatal(err)
	}
	for id, p := range panes {
		if p.probes != 1 || p.sentExec != 1 {
			t.Errorf("pane %s: %d probe(s), %d exec(s); want 1 and 1", id, p.probes, p.sentExec)
		}
	}
}

// A shell that never reads is reported rather than retried forever.
func TestHerdr_GivesUpAndSaysSo(t *testing.T) {
	shellHerdr(t, 1<<30, true)
	err := Launch(NewHerdr(), "drako-t", cells(1), []int{1}, filepath.Join(t.TempDir(), "batch"))
	if err == nil || !strings.Contains(err.Error(), "did not start") {
		t.Fatalf("a cell that never starts must say so, got %v", err)
	}
}

// Only a pane that is gone counts as a cell that ran and closed; a herdr call
// that merely failed proves nothing and must not be reported as a start.
func TestHerdr_FailedProcessInfoIsNotAStart(t *testing.T) {
	shellHerdr(t, 0, false)
	inner := herdrExec
	herdrExec = func(argv []string) ([]byte, error) {
		if argv[2] == "process-info" {
			return nil, fmt.Errorf("pane process-info: exit status 1: server busy")
		}
		return inner(argv)
	}
	err := Launch(NewHerdr(), "drako-t", cells(1), []int{1}, filepath.Join(t.TempDir(), "batch"))
	if err == nil {
		t.Error("an unconfirmed cell was reported as started")
	}
}
