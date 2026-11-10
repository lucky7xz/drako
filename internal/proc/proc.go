// Package proc is the only place drako starts another program. It starts
// three kinds: the command a launch.Plan describes (Run, Capture, Replace),
// the user's own editor (Editor), and the helper programs listed in tool.go.
// internal/arch checks that no other package imports os/exec or syscall.
package proc

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/lucky7xz/drako/internal/launch"
)

// errNothingToRun answers a plan with no command. Callers ask plan.Runnable
// first; this keeps a missed check from becoming a crash.
var errNothingToRun = errors.New("nothing to run")

// Run runs plan's command as a child that owns drako's terminal, and waits
// for it to end.
func Run(plan launch.Plan) error {
	if !plan.Runnable() {
		return errNothingToRun
	}
	cmd := command(plan, launch.Env(os.Environ(), plan.Profile))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return runRelayed(cmd)
}

// Capture runs plan's command with nothing on screen and returns what it
// printed, stdout and stderr together. It is how debug_execution runs.
func Capture(plan launch.Plan) ([]byte, error) {
	if !plan.Runnable() {
		return nil, errNothingToRun
	}
	cmd := command(plan, launch.Env(os.Environ(), plan.Profile))
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	err := runRelayed(cmd)
	return output.Bytes(), err
}

// command turns a runnable plan into an exec.Cmd with the prepared child
// environment attached — the same env for every run mode, debug included.
func command(plan launch.Plan, env []string) *exec.Cmd {
	cmd := exec.Command(plan.Argv[0], plan.Argv[1:]...)
	passCmdExeLine(cmd, plan.Argv)
	cmd.Env = env
	return cmd
}

// runRelayed runs cmd and keeps drako alive until it exits: Ctrl-C and Ctrl-\
// already reach it through the terminal, TERM and HUP are passed on.
func runRelayed(cmd *exec.Cmd) error {
	sigs := make(chan os.Signal, 1)
	// Registered before Start, so no signal can take drako's default exit.
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigs)
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			select {
			case sig := <-sigs:
				if sig == syscall.SIGTERM || sig == syscall.SIGHUP {
					cmd.Process.Signal(sig)
				}
			case <-done:
				return
			}
		}
	}()
	return cmd.Wait()
}
