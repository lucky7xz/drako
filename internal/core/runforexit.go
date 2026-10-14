package core

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/lucky7xz/drako/internal/config"
)

// RunForExit is RunCommand for the shell: it runs commandStr through the
// configured shell and returns the child's exit code, with no TUI pauses.
// name labels the run in history.log. A command that ran and failed is
// (code, nil); an error means it never started.
func RunForExit(cfg config.Config, name, commandStr, activeProfile string) (int, error) {
	if strings.TrimSpace(commandStr) == "" {
		return 0, fmt.Errorf("%q has no command for this platform", name)
	}
	plan := execPlan{kind: planShell, shell: cfg.DefaultShell, commandStr: commandStr}
	cmd := assembleCmd(plan, CommandEnv(os.Environ(), cfg.EnvWhitelist, activeProfile))
	LogExecution(name, strings.Join(cmd.Args, " "))

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 0, fmt.Errorf("could not run %q: %w", name, err)
	}
	return 0, nil
}
