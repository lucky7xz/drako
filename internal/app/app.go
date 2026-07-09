package app

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/cli"
	"github.com/lucky7xz/drako/internal/core"
	"github.com/lucky7xz/drako/internal/paths"
	"github.com/lucky7xz/drako/internal/ui"
)

// Run wires everything together. It keeps the program running so that after a command
// finishes we jump back into the TUI without losing state or the screen layout.
// It returns the process exit code; main is the only caller of os.Exit, so
// deferred cleanups here always run before the process ends.

func Run() int {
	glassrootMode := false
	isTuiMode := false

	// CLI handling
	// =======================================
	// Check for TUI-specific flags (Glassroot)
	// If present, we short-circuit the CLI handler entirely.
	for _, arg := range os.Args {
		if arg == "--glassroot" {
			isTuiMode = true
			glassrootMode = true
			break
		}
	}

	// 1. If NOT in TUI mode, try to handle as a CLI command (e.g. "drako summon", "drako purge")
	if !isTuiMode {
		if handled, code := cli.HandleCLI(os.Args); handled {
			return code
		}
	}

	// 2. If we are here, we are launching the TUI.
	// =======================================

	// Proceed with TUI mode
	configDir, err := paths.ConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get config dir: %v\n", err)
		return 1
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "could not create config dir: %v\n", err)
		return 1
	}
	// =======================================
	// Logging setup

	// Rotate if > 1MB
	logPath := paths.LogFile(configDir)
	core.RotateLogIfNeeded(logPath, paths.LogArchive(configDir), 1024*1024)

	f, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		// NOTE: cli.go treats this as a non-fatal warning and continues;
		// reconcile (warn vs exit) in a follow-up.
		fmt.Fprintf(os.Stderr, "could not open log file: %v\n", err)
		return 1
	}
	defer f.Close()
	log.SetOutput(f)

	// Start of TUI Loop
	for {

		// Start the TUI program (Model/View/Update is now in internal/ui)
		// We initialize with the *current* directory which might have changed
		// from manual Chdir or from internal logic

		initial := ui.InitialModel(glassrootMode)
		if initial.Quitting {
			// e.g. glassroot with a broken profile: fail before the TUI starts
			return initial.ExitCode
		}
		program := tea.NewProgram(initial)

		result, err := program.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Alas, there's been an error: %v\n", err)
			return 1
		}

		// Cast result to ui.Model
		state, ok := result.(ui.Model)
		if !ok {
			// Should not happen, but safe exit
			return 0
		}

		if state.Quitting {
			return state.ExitCode
		}

		if state.Selected != "" {
			// Internal "drako …" cells are dispatched here, at the layer that
			// may wire cli and core together; everything else runs via core.
			switch {
			case strings.HasPrefix(state.Selected, "drako purge"):
				if runInternalPurge(state.Selected) {
					return 0 // successful purge ends the session
				}
			case strings.HasPrefix(state.Selected, "drako open"):
				cli.HandleOpenCommand(state.Selected)
			default:
				core.RunCommand(state.Config, state.Selected)
			}

			cmd := exec.Command("clear")
			cmd.Stdout = os.Stdout
			_ = cmd.Run()
		}
	}
}

// runInternalPurge runs a "drako purge …" cell in-process and reports whether
// the purge succeeded (after which the app should exit rather than relaunch
// the TUI on a config that may no longer exist).
func runInternalPurge(command string) bool {
	// Strip "drako purge" to match what os.Args[2:] would provide.
	parts := strings.Fields(command)
	if len(parts) < 2 {
		log.Printf("Invalid purge command: %s", command)
		return false
	}

	if err := cli.ExecutePurge(parts[2:]); err != nil {
		fmt.Printf("\nInternal Purge Error: %v\n", err)
		core.Pause("\nPress any key...")
		return false
	}
	fmt.Printf("\npress any key to exit...")
	core.Pause("")
	return true
}
