package core

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/lucky7xz/drako/internal/launch"
	"golang.org/x/term"
)

var (
	pauseFn   = Pause
	commandFn = exec.Command
)

// waitForAnyKey waits for any single keypress in raw mode.
func waitForAnyKey() {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Fallback: just wait for Enter if raw mode fails
		fmt.Scanln()
		return
	}
	defer term.Restore(fd, oldState)

	// One read takes a whole key, so an escape sequence (arrow, Alt+key) can't
	// leave bytes behind for the next TUI to read as keys.
	buf := make([]byte, 64)
	_, _ = os.Stdin.Read(buf)
}

// Pause prints msg and blocks until any keypress. Exported for the app layer,
// which needs the same "press any key" beat between TUI sessions.
func Pause(msg string) {
	if msg != "" {
		fmt.Print(msg)
	}
	waitForAnyKey()
}

// command turns a runnable plan into an exec.Cmd with the prepared child
// environment attached — the same env for every run mode, debug included.
func command(plan launch.Plan, env []string) *exec.Cmd {
	cmd := commandFn(plan.Argv[0], plan.Argv[1:]...)
	passCmdExeLine(cmd, plan.Argv)
	cmd.Env = env
	return cmd
}

// RunCommand carries out plan in the terminal the TUI has just left, and
// pauses for a keypress where the plan or the outcome calls for one.
func RunCommand(plan launch.Plan) {
	selected := plan.Name

	if !plan.Runnable() {
		log.Printf("No command configured for: %s", selected)
		fmt.Printf("\n--- No Command Configured ---\n")
		fmt.Printf("Command: '%s'\n", selected)
		pauseFn("\nPress any key to return to the application.")
		return
	}

	cmd := command(plan, launch.Env(os.Environ(), plan.Profile))

	// Log the execution to history.log; best-effort, never blocks the run.
	LogExecution(selected, strings.Join(cmd.Args, " "))

	if plan.Debug {
		// Debug: capture combined output and pause.
		var output bytes.Buffer
		cmd.Stdout, cmd.Stderr = &output, &output
		err := runRelayed(cmd)
		fmt.Printf("\n--- Command Output ---\n")
		fmt.Printf("Command: '%s'\n\n", selected)
		fmt.Print(output.String())
		if err != nil {
			fmt.Printf("\n--- Command Failed ---\n")
			fmt.Printf("Error: %v\n", err)
		}
		pauseFn("\nPress any key to return to the application.")
		return
	}

	// Live: stream I/O directly to the terminal.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := runRelayed(cmd); err != nil {
		fmt.Printf("\n--- Command Failed ---\n")
		fmt.Printf("Command: '%s'\n", selected)
		fmt.Printf("Error: %v\n", err)
		pauseFn("\nPress any key to return to the application.")

		return
	}

	// If we shouldn't auto-close after success, pause so the user can read output.
	if !plan.AutoClose {
		fmt.Printf("\n--- Command Finished ---\n")
		fmt.Printf("Command: '%s'\n", selected)
		pauseFn("\nPress any key to return to the application.")

		return
	}
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
