package app

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lucky7xz/drako/internal/lair"
	"github.com/lucky7xz/drako/internal/launch"
	"github.com/lucky7xz/drako/internal/proc"
	"golang.org/x/term"
)

// pauseFn is pause, swappable so a test never waits on a keypress.
var pauseFn = pause

// runPlan carries out plan in the terminal the TUI has just left, and pauses
// for a keypress where the plan or the outcome calls for one.
//
// end is true when drako was told to end (TERM, HUP) while the command ran:
// the command has ended with it, and drako exits with code without a pause.
func runPlan(home *lair.Lair, plan launch.Plan) (code int, end bool) {
	selected := plan.Name

	if !plan.Runnable() {
		log.Printf("No command configured for: %s", selected)
		fmt.Printf("\n--- No Command Configured ---\n")
		fmt.Printf("Command: '%s'\n", selected)
		pauseFn("\nPress any key to return to the application.")
		return 0, false
	}

	// Log the execution to history.log; best-effort, never blocks the run.
	home.LogExecution(selected, strings.Join(plan.Argv, " "))

	if plan.Debug {
		// Debug: capture combined output and pause.
		output, err := proc.Capture(plan)
		if told := (*proc.ToldToEnd)(nil); errors.As(err, &told) {
			return told.Code, true
		}
		fmt.Printf("\n--- Command Output ---\n")
		fmt.Printf("Command: '%s'\n\n", selected)
		fmt.Print(string(output))
		if err != nil {
			fmt.Print(failedText(err))
		}
		pauseFn("\nPress any key to return to the application.")
		return 0, false
	}

	// Live: the command owns the terminal until it ends.
	if err := proc.Run(plan); err != nil {
		if told := (*proc.ToldToEnd)(nil); errors.As(err, &told) {
			return told.Code, true
		}
		fmt.Print(failedText(err))
		fmt.Printf("Command: '%s'\n", selected)
		pauseFn("\nPress any key to return to the application.")
		return 0, false
	}

	// If we shouldn't auto-close after success, pause so the user can read output.
	if !plan.AutoClose {
		fmt.Printf("\n--- Command Finished ---\n")
		fmt.Printf("Command: '%s'\n", selected)
		pauseFn("\nPress any key to return to the application.")
	}
	return 0, false
}

// failedText says a command failed: with its exit code when it ended with
// one, as a batch pane does, else with the reason (a signal, a missing shell).
func failedText(err error) string {
	var exited interface{ ExitCode() int }
	if errors.As(err, &exited) && exited.ExitCode() >= 0 {
		return fmt.Sprintf("\n--- Command Failed (exit %d) ---\n", exited.ExitCode())
	}
	return fmt.Sprintf("\n--- Command Failed ---\nError: %v\n", err)
}

// pause prints msg and blocks until any keypress: the "press any key" beat
// between a finished command and the next TUI session.
func pause(msg string) {
	if msg != "" {
		fmt.Print(msg)
	}
	waitForAnyKey()
}

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
