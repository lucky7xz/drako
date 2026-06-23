package cli

import (
	"fmt"
	"log"
	"os"

	"github.com/lucky7xz/drako/internal/config" // drako.chronyx.xyz
	"github.com/lucky7xz/drako/internal/paths"
)

// HandleCLI checks if the program was invoked with CLI arguments (not TUI mode).
// Returns true if a CLI command was handled (or errored out), false if it should proceed to TUI.
func HandleCLI(args []string) bool {
	if len(args) <= 1 {
		return false
	}

	command := args[1]

	switch command {
	case "summon", "--summon":
		HandleSummonCommand(args)
		return true
	case "purge", "--purge":
		HandlePurgeCommand(args)
		return true
	case "spec", "--spec":
		HandleSpecCommand(args)
		return true
	case "stash", "--stash":
		HandleStashCommand(args)
		return true
	case "strip", "--strip":
		HandleStripCommand(args)
		return true
	case "open", "--open":
		HandleOpenCLI(args)
		return true
	case "version", "--version", "-v":
		fmt.Printf("%s %s\n", config.AppName, config.Version())
		return true
	case "help", "--help", "-h":
		PrintUsage()
		return true
	default:
		// If we have an argument that isn't a known command, it's an error.
		// We print usage and exit with 1.
		fmt.Printf("Unknown command or argument: '%s'\n\n", command)
		PrintUsage()
		os.Exit(1)
		return true // unreachable
	}
}

func PrintUsage() {
	fmt.Printf("%s %s\n", config.AppName, config.Version())
	fmt.Printf("Usage: drako <command> [arguments]\n\n")
	table(os.Stdout, []string{"Command", "Description"}, [][]string{
		{"summon <url>", "Summon profile(s) from a URL"},
		{"spec list", "Shows available specs"},
		{"spec <name>", "Apply a spec: move related profiles files to inventory."},
		{"stash <name>", "Stash the related prifle files to inventory."},
		{"strip", "Strip all profiles from inventory, except the default core."},
		{"purge <name>", "Delete profiles or config. Use 'purge -i' for interacive mode"},
		{"version", "Show version information"},
		{"help", "Show this help message"},
		{"open <path/url>", "Open a text file/directory/browser link (sys-defaults rescue)"},
	})
}

// HandleSummonCommand processes the 'drako summon <url>' command
func HandleSummonCommand(args []string) {
	if len(args) < 3 {
		PrintSummonUsage()
		os.Exit(1)
	}

	configDir, err := paths.ConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get config dir: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "could not create config dir: %v\n", err)
		os.Exit(1)
	}

	// Setup logging for CLI command
	logPath := paths.LogFile(configDir)
	logFile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not open log file: %v\n", err)
	} else {
		defer logFile.Close()
		log.SetOutput(logFile)
	}

	sourceURL := args[2]
	log.Printf("Attempting to summon profile from: %s", sourceURL)

	if err := SummonProfile(sourceURL, configDir); err != nil {
		log.Printf("Summon failed: %v", err)
		fmt.Fprintf(os.Stderr, "Summon failed: %v\n", err)
		os.Exit(1)
	}

	inventoryDir := paths.InventoryDir(configDir)
	fmt.Printf("\n✓ Profile summoned successfully to %s\n", inventoryDir)
	os.Exit(0)
}

// PrintSummonUsage prints the usage information for the summon command
func PrintSummonUsage() {
	fmt.Fprintf(os.Stderr, "Usage: drako summon <url>\n")
	fmt.Fprintf(os.Stderr, "\nSummoned profiles are saved to ~/.config/drako/inventory/\n")
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  # Summon a single profile file:\n")
	fmt.Fprintf(os.Stderr, "  drako summon https://raw.githubusercontent.com/user/repo/main/profile.profile.toml\n")
	fmt.Fprintf(os.Stderr, "\n  # Summon from a git repository (finds all .profile.toml files):\n")
	fmt.Fprintf(os.Stderr, "  drako summon git@github.com:user/repo.git\n")
	fmt.Fprintf(os.Stderr, "  drako summon https://github.com/user/repo.git\n")
}

// HandleOpenCLI processes the 'drako open <path>' command from the shell.
func HandleOpenCLI(args []string) {
	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: drako open <path>\n")
		os.Exit(1)
	}

	// Reconstruct the argument or take the last one.
	path := args[2]

	if err := OpenPath(path); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening '%s': %v\n", path, err)
		os.Exit(1)
	}

	// Success
	os.Exit(0)
}
