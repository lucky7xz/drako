package core

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/lucky7xz/drako/internal/config"
	"golang.org/x/term"
)

var (
	pauseFn    = pause
	lookPathFn = exec.LookPath
	commandFn  = exec.Command
)

// - Optional booleans in config are represented as *bool (pointer-to-bool) so we
//   can distinguish "unset" (nil) from "false". We then resolve them via
//   boolOrDefault.

func boolOrDefault(ptr *bool, def bool) bool {
	if ptr == nil {
		return def
	}
	return *ptr
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

	// Read one byte (ignore errors since we're just pausing)
	buf := make([]byte, 1)
	_, _ = os.Stdin.Read(buf)
}

func pause(msg string) {
	if msg != "" {
		fmt.Print(msg)
	}
	waitForAnyKey()
}

// Pause prints msg and blocks until any keypress. Exported for the app layer,
// which needs the same "press any key" beat between TUI sessions.
func Pause(msg string) {
	pause(msg)
}

// Selection is what the user picked: the cell named Cell, or item Item (1-based)
// of its dropdown. Item names alone can repeat across folders.
type Selection struct {
	Cell string
	Item int // 0 selects the cell itself
}

// Resolve finds the cell among top-level commands (drako check keeps their
// names unique) and, for Item > 0, the item inside it.
func (s Selection) Resolve(cfg config.Config) (cell *config.Command, item *config.CommandItem, ok bool) {
	for i := range cfg.Commands {
		c := &cfg.Commands[i]
		if c.Name != s.Cell {
			continue
		}
		switch {
		case s.Item == 0:
			return c, nil, true
		case s.Item > 0 && s.Item <= len(c.Items):
			return c, &c.Items[s.Item-1], true
		}
		return nil, nil, false
	}
	return nil, nil, false
}

// Name is the picked entry's name as shown on screen: the item's, else the cell's.
func (s Selection) Name(cfg config.Config) string {
	if _, item, ok := s.Resolve(cfg); ok && item != nil {
		return item.Name
	}
	return s.Cell
}

// - exec.Cmd is like subprocess, constructed with argv (no implicit shell).
// - We explicitly ask for a shell with buildShellCmd.

// Default shell for inline strings; can be wired from config later.

// buildShellCmd constructs an *exec.Cmd for the given shell. Pure function: no execution.
func buildShellCmd(shell_config, commandStr string) *exec.Cmd {
	argv := ShellArgv(shell_config, commandStr)
	return exec.Command(argv[0], argv[1:]...)
}

// ShellArgv is the command line that runs commandStr under the configured shell,
// for single runs and batch cells alike; unknown names get the platform default.
func ShellArgv(shell, commandStr string) []string {
	switch shell {
	case "bash":
		return []string{"bash", "-lc", commandStr}
	case "sh":
		return []string{"sh", "-c", commandStr}
	case "zsh":
		return []string{"zsh", "-lc", commandStr}
	case "fish":
		return []string{"fish", "-c", commandStr}
	case "pwsh":
		return []string{"pwsh", "-NoLogo", "-NoProfile", "-Command", commandStr}
	// powershell.exe 5.1 ships with Windows; pwsh (7+) must be installed.
	case "powershell", "powershell.exe":
		return []string{"powershell", "-NoLogo", "-NoProfile", "-Command", commandStr}
	case "cmd", "cmd.exe":
		return []string{"cmd", "/C", commandStr}
	default:
		return ShellArgv(fallbackShell(runtime.GOOS), commandStr)
	}
}

// fallbackShell picks the shell for unknown/empty shell settings.
func fallbackShell(goos string) string {
	if goos == "windows" {
		return "powershell"
	}
	return "bash"
}

// RunCommand finds the selected command from the loaded config and executes
// it. activeProfile is exported to the child process as DRAKO_PROFILE.
func RunCommand(cfg config.Config, sel Selection, activeProfile string) {
	plan := buildExecPlan(cfg, sel, lookPathFn)
	selected := sel.Name(cfg)

	switch plan.kind {
	case planNoCommand:
		log.Printf("No command configured for: %s", selected)
		fmt.Printf("\n--- No Command Configured ---\n")
		fmt.Printf("Command: '%s'\n", selected)
		pauseFn("\nPress any key to return to the application.")
		return
	case planNotFound:
		log.Printf("Executable not found in PATH: %s", selected)
		return
	}

	cmd := assembleCmd(plan, CommandEnv(os.Environ(), activeProfile))

	// Log the execution to history.log; best-effort, never blocks the run.
	LogExecution(selected, strings.Join(cmd.Args, " "))

	if plan.debug {
		// Debug: capture combined output and pause.
		output, err := cmd.CombinedOutput()
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

	// Live: stream I/O directly to the terminal.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("\n--- Command Failed ---\n")
		fmt.Printf("Command: '%s'\n", selected)
		fmt.Printf("Error: %v\n", err)
		pauseFn("\nPress any key to return to the application.")

		return
	}

	// If we shouldn't auto-close after success, pause so the user can read output.
	if !plan.autoClose {
		fmt.Printf("\n--- Command Finished ---\n")
		fmt.Printf("Command: '%s'\n", selected)
		pause("\nPress any key to return to the application.")

		return
	}
}
