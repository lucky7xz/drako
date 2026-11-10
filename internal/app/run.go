package app

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lucky7xz/drako/internal/core"
	"github.com/lucky7xz/drako/internal/launch"
	"github.com/lucky7xz/drako/internal/proc"
	"golang.org/x/term"
)

// pauseFn is pause, swappable so a test never waits on a keypress.
var pauseFn = pause

// runPlan carries out plan in the terminal the TUI has just left, and pauses
// for a keypress where the plan or the outcome calls for one.
func runPlan(plan launch.Plan) {
	selected := plan.Name

	if !plan.Runnable() {
		log.Printf("No command configured for: %s", selected)
		fmt.Printf("\n--- No Command Configured ---\n")
		fmt.Printf("Command: '%s'\n", selected)
		pauseFn("\nPress any key to return to the application.")
		return
	}

	// Log the execution to history.log; best-effort, never blocks the run.
	core.LogExecution(selected, strings.Join(plan.Argv, " "))

	if plan.Debug {
		// Debug: capture combined output and pause.
		output, err := proc.Capture(plan)
		fmt.Printf("\n--- Command Output ---\n")
		fmt.Printf("Command: '%s'\n\n", selected)
		fmt.Print(string(output))
		if err != nil {
			fmt.Printf("\n--- Command Failed ---\n")
			fmt.Printf("Error: %v\n", err)
		}
		pauseFn("\nPress any key to return to the application.")
		return
	}

	// Live: the command owns the terminal until it ends.
	if err := proc.Run(plan); err != nil {
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
	}
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
