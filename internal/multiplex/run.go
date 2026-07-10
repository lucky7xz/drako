package multiplex

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// execStep runs one tmux invocation. Test seam, like core's commandFn.
// The attach step is the terminal handoff: it gets raw stdio and the
// sanitized child environment; setup steps run silently.
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

// Run executes a planned session: write the scripts (0700 — they contain the
// user's command text, same sensitivity as history.log), then run the steps
// in order. For an attached session the script dir is removed once control
// returns (panes hold their script fds, so unlinking is safe mid-run). A
// nested launch returns immediately while tmux is still spawning panes, so
// its scripts are left for the OS tmp reaper rather than racing the spawn.
func Run(s Session, scriptDir string, env []string) error {
	if err := os.MkdirAll(scriptDir, 0o700); err != nil {
		return fmt.Errorf("could not create script dir: %w", err)
	}
	for name, content := range s.Scripts {
		if err := os.WriteFile(filepath.Join(scriptDir, name), []byte(content), 0o700); err != nil {
			return fmt.Errorf("could not write %s: %w", name, err)
		}
	}

	for i, step := range s.Steps {
		attach := s.Attach && i == len(s.Steps)-1
		if err := execStep(step, attach, env); err != nil {
			return fmt.Errorf("tmux %s failed: %w", step[1], err)
		}
	}

	if s.Attach {
		if err := os.RemoveAll(scriptDir); err != nil {
			return fmt.Errorf("could not clean up scripts: %w", err)
		}
	}
	return nil
}
