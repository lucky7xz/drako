package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

func runCommand(config Config, selected string) {
	var cmd *exec.Cmd
	isInteractive := true
	holdAfter := false


	found := false
	for _, customCmd := range config.Commands {
		if customCmd.Name == selected {
			commandStr := strings.ReplaceAll(customCmd.Command, "{dR4ko_path}", config.DR4koPath)
			if commandStr != "" {
				cmd = exec.Command("sh", "-c", commandStr)
				isInteractive = customCmd.Interactive
				holdAfter = customCmd.HoldAfter
			}
			found = true
			break
		}
		// Check dropdown items
		for _, item := range customCmd.Items {
			if item.Name == selected {
				commandStr := strings.ReplaceAll(item.Command, "{dR4ko_path}", config.DR4koPath)
				if commandStr != "" {
					cmd = exec.Command("sh", "-c", commandStr)
					isInteractive = item.Interactive
					holdAfter = item.HoldAfter
				}
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if cmd == nil {
		switch selected {
		default:
			if path, err := exec.LookPath(selected); err == nil {
				cmd = exec.Command(path)
			} else {
				log.Printf("Executable not found in PATH: %s", selected)
				return
			}
		}
	}

	if cmd == nil {
		return
	}

	if isInteractive {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("\n--- Command Failed ---\n")
			fmt.Printf("Command: '%s'\n", selected)
			fmt.Printf("Error: %v\n", err)
			fmt.Printf("\nPress Enter to return to the application.")
			fmt.Scanln()
		} else if holdAfter {
			fmt.Printf("\n--- Command Finished ---\n")
			fmt.Printf("Command: '%s'\n", selected)
			fmt.Printf("\nPress Enter to return to the application.")
			fmt.Scanln()
		}
		return
	}

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
}
