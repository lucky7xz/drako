package multiplex

import (
	"fmt"
	"os"
	"strings"

	"github.com/lucky7xz/drako/internal/proc"
)

// The names the backends answer to, and the binaries they need on PATH.
const (
	tmuxMux  = "tmux"
	herdrMux = "herdr"
)

// Env is what drako can see about the terminal it was started in. It is a
// struct rather than direct os.Getenv calls so the whole decision can be
// exercised without setting environment variables.
type Env struct {
	Tmux  string // $TMUX — set inside a tmux session
	Herdr string // $HERDR_ENV — "1" inside a herdr pane, per herdr's own check
	// InTmuxPane: drako's terminal is a tmux pane, so inside both multiplexers
	// tmux is the nearer one. OSEnv only asks tmux when both are set.
	InTmuxPane bool
	// InContainer: drako runs in a container (see containerMarkers).
	InContainer bool
	LookPath    func(string) (string, error)
}

// containerMarkers are the files whose presence means drako runs in a
// container. Capability probes rather than a list of tools: podman writes the
// first in every container it runs (distrobox and toolbx boxes included),
// docker the second.
var containerMarkers = []string{
	"/run/.containerenv", // podman
	"/.dockerenv",        // docker
}

// containerMarked reports whether any container marker exists.
func containerMarked() bool {
	for _, marker := range containerMarkers {
		if _, err := os.Stat(marker); err == nil {
			return true
		}
	}
	return false
}

// OSEnv reads what Resolve needs from the process environment. The variable
// names live here and nowhere else.
func OSEnv() Env {
	e := Env{
		Tmux:        os.Getenv("TMUX"),
		Herdr:       os.Getenv("HERDR_ENV"),
		InContainer: containerMarked(),
		LookPath:    proc.LookPath,
	}
	if e.insideTmux() && e.insideHerdr() {
		e.InTmuxPane = ttyIsTmuxPane()
	}
	return e
}

// ttyIsTmuxPane reports whether drako's terminal is one of tmux's panes.
func ttyIsTmuxPane() bool {
	self, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	out, err := proc.Tmux.Command("list-panes", "-a", "-F", "#{pane_tty}").Output()
	if err != nil {
		return false
	}
	for _, tty := range strings.Fields(string(out)) {
		if pane, err := os.Stat(tty); err == nil && os.SameFile(self, pane) {
			return true
		}
	}
	return false
}

func (e Env) insideTmux() bool  { return e.Tmux != "" }
func (e Env) insideHerdr() bool { return e.Herdr == "1" }

func (e Env) installed(name string) bool {
	_, err := e.LookPath(name)
	return err == nil
}

// Resolve picks the backend for a batch. By default that is whichever
// multiplexer drako already runs inside (inside both, the nearer one), so a
// batch never nests one in the other — and outside both only tmux is
// reachable, because tmux can build a session from nothing and herdr cannot.
//
// forceTmux is the one override worth having: from inside herdr, run batches
// through tmux anyway. Every other preference either matches the default or
// cannot work at all, which is why this is a bool and not a multiplexer name.
//
// The error is the message the user sees, so it says what to do about it.
func Resolve(forceTmux bool, env Env) (Backend, error) {
	// A container may share the host's /tmp, and with it the multiplexer's
	// socket: the tabs would then run on the host, not where drako runs.
	// drako cannot tell which, so batch is off; a batch started on the host
	// can still enter a container from its command.
	if env.InContainer {
		return nil, fmt.Errorf("Batch is off inside a container: start it from the host")
	}
	if forceTmux {
		if !env.installed(tmuxMux) {
			return nil, fmt.Errorf("Batch needs tmux installed")
		}
		return NewTmux(env.insideTmux()), nil
	}

	switch {
	case env.insideHerdr() && !env.InTmuxPane && env.installed(herdrMux):
		return NewHerdr(), nil
	case env.installed(tmuxMux):
		return NewTmux(env.insideTmux()), nil
	case env.installed(herdrMux):
		// The binary is there; it is the position that is wrong.
		return nil, fmt.Errorf("Batch needs tmux, or drako running inside herdr")
	}
	return nil, fmt.Errorf("Batch needs tmux or herdr installed")
}
