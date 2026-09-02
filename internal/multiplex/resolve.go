package multiplex

import (
	"fmt"
	"os"
	"os/exec"
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
	Tmux     string // $TMUX — set inside a tmux session
	Herdr    string // $HERDR_ENV — "1" inside a herdr pane, per herdr's own check
	LookPath func(string) (string, error)
}

// OSEnv reads what Resolve needs from the process environment. The variable
// names live here and nowhere else.
func OSEnv() Env {
	return Env{
		Tmux:     os.Getenv("TMUX"),
		Herdr:    os.Getenv("HERDR_ENV"),
		LookPath: exec.LookPath,
	}
}

func (e Env) insideTmux() bool  { return e.Tmux != "" }
func (e Env) insideHerdr() bool { return e.Herdr == "1" }

func (e Env) installed(name string) bool {
	_, err := e.LookPath(name)
	return err == nil
}

// Resolve picks the backend for a batch. By default that is whichever
// multiplexer drako already runs inside, so a batch never nests one in the
// other — and outside both only tmux is reachable, because tmux can build a
// session from nothing and herdr cannot.
//
// forceTmux is the one override worth having: from inside herdr, run batches
// through tmux anyway. Every other preference either matches the default or
// cannot work at all, which is why this is a bool and not a multiplexer name.
//
// The error is the message the user sees, so it says what to do about it.
func Resolve(forceTmux bool, env Env) (Backend, error) {
	if forceTmux {
		if !env.installed(tmuxMux) {
			return nil, fmt.Errorf("Batch needs tmux installed")
		}
		return NewTmux(env.insideTmux()), nil
	}

	switch {
	case env.insideHerdr() && env.installed(herdrMux):
		return NewHerdr(), nil
	case env.installed(tmuxMux):
		return NewTmux(env.insideTmux()), nil
	case env.installed(herdrMux):
		// The binary is there; it is the position that is wrong.
		return nil, fmt.Errorf("Batch needs tmux, or drako running inside herdr")
	}
	return nil, fmt.Errorf("Batch needs tmux or herdr installed")
}
