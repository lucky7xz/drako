package core

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/lucky7xz/drako/internal/cli"
	"github.com/lucky7xz/drako/internal/config"
	"golang.org/x/term"
)

var (
	pauseFn    = pause
	lookPathFn = exec.LookPath
	commandFn  = exec.Command
	//setenvFn   = os.Setenv
	//unsetenvFn = os.Unsetenv
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
	case "pwsh", "powershell":
		return exec.Command("pwsh", "-NoLogo", "-NoProfile", "-Command", commandStr)
	case "cmd", "cmd.exe":
		return exec.Command("cmd", "/C", commandStr)
	default:
		return exec.Command("bash", "-lc", commandStr)
	}
}

// RunCommand finds the selected command from the loaded config and executes it.
func RunCommand(cfg config.Config, selected string) {
	// Handle special internal commands first
	if strings.HasPrefix(selected, "drako purge") {
		handleInternalPurge(selected)
		// Since purge often resets state or exits, we might want to just return here
		// But standard purge flow ends with Exit(0) usually.
		// If it returns, we might want to pause.
		return
	}

	// cmd will hold the prepared command to run. It's a pointer type; zero value is nil.
	var cmd *exec.Cmd
	// Pointers to per-command overrides; nil means "use default".
	var autoClosePtr *bool
	var debugPtr *bool

	// Default shell to use for string commands (honors config/profile).
	shell_config := cfg.DefaultShell

	// Resolve a top-level command or nested item by name.
	parentCmd, itemCfg, found := FindCommandByName(cfg, selected)
	if found {
		if itemCfg == nil {
			// top-level command
			commandStr := parentCmd.Command
			if commandStr != "" {
				cmd = buildShellCmd(shell_config, commandStr)
				autoClosePtr = parentCmd.AutoCloseExecution
				debugPtr = parentCmd.DebugExecution
			}
		} else {
			// dropdown item
			commandStr := itemCfg.Command
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

	// Sanitize environment variables
	// If EnvWhitelist is configured, we restrict the environment.
	// Otherwise, we inherit the full parent environment (pass-through).
	cmd.Env = PrepareEnv(os.Environ(), cfg.EnvWhitelist)

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

func handleInternalPurge(command string) {
	// Parse the command string manually since we're bypassing the shell
	// Expected format: "drako purge --target core" or "drako purge --interactive"
	parts := strings.Fields(command)

	// We need to parse flags again, locally.
	// Since we can't reuse the flag.FlagSet from cli.HandlePurgeCommand easily without hacking os.Args
	// We will manually construct the opts or helper logic.

	configDir, err := config.GetConfigDir()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		pause("\nPress any key...")
		return
	}

	opts := cli.PurgeOptions{}
	interactive := false

	// Simple manual parsing for the internal use case
	for i := 0; i < len(parts); i++ {
		if parts[i] == "--target" && i+1 < len(parts) {
			val := parts[i+1]
			if val == "core" {
				opts.TargetCore = true
			} else {
				opts.TargetProfile = val
			}
		}
		if parts[i] == "--destroyeverything" {
			opts.DestroyEverything = true
		}
		if parts[i] == "--interactive" {
			interactive = true
		}
	}

	if interactive {
		fmt.Print("Enter profile name to reset (e.g. 'git'): ")
		var name string
		if _, err := fmt.Scanln(&name); err != nil {
			// Handle EOF or empty input gracefully
			fmt.Println("\nInput cancelled.")
			return
		}
		name = strings.TrimSpace(name)
		if name == "" {
			fmt.Println("No profile name provided.")
			return
		}
		opts.TargetProfile = name
	}

	// Execute with confirmation
	confirmMsg := "⚠️  Confirm reset of Core configuration?"
	if opts.TargetProfile != "" {
		confirmMsg = fmt.Sprintf("⚠️  Confirm removal of profile '%s'?", opts.TargetProfile)
	}

	if !cli.ConfirmAction(confirmMsg) {
		return
	}

	if err := cli.PurgeConfig(configDir, opts); err != nil {
		fmt.Printf("\nError: %v\n", err)
		pause("\nPress any key...")
	} else {
		fmt.Printf("\n✓ Operation successful.\n")
		// Since we modified config, we should probably exit to let the loop reload or just exit
		os.Exit(0)
	}
}
