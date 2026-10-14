package core

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/lucky7xz/drako/internal/config"
)

// RunForExit is RunCommand without the TUI: it returns the command's exit
// code, and an error only when the command never started.
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
			// Killed by a signal: report 128+N, as a shell would.
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
				return 128 + int(ws.Signal()), nil
			}
			return exitErr.ExitCode(), nil
		}
		return 0, fmt.Errorf("could not run %q: %w", name, err)
	}
	return 0, nil
}
