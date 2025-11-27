package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
)

// main wires everything together. It keeps the program running so that after a command
// finishes we jump back into the TUI without losing state or the screen layout.
func main() {
	// Check if CLI command was invoked (e.g., drako sync <url>)
	if handleCLI() {
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
		program := tea.NewProgram(initialModel())

		result, err := program.Run()
		if err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}

		state, ok := result.(model)
		if !ok || state.quitting {
			return
		}

		if state.selected != "" {
			runCommand(state.config, state.selected)

			cmd := exec.Command("clear")
			cmd.Stdout = os.Stdout
			_ = cmd.Run()
		}
	}
}
