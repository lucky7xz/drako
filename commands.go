package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

// - Optional booleans in config are represented as *bool (pointer-to-bool) so we
//   can distinguish "unset" (nil) from "false". We then resolve them via
//   boolOrDefault.

func boolOrDefault(ptr *bool, def bool) bool {
	if ptr == nil { return def }
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

// runCommand finds the selected command from the loaded config and executes it.

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
	case "pwsh", "powershell":
		return exec.Command("pwsh", "-NoLogo", "-NoProfile", "-Command", commandStr)
	case "cmd", "cmd.exe":
		return exec.Command("cmd", "/C", commandStr)
	default:
		return exec.Command("bash", "-lc", commandStr)
	}
}

func runCommand(config Config, selected string) {
	// cmd will hold the prepared command to run. It's a pointer type; zero value is nil.
	var cmd *exec.Cmd
	// Pointers to per-command overrides; nil means "use default".
	var autoClosePtr *bool
	var debugPtr *bool

	// Default shell to use for string commands (honors config/profile).
	shell_config := config.DefaultShell

	// Search for a top-level command or a nested item matching the selected name.
	found := false
	for _, customCmd := range config.Commands {
		if customCmd.Name == selected {
			// Replace {dR4ko_path} token before building the command.
			commandStr := strings.ReplaceAll(customCmd.Command, "{dR4ko_path}", config.DR4koPath)
			if commandStr != "" {
				cmd = buildShellCmd(shell_config, commandStr)
				autoClosePtr = customCmd.AutoCloseExecution
				debugPtr = customCmd.DebugExecution
			}
			found = true
			break
		}
		// Scan nested menu items.
		for _, item := range customCmd.Items {
			if item.Name == selected {
				commandStr := strings.ReplaceAll(item.Command, "{dR4ko_path}", config.DR4koPath)
				if commandStr != "" {
					cmd = buildShellCmd(shell_config, commandStr)
					autoClosePtr = item.AutoCloseExecution
					debugPtr = item.DebugExecution
				}
				found = true
				break
			}
		}
		if found { break }
	}

	// If we didn't find a prepared command to run via a shell:
	// - If a config match existed but had no command string, don't try PATH; just inform and return.
	// - Otherwise, try to execute the "selected" token directly as a binary in PATH (no shell).
	if cmd == nil {
		if found {
			log.Printf("No command configured for: %s", selected)
			fmt.Printf("\n--- No Command Configured ---\n")
			fmt.Printf("Command: '%s'\n", selected)
			fmt.Printf("\nPress any key to return to the application.")
			waitForAnyKey()
			return
		}
		if path, err := exec.LookPath(selected); err == nil {
			// This is like subprocess.run([path]) in Python; argv is literal (no shell).
			cmd = exec.Command(path)
		} else {
			log.Printf("Executable not found in PATH: %s", selected)
			return
		}
	}

	// Resolve flags after overrides may have been set.
	autoClose := boolOrDefault(autoClosePtr, true)
	debug := boolOrDefault(debugPtr, false)

	if debug {
		// Debug: capture combined output and pause.
		output, err := cmd.CombinedOutput()
		fmt.Printf("\n--- Command Output ---\n")
		fmt.Printf("Command: '%s'\n\n", selected)
		fmt.Print(string(output))
		if err != nil {
			fmt.Printf("\n--- Command Failed ---\n")
			fmt.Printf("Error: %v\n", err)
		}
		fmt.Printf("\nPress any key to return to the application.")
		waitForAnyKey()
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
		fmt.Printf("\nPress any key to return to the application.")
		waitForAnyKey()
		return
	}

	// If we shouldn't auto-close after success, pause so the user can read output.
	if !autoClose {
		fmt.Printf("\n--- Command Finished ---\n")
		fmt.Printf("Command: '%s'\n", selected)
		fmt.Printf("\nPress any key to return to the application.")
		waitForAnyKey()
		return
	}
}
