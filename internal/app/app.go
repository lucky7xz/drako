package app

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/cli"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
	"github.com/lucky7xz/drako/internal/ui"
)

// Run wires everything together. It keeps the program running so that after a command
// finishes we jump back into the TUI without losing state or the screen layout.

func Run() {
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
		if cli.HandleCLI(os.Args) {
			// HandleCLI returns true/false to indicate success.
			// Either way, we exit here. No TUI.
			return
		}
	}

	// 2. If we are here, we are launching the TUI.
	// =======================================

	// Proceed with TUI mode
	configDir, err := config.GetConfigDir()
	if err != nil {
		fmt.Printf("could not get config dir: %v", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		fmt.Printf("could not create config dir: %v", err)
		os.Exit(1)
	}
	// =======================================
	// Logging setup

	// Rotate if > 1MB
	logPath := filepath.Join(configDir, "drako.log")
	core.RotateLogIfNeeded(logPath, 1024*1024)

	f, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("could not open log file: %v", err)
		os.Exit(1)
	}
	defer f.Close()
	log.SetOutput(f)

	// Start of TUI Loop
	for {

		// Start the TUI program (Model/View/Update is now in internal/ui)
		// We initialize with the *current* directory which might have changed
		// from manual Chdir or from internal logic

		program := tea.NewProgram(ui.InitialModel(glassrootMode))

		result, err := program.Run()
		if err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}

		// Cast result to ui.Model
		state, ok := result.(ui.Model)
		if !ok {
			// Should not happen, but safe exit
			return
		}

		if state.Quitting {
			return
		}

		if state.Selected != "" {
			core.RunCommand(state.Config, state.Selected)

			cmd := exec.Command("clear")
			cmd.Stdout = os.Stdout
			_ = cmd.Run()
		}
	}
}
