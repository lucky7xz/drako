package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// handleCLI checks if the program was invoked with CLI arguments (not TUI mode).
// Returns true if a CLI command was handled, false if it should proceed to TUI.
func handleCLI() bool {
	if len(os.Args) <= 1 {
		return false
	}

	command := os.Args[1]

	switch command {
	case "summon", "--summon":
		handleSummonCommand()
		return true
	case "purge", "--purge":
		handlePurgeCommand()
		return true
	default:
		return false
	}
}

// handleSummonCommand processes the 'drako summon <url>' command
func handleSummonCommand() {
	if len(os.Args) < 3 {
		printSummonUsage()
		os.Exit(1)
	}

	configDir, err := getConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get config dir: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "could not create config dir: %v\n", err)
		os.Exit(1)
	}

	// Setup logging for CLI command
	logPath := filepath.Join(configDir, "drako.log")
	logFile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not open log file: %v\n", err)
	} else {
		defer logFile.Close()
		log.SetOutput(logFile)
	}

	sourceURL := os.Args[2]
	log.Printf("Attempting to summon profile from: %s", sourceURL)
	
	if err := summonProfile(sourceURL, configDir); err != nil {
		log.Printf("Summon failed: %v", err)
		fmt.Fprintf(os.Stderr, "Summon failed: %v\n", err)
		os.Exit(1)
	}

	inventoryDir := filepath.Join(configDir, "inventory")
	fmt.Printf("\n✓ Profile summoned successfully to %s\n", inventoryDir)
	os.Exit(0)
}

// printSummonUsage prints the usage information for the summon command
func printSummonUsage() {
	fmt.Fprintf(os.Stderr, "Usage: drako summon <url>\n")
	fmt.Fprintf(os.Stderr, "\nSummoned profiles are saved to ~/.config/drako/inventory/\n")
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  # Summon a single profile file:\n")
	fmt.Fprintf(os.Stderr, "  drako summon https://raw.githubusercontent.com/user/repo/main/profile.profile.toml\n")
	fmt.Fprintf(os.Stderr, "\n  # Summon from a git repository (finds all .profile.toml files):\n")
	fmt.Fprintf(os.Stderr, "  drako summon git@github.com:user/repo.git\n")
	fmt.Fprintf(os.Stderr, "  drako summon https://github.com/user/repo.git\n")
}

// handlePurgeCommand processes the 'drako purge' command
func handlePurgeCommand() {
	configDir, err := getConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get config dir: %v\n", err)
		os.Exit(1)
	}

	// Check for --all flag
	nukeAll := false
	if len(os.Args) > 2 && (os.Args[2] == "--all" || os.Args[2] == "-a") {
		nukeAll = true
	}

	// Setup logging for CLI command (if not nuking everything)
	if !nukeAll {
		logPath := filepath.Join(configDir, "drako.log")
		logFile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not open log file: %v\n", err)
		} else {
			defer logFile.Close()
			log.SetOutput(logFile)
		}
	}

	if nukeAll {
		log.Printf("Purge --all command invoked")
	} else {
		log.Printf("Purge command invoked")
	}
	
	if err := purgeConfig(configDir, nukeAll); err != nil {
		log.Printf("Purge failed: %v", err)
		fmt.Fprintf(os.Stderr, "Purge failed: %v\n", err)
		os.Exit(1)
	}

	if nukeAll {
		fmt.Printf("\n✓ Full purge completed - %s has been deleted\n", configDir)
	} else {
		fmt.Printf("\n✓ Purge completed successfully\n")
		fmt.Printf("✓ config.toml has been preserved at %s/config.toml\n", configDir)
	}
	os.Exit(0)
}

