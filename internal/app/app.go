package app

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/cli"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
	"github.com/lucky7xz/drako/internal/ui"
)

// Run wires everything together. It keeps the program running so that after a command
// finishes we jump back into the TUI without losing state or the screen layout.
func Run() {
	// Simple manual flag check for --cwd-file since standard flag package
	// might interfere with other CLI logic if not careful.
	// We want to extract it before HandleCLI or anything else.
	var cwdFile string
	cleanArgs := []string{}
	skipNext := false

	for i, arg := range os.Args {
		if skipNext {
			skipNext = false
			continue
		}
		if arg == "--cwd-file" {
			if i+1 < len(os.Args) {
				cwdFile = os.Args[i+1]
				skipNext = true
			}
			continue
		}
		if strings.HasPrefix(arg, "--cwd-file=") {
			cwdFile = strings.TrimPrefix(arg, "--cwd-file=")
			continue
		}
		cleanArgs = append(cleanArgs, arg)
	}
	// Temporarily override args for downstream parsers if needed,
	// but standard cli.HandleCLI uses flag package?
	// It parses its own flags.
	// For now, we just intercept our special integration flag.

	// Check if CLI command was invoked (e.g., drako sync <url>)
	// Note: HandleCLI parses os.Args directly.
	if cli.HandleCLI(cleanArgs) {
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
	// Rotate if > 1MB
	core.RotateLogIfNeeded(logPath, 1024*1024)

	f, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("could not open log file: %v", err)
		os.Exit(1)
	}
	defer f.Close()
	log.SetOutput(f)

	// Keep track of the last known directory
	lastCwd, _ := os.Getwd()

	for {
		// Start the TUI program (Model/View/Update is now in internal/ui)
		// We initialize with the *current* directory which might have changed
		// from manual Chdir or from internal logic
		program := tea.NewProgram(ui.InitialModel())

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

		// Update our last known CWD from the model state
		// We access via state.Path.CurrentPath (exposed via PathModel)
		// Wait, state is ui.Model, field is 'path', which is not exported?
		// We need to export 'Path' in Model, or access CurrentPath logic.
		// Ah, in previous step I named field 'path' (lowercase).
		// I must update model.go to export 'Path' or access method.
		// Actually, I can rely on os.Getwd() since Model does chdir!
		// Let's verify: UI calls os.Chdir() on confirm.
		// So os.Getwd() here is sufficient.
		if wd, err := os.Getwd(); err == nil {
			lastCwd = wd
		}

		if state.Quitting {
			// Write the final CWD if requested
			if cwdFile != "" {
				if err := os.WriteFile(cwdFile, []byte(lastCwd), 0644); err != nil {
					log.Printf("Failed to write cwd file: %v", err)
				}
			}
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
