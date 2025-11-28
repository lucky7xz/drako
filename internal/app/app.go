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
	// Check if CLI command was invoked (e.g., drako sync <url>)
	if cli.HandleCLI() {
		return
	}

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
	logPath := filepath.Join(configDir, "drako.log")
	f, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("could not open log file: %v", err)
		os.Exit(1)
	}
	defer f.Close()
	log.SetOutput(f)

	for {
		// Start the TUI program (Model/View/Update is now in internal/ui)
		program := tea.NewProgram(ui.InitialModel())

		result, err := program.Run()
		if err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}

		// Cast result to ui.Model
		state, ok := result.(ui.Model)
		if !ok || state.Quitting {
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

