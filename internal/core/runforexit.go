package core

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/lucky7xz/drako/internal/config"
)

// RunForExit is RunCommand without the TUI. Outside Windows the command
// replaces drako, so every signal reaches it and its exit status is drako
// run's; it returns only to say why the command never started.
func RunForExit(cfg config.Config, name, commandStr, activeProfile string) (int, error) {
	if strings.TrimSpace(commandStr) == "" {
		return 0, fmt.Errorf("%q has no command for this platform", name)
	}
	plan := execPlan{kind: planShell, shell: cfg.DefaultShell, commandStr: commandStr}
	cmd := assembleCmd(plan, CommandEnv(os.Environ(), activeProfile))
	LogExecution(name, strings.Join(cmd.Args, " "))

	if runtime.GOOS == "windows" {
		return waitForExit(cmd, name)
	}
	err := cmd.Err
	if err == nil {
		err = syscall.Exec(cmd.Path, cmd.Args, cmd.Env)
	}
	return 0, fmt.Errorf("could not run %q: %w", name, err)
}

// waitForExit runs the command as a child, since Windows has no exec, and
// outlives the Ctrl-C the console sends to both.
func waitForExit(cmd *exec.Cmd, name string) (int, error) {
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	defer signal.Stop(interrupts)

	err := cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	if err != nil {
		return 0, fmt.Errorf("could not run %q: %w", name, err)
	}
	return 0, nil
}
