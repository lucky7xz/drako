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

// FindCommandByName returns a pointer to the matching top-level command or a nested item.
// If an item is returned, the parent command is also returned.
func FindCommandByName(cfg config.Config, name string) (parent *config.Command, item *config.CommandItem, ok bool) {
	for i := range cfg.Commands {
		c := &cfg.Commands[i]
		if c.Name == name {
			return c, nil, true
		}
		for j := range c.Items {
			if c.Items[j].Name == name {
				return c, &c.Items[j], true
			}
		}
	}
	return nil, nil, false
}

// - exec.Cmd is like subprocess, constructed with argv (no implicit shell).
// - We explicitly ask for a shell with buildShellCmd.

// Default shell for inline strings; can be wired from config later.

// buildShellCmd constructs an *exec.Cmd for the given shell. Pure function: no execution.
func buildShellCmd(shell_config, commandStr string) *exec.Cmd {
	switch shell_config {
	case "bash":
		return exec.Command("bash", "-lc", commandStr)
	case "sh":
		return exec.Command("sh", "-c", commandStr)
	case "zsh":
		return exec.Command("zsh", "-lc", commandStr)
	case "fish":
		return exec.Command("fish", "-c", commandStr)
	case "pwsh":
		return exec.Command("pwsh", "-NoLogo", "-NoProfile", "-Command", commandStr)
	// powershell.exe 5.1 ships with Windows; pwsh (7+) must be installed.
	case "powershell", "powershell.exe":
		return exec.Command("powershell", "-NoLogo", "-NoProfile", "-Command", commandStr)
	case "cmd", "cmd.exe":
		return exec.Command("cmd", "/C", commandStr)
	default:
		return buildShellCmd(fallbackShell(runtime.GOOS), commandStr)
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
// it. activeProfile is the name of the profile the selection came from; it is
// exported to the child process as DRAKO_PROFILE.
func RunCommand(cfg config.Config, selected, activeProfile string) {
	plan := buildExecPlan(cfg, selected, lookPathFn)

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

	// Sanitize the child environment: the whitelist (if configured) restricts
	// what commands inherit; drako's own DRAKO_PROFILE is always present.
	cmd := assembleCmd(plan, CommandEnv(os.Environ(), cfg.EnvWhitelist, activeProfile))

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
