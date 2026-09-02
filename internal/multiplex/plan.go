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

// Plan lays out the tmux session for cmds. Up to four commands share one
// window as tiled panes; more get one named window each so the tab bar stays
// readable. insideTmux means drako already runs inside a session ($TMUX set):
// then the commands join the current session and there is no attach step.
// scriptDir is where the caller will write the scripts; only paths are
// computed here.
func Plan(session string, cmds []Command, insideTmux bool, scriptDir string) (Session, error) {
	if len(cmds) == 0 {
		return Session{}, fmt.Errorf("nothing to launch")
	}
	if len(cmds) > MaxCommands {
		return Session{}, fmt.Errorf("at most %d commands per batch (got %d)", MaxCommands, len(cmds))
	}

	s := Session{Name: session, Scripts: map[string]string{}}

	paths := make([]string, len(cmds))
	for i, c := range cmds {
		filename := fmt.Sprintf("%02d-%s.sh", i+1, sanitizeName(c.Name))
		s.Scripts[filename] = buildScript(c)
		paths[i] = filepath.Join(scriptDir, filename)
	}

	panes := len(cmds) <= 4

	for i, c := range cmds {
		switch {
		case insideTmux && panes:
			s.Steps = append(s.Steps, []string{"tmux", "split-window", paths[i]})
		case insideTmux:
			s.Steps = append(s.Steps, []string{"tmux", "new-window", "-n", c.Name, paths[i]})
		case i == 0 && panes:
			s.Steps = append(s.Steps, []string{"tmux", "new-session", "-d", "-s", session, paths[i]})
		case i == 0:
			s.Steps = append(s.Steps, []string{"tmux", "new-session", "-d", "-s", session, "-n", c.Name, paths[i]})
		case panes:
			s.Steps = append(s.Steps, []string{"tmux", "split-window", "-t", session, paths[i]})
		default:
			s.Steps = append(s.Steps, []string{"tmux", "new-window", "-t", session, "-n", c.Name, paths[i]})
		}
	}

	if panes && len(cmds) > 1 {
		layout := []string{"tmux", "select-layout"}
		if !insideTmux {
			layout = append(layout, "-t", session)
		}
		s.Steps = append(s.Steps, append(layout, "tiled"))
	}

	if !insideTmux {
		s.Steps = append(s.Steps, []string{"tmux", "attach-session", "-t", session})
		s.Attach = true
	}
	return s, nil
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
