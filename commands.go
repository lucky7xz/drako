package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/lucky7xz/drako/internal/config"
	"golang.org/x/term"
)

var (
	pauseFn       = pause
	lookPathFn    = exec.LookPath
	commandFn     = exec.Command
	setenvFn      = os.Setenv
	unsetenvFn    = os.Unsetenv
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

func pause(msg string) {
	if msg != "" {
		fmt.Print(msg)
	}
	waitForAnyKey()
}

// findCommandByName returns a pointer to the matching top-level command or a nested item.
// If an item is returned, the parent command is also returned.
func findCommandByName(cfg config.Config, name string) (parent *config.Command, item *config.CommandItem, ok bool) {
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

func runCommand(cfg config.Config, selected string) {
	// cmd will hold the prepared command to run. It's a pointer type; zero value is nil.
	var cmd *exec.Cmd
	// Pointers to per-command overrides; nil means "use default".
	var autoClosePtr *bool
	var debugPtr *bool

	// Default shell to use for string commands (honors config/profile).
	shell_config := cfg.DefaultShell

	// Resolve a top-level command or nested item by name.
	parentCmd, itemCfg, found := findCommandByName(cfg, selected)
	if found {
		if itemCfg == nil {
			// top-level command
			commandStr := config.ExpandCommandTokens(parentCmd.Command, cfg)
			if commandStr != "" {
				cmd = buildShellCmd(shell_config, commandStr)
				autoClosePtr = parentCmd.AutoCloseExecution
				debugPtr = parentCmd.DebugExecution
			}
		} else {
			// dropdown item
			commandStr := config.ExpandCommandTokens(itemCfg.Command, cfg)
			if commandStr != "" {
				cmd = buildShellCmd(shell_config, commandStr)
				autoClosePtr = itemCfg.AutoCloseExecution
				debugPtr = itemCfg.DebugExecution
			}
		}
	}

	// If we didn't find a prepared command to run via a shell:
	// - If a config match existed but had no command string, don't try PATH; just inform and return.
	// - Otherwise, try to execute the "selected" token directly as a binary in PATH (no shell).
	if cmd == nil {
		if found {
			log.Printf("No command configured for: %s", selected)
			fmt.Printf("\n--- No Command Configured ---\n")
			fmt.Printf("Command: '%s'\n", selected)
			pauseFn("\nPress any key to return to the application.")

			return
		}
		if path, err := lookPathFn(selected); err == nil {
			// This is like subprocess.run([path]) in Python; argv is literal (no shell).
			cmd = commandFn(path)
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
	if !autoClose {
		fmt.Printf("\n--- Command Finished ---\n")
		fmt.Printf("Command: '%s'\n", selected)
		pause("\nPress any key to return to the application.")

		return
	}
}
