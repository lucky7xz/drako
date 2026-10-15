package multiplex

import (
	"fmt"
	"strings"
)

// MaxCommands mirrors drako's 1-9 idiom (quick-nav, Alt+1-9).
const MaxCommands = 9

// Command is one grid cell to launch.
type Command struct {
	Name     string   // cell name — becomes the window name and script name
	Argv     []string // the command line, as a single run would use it (core.ShellArgv)
	KeepOpen bool     // pause after success too (auto_close_execution = false, debug_execution)
	Profile  string   // the active deck, exported as DRAKO_PROFILE like a single run
}

// Session is an inert tmux launch plan: the invocations to run, in order.
// Nothing has happened yet when Plan returns.
type Session struct {
	Name   string
	Steps  [][]string // argv slices, executed in order
	Attach bool       // the last step attaches: hand it the terminal
}

// Plan lays out the tmux session for cmds. tabs says how many cells each tab
// holds, in order; paths[i] is the already-written script for cmds[i]. The
// layout is decided before Plan is called, not derived from the cell count
// here. insideTmux means drako already runs inside a session ($TMUX set): then
// the tabs join the current session and there is no attach step.
func Plan(session string, cmds []Command, tabs []int, paths []string, insideTmux bool) (Session, error) {
	if err := checkTabs(tabs, len(cmds)); err != nil {
		return Session{}, err
	}
	if len(paths) != len(cmds) {
		return Session{}, fmt.Errorf("got %d scripts for %d cells", len(paths), len(cmds))
	}

	s := Session{Name: session}

	// Outside tmux every step names the session we are building; inside it,
	// nothing is targeted and the steps land in the session drako is in.
	var target []string
	if !insideTmux {
		target = []string{"-t", session}
	}

	// A tab is one tmux call, its commands joined by ";": tmux runs them without
	// interleaving input, so a window switch mid-setup cannot redirect the splits.
	cell := 0
	for t, panes := range tabs {
		group := cmds[cell : cell+panes]
		var step []string
		if !insideTmux && t == 0 {
			step = []string{"tmux", "new-session", "-d", "-s", session}
		} else {
			step = append([]string{"tmux", "new-window"}, target...)
		}
		step = append(append(step, "-n", tmuxArg(tabLabel(group))), runScript(paths[cell])...)
		for i := 1; i < panes; i++ {
			step = append(append(append(step, ";", "split-window"), target...), runScript(paths[cell+i])...)
		}
		if panes > 1 {
			step = append(append(append(step, ";", "select-layout"), target...), "tiled")
		}
		s.Steps = append(s.Steps, step)
		cell += panes
	}

	if !insideTmux {
		s.Steps = append(s.Steps, []string{"tmux", "attach-session", "-t", session})
		s.Attach = true
	}
	return s, nil
}

// checkTabs rejects a vector that would drop or invent panes, or overfill a
// tab. The ceiling is what lets splitTree assume a layout exists for every tab
// it is handed.
func checkTabs(tabs []int, cells int) error {
	total := 0
	for _, panes := range tabs {
		if panes < 1 {
			return fmt.Errorf("a tab must hold at least one cell (got %v)", tabs)
		}
		if panes > PanesPerTab {
			return fmt.Errorf("a tab holds at most %d cells (got %v)", PanesPerTab, tabs)
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

// tmuxArg escapes a trailing ";", which tmux reads as the end of a command
// wherever it appears; tmux turns "\;" back into ";".
func tmuxArg(s string) string {
	if strings.HasSuffix(s, ";") {
		return s[:len(s)-1] + `\;`
	}
	return s
}

// buildScript wraps one command for its pane. The script itself is plain sh;
// the user's command runs under the profile's shell exactly like single-run
// (`bash -lc '<cmd>'`), with the command single-quote-escaped — it is never
// interpolated into a tmux argument, which is the classic batch footgun.
func buildScript(c Command) string {
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	fmt.Fprintf(&b, "# drako batch cell: %s\n", strings.ReplaceAll(c.Name, "\n", " "))

	if c.Profile != "" { // the pane keeps its own environment; only the deck is added
		fmt.Fprintf(&b, "export DRAKO_PROFILE=%s\n", posixSingleQuote(c.Profile))
	}
	if n := len(c.Argv); n > 0 { // the shell and its flags are fixed words; the command is quoted
		fmt.Fprintf(&b, "%s %s\n", strings.Join(c.Argv[:n-1], " "), posixSingleQuote(c.Argv[n-1]))
	}
	// Like a single run, a failed cell waits so its error stays readable.
	b.WriteString("status=$?\n")
	b.WriteString(`if [ "$status" -ne 0 ]; then printf '\n--- Command Failed (exit %s) ---\nPress Enter to close.' "$status"; read _` + "\n")
	if c.KeepOpen {
		b.WriteString(`else printf '\n--- Command Finished ---\nPress Enter to close.'; read _` + "\n")
	}
	b.WriteString("fi\n")
	return b.String()
}

// runScript is a pane's command as separate arguments, which tmux runs without a
// shell: the path is never parsed, and sh reads a script on a noexec mount.
func runScript(path string) []string {
	return []string{"sh", path}
}

// posixSingleQuote wraps s in single quotes, escaping any it already contains
// the standard POSIX way (see below), so the string survives the sh line
// verbatim regardless of its content.
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
