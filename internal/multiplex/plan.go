// Package multiplex builds tmux launch plans for drako's batch mode. It
// follows the same discipline as core's buildExecPlan: Plan constructs — it
// never executes, touches the filesystem, or reads process state. The risky
// parts of batch launching (quoting, nested-tmux, layout) all live here where
// they are table-testable.
package multiplex

import (
	"fmt"
	"path/filepath"
	"strings"
)

// MaxCommands mirrors drako's 1-9 idiom (quick-nav, Alt+1-9).
const MaxCommands = 9

// Command is one grid cell to launch.
type Command struct {
	Name     string // cell name — becomes the window name and script name
	Script   string // the resolved command string, verbatim from the profile
	Shell    string // the profile's shell, invoked like single-run (-lc)
	KeepOpen bool   // auto_close_execution = false: pause before the pane closes

	// Env entries applied to this cell's command. Isolate replaces the
	// inherited environment with exactly Env (env -i); otherwise Env is
	// exported on top of it.
	Env     []string
	Isolate bool
}

// Session is a complete, inert launch plan: the scripts to write and the tmux
// invocations to run, in order. Nothing has happened yet when Plan returns.
type Session struct {
	Name    string
	Scripts map[string]string // filename (inside the script dir) → content
	Steps   [][]string        // argv slices, executed in order
	Attach  bool              // the last step attaches: hand it the terminal
}

// Plan lays out the tmux session for cmds. tabs says how many cells each tab
// holds, in order, and must account for every cell — the layout is decided
// before Plan is called, not derived from the cell count here. insideTmux means
// drako already runs inside a session ($TMUX set): then the tabs join the
// current session and there is no attach step. scriptDir is where the caller
// will write the scripts; only paths are computed here.
func Plan(session string, cmds []Command, tabs []int, insideTmux bool, scriptDir string) (Session, error) {
	if len(cmds) == 0 {
		return Session{}, fmt.Errorf("nothing to launch")
	}
	if len(cmds) > MaxCommands {
		return Session{}, fmt.Errorf("at most %d commands per batch (got %d)", MaxCommands, len(cmds))
	}
	if err := checkTabs(tabs, len(cmds)); err != nil {
		return Session{}, err
	}

	s := Session{Name: session, Scripts: map[string]string{}}

	paths := make([]string, len(cmds))
	for i, c := range cmds {
		filename := fmt.Sprintf("%02d-%s.sh", i+1, sanitizeName(c.Name))
		s.Scripts[filename] = buildScript(c)
		paths[i] = filepath.Join(scriptDir, filename)
	}

	// Outside tmux every step names the session we are building; inside it,
	// nothing is targeted and the steps land in the session drako is in.
	var target []string
	if !insideTmux {
		target = []string{"-t", session}
	}

	cell := 0
	for t, panes := range tabs {
		group := cmds[cell : cell+panes]
		switch {
		case !insideTmux && t == 0:
			s.Steps = append(s.Steps, []string{"tmux", "new-session", "-d", "-s", session, "-n", tabLabel(group), paths[cell]})
		default:
			step := append([]string{"tmux", "new-window"}, target...)
			s.Steps = append(s.Steps, append(step, "-n", tabLabel(group), paths[cell]))
		}
		for i := 1; i < panes; i++ {
			step := append([]string{"tmux", "split-window"}, target...)
			s.Steps = append(s.Steps, append(step, paths[cell+i]))
		}
		if panes > 1 {
			step := append([]string{"tmux", "select-layout"}, target...)
			s.Steps = append(s.Steps, append(step, "tiled"))
		}
		cell += panes
	}

	if !insideTmux {
		s.Steps = append(s.Steps, []string{"tmux", "attach-session", "-t", session})
		s.Attach = true
	}
	return s, nil
}

// checkTabs rejects a vector that would drop or invent panes.
func checkTabs(tabs []int, cells int) error {
	total := 0
	for _, panes := range tabs {
		if panes < 1 {
			return fmt.Errorf("a tab must hold at least one cell (got %v)", tabs)
		}
		total += panes
	}
	if total != cells {
		return fmt.Errorf("tabs %v lay out %d cells, want %d", tabs, total, cells)
	}
	return nil
}

// maxTabLabel keeps a joined label from swamping the tab bar.
const maxTabLabel = 40

// tabLabel names a tab after the cells it holds — the identity a pane cannot
// carry, since neither tmux nor herdr can name a pane at creation.
func tabLabel(cmds []Command) string {
	names := make([]string, len(cmds))
	for i, c := range cmds {
		names[i] = strings.ReplaceAll(c.Name, "\n", " ")
	}
	label := []rune(strings.Join(names, "·"))
	if len(label) > maxTabLabel {
		label = label[:maxTabLabel]
	}
	return string(label)
}

// buildScript wraps one command for its pane. The script itself is plain sh;
// the user's command runs under the profile's shell exactly like single-run
// (`bash -lc '<cmd>'`), with the command single-quote-escaped — it is never
// interpolated into a tmux argument, which is the classic batch footgun.
func buildScript(c Command) string {
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	fmt.Fprintf(&b, "# drako batch cell: %s\n", strings.ReplaceAll(c.Name, "\n", " "))

	prefix := ""
	if c.Isolate {
		prefix = strings.Join(append([]string{"env", "-i"}, envAssignments(c.Env)...), " ") + " "
	} else {
		for _, a := range envAssignments(c.Env) {
			fmt.Fprintf(&b, "export %s\n", a)
		}
	}
	fmt.Fprintf(&b, "%s%s -lc %s\n", prefix, shellBinary(c.Shell), posixSingleQuote(c.Script))
	if c.KeepOpen {
		b.WriteString("status=$?\n")
		b.WriteString(`printf '\n--- Command Finished (exit %s) ---\nPress Enter to close.' "$status"` + "\n")
		b.WriteString("read _\n")
	}
	return b.String()
}

// envAssignments renders env entries as shell-safe `K='v'` pairs, dropping
// anything without a '='.
func envAssignments(env []string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		if k, v, ok := strings.Cut(e, "="); ok {
			out = append(out, k+"="+posixSingleQuote(v))
		}
	}
	return out
}

// shellBinary maps a configured shell to the binary invoked with -lc,
// mirroring core's buildShellCmd cases (tmux implies a unix host, so the
// Windows shells are not mapped; unknown values fall back to bash).
func shellBinary(shell string) string {
	switch shell {
	case "bash", "sh", "zsh", "fish":
		return shell
	default:
		return "bash"
	}
}

// posixSingleQuote wraps s in single quotes with the standard '\'' escape, so
// the string survives the sh line verbatim regardless of its content.
func posixSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// sanitizeName reduces a cell name to a filename-safe slug.
func sanitizeName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = "cell"
	}
	if len(slug) > 40 {
		slug = slug[:40]
	}
	return slug
}
