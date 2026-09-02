package multiplex

import (
	"fmt"
	"os"
	"os/exec"
)

// Tmux launches a batch through tmux. Its plan is built whole before anything
// runs (see Plan) — tmux takes the command as an argument to the call that
// creates the pane, and every target is a name drako chose, so nothing has to
// be read back.
type Tmux struct {
	// Inside means drako already runs in a tmux session ($TMUX set): the tabs
	// join it and there is no attach step.
	Inside bool
}

func NewTmux(inside bool) *Tmux { return &Tmux{Inside: inside} }

func (t *Tmux) Name() string { return tmuxMux }

func (t *Tmux) Launch(session string, cmds []Command, tabs []int, paths, env []string) (bool, error) {
	plan, err := Plan(session, cmds, tabs, paths, t.Inside)
	if err != nil {
		return false, err
	}
	for i, step := range plan.Steps {
		attach := plan.Attach && i == len(plan.Steps)-1
		if err := execStep(step, attach, env); err != nil {
			return false, fmt.Errorf("%s failed: %w", step[1], err)
		}
	}
	return plan.Attach, nil
}

// execStep runs one tmux invocation. Test seam, like core's commandFn. The
// attach step is the terminal handoff: it gets raw stdio and the sanitized
// child environment; setup steps run silently.
var execStep = func(argv []string, attach bool, env []string) error {
	cmd := exec.Command(argv[0], argv[1:]...)
	if attach {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = env
		return cmd.Run()
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	return nil
}
