package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

func boolOrDefault(ptr *bool, def bool) bool {
	if ptr == nil { return def }
	return *ptr
}

func runCommand(config Config, selected string) {
	var cmd *exec.Cmd
	var autoClosePtr *bool
	var debugPtr *bool

	found := false
	for _, customCmd := range config.Commands {
		if customCmd.Name == selected {
			commandStr := strings.ReplaceAll(customCmd.Command, "{dR4ko_path}", config.DR4koPath)
			if commandStr != "" {
				cmd = exec.Command("sh", "-c", commandStr)
				autoClosePtr = customCmd.AutoCloseExecution
				debugPtr = customCmd.DebugExecution
			}
			found = true
			break
		}
		for _, item := range customCmd.Items {
			if item.Name == selected {
				commandStr := strings.ReplaceAll(item.Command, "{dR4ko_path}", config.DR4koPath)
				if commandStr != "" {
					cmd = exec.Command("sh", "-c", commandStr)
					autoClosePtr = item.AutoCloseExecution
					debugPtr = item.DebugExecution
				}
				found = true
				break
			}
		}
		if found { break }
	}

	if cmd == nil {
		if path, err := exec.LookPath(selected); err == nil {
			cmd = exec.Command(path)
		} else {
			log.Printf("Executable not found in PATH: %s", selected)
			return
		}
	}

	autoClose := boolOrDefault(autoClosePtr, true)
	debug := boolOrDefault(debugPtr, false)

	if debug {
		// Debug route: buffer output, print it, and ALWAYS halt like errors
		output, err := cmd.CombinedOutput()
		fmt.Printf("\n--- Command Output ---\n")
		fmt.Printf("Command: '%s'\n\n", selected)
		fmt.Print(string(output))
		if err != nil {
			fmt.Printf("\n--- Command Failed ---\n")
			fmt.Printf("Error: %v\n", err)
		}
		fmt.Printf("\nPress Enter to return to the application.")
		fmt.Scanln()
		return
	}

	// Live route: wire stdio to terminal
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("\n--- Command Failed ---\n")
		fmt.Printf("Command: '%s'\n", selected)
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("\nPress Enter to return to the application.")
		fmt.Scanln()
		return
	}

	// Success in live mode: hold only if autoClose is false
	if !autoClose {
		fmt.Printf("\n--- Command Finished ---\n")
		fmt.Printf("Command: '%s'\n", selected)
		fmt.Printf("\nPress Enter to return to the application.")
		fmt.Scanln()
	}
}
