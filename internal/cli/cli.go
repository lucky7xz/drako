package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/lucky7xz/drako/internal/config" // drako.chronyx.xyz
)

// HandleCLI checks if the program was invoked with CLI arguments (not TUI mode).
// Returns true if a CLI command was handled, false if it should proceed to TUI.
func HandleCLI() bool {
	if len(os.Args) <= 1 {
		return false
	}

	command := os.Args[1]

	switch command {
	case "summon", "--summon":
		HandleSummonCommand()
		return true
	case "purge", "--purge":
		HandlePurgeCommand()
		return true
	case "spec", "--spec":
		HandleSpecCommand()
		return true
	case "stash", "--stash":
		HandleStashCommand()
		return true
	default:
		return false
	}
}

// HandleSummonCommand processes the 'drako summon <url>' command
func HandleSummonCommand() {
	if len(os.Args) < 3 {
		PrintSummonUsage()
		os.Exit(1)
	}

	configDir, err := config.GetConfigDir()
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

	if err := SummonProfile(sourceURL, configDir); err != nil {
		log.Printf("Summon failed: %v", err)
		fmt.Fprintf(os.Stderr, "Summon failed: %v\n", err)
		os.Exit(1)
	}

	inventoryDir := filepath.Join(configDir, "inventory")
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

// HandlePurgeCommand processes the 'drako purge' command
func HandlePurgeCommand() {
	configDir, err := config.GetConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get config dir: %v\n", err)
		os.Exit(1)
	}

	// Parse purge specific flags
	// We need a custom flagset to parse args after "drako purge"
	purgeCmd := flag.NewFlagSet("purge", flag.ExitOnError)
	target := purgeCmd.String("target", "", "Target to purge: 'core' or profile name (e.g. 'git')")
	destroyEverything := purgeCmd.Bool("destroyeverything", false, "DANGEROUS: Delete entire config directory (no trash)")
	interactive := purgeCmd.Bool("interactive", false, "Interactively select a profile to purge")

	// Parse args starting from index 2 (skipping "drako" and "purge")
	if err := purgeCmd.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Setup options based on flags
	opts := PurgeOptions{
		DestroyEverything: *destroyEverything,
	}

	if *interactive {
		fmt.Print("Enter profile name to purge (e.g. 'git'): ")
		var name string
		if _, err := fmt.Scanln(&name); err != nil {
			fmt.Println("\nInput cancelled.")
			os.Exit(0)
		}
		name = filepath.Base(name) // Basic sanitization
		if name == "" {
			fmt.Println("No profile name provided.")
			os.Exit(1)
		}
		opts.TargetProfile = name
	} else if *target == "core" {
		opts.TargetCore = true
	} else if *target != "" {
		opts.TargetProfile = *target
	} else if !*destroyEverything {
		// Legacy behavior: "drako purge" without args -> Standard cleanup (preserve config.toml)
		// BUT WAIT, the user wants "Full Circle" safe purge.
		// Let's default to standard cleanup but now it MOVES to trash instead of delete.
		// And it EXCLUDES config.toml by default (just like old PurgeConfig(..., false))
		// Wait, if no target is specified, do we clean everything else?
		// Old behavior: "PurgeConfig(..., false)" -> Deleted everything EXCEPT config.toml
		// Let's keep that behavior for "drako purge" with no args, but SAFE (trash).
	}

	// Logging setup
	if !opts.DestroyEverything {
		logPath := filepath.Join(configDir, "drako.log")
		logFile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not open log file: %v\n", err)
		} else {
			defer logFile.Close()
			log.SetOutput(logFile)
		}
	}

	log.Printf("Purge command invoked: %+v", opts)

	// Confirmations
	confirmMsg := ""
	if opts.DestroyEverything {
		confirmMsg = fmt.Sprintf("💀 This will DESTROY EVERYTHING in %s.\n   NO UNDO. NO TRASH.\n   Are you absolutely sure?", configDir)
	} else if opts.TargetCore {
		confirmMsg = "⚠️  This will reset your Core configuration (config.toml). Proceed?"
	} else if opts.TargetProfile != "" {
		confirmMsg = fmt.Sprintf("⚠️  This will remove profile '%s'. Proceed?", opts.TargetProfile)
	} else {
		confirmMsg = fmt.Sprintf("⚠️  This will move all profiles and data in %s to trash\n   (config.toml will be preserved). Proceed?", configDir)
	}

	if !ConfirmAction(confirmMsg) {
		log.Printf("Purge cancelled by user")
		os.Exit(0)
	}

	if err := PurgeConfig(configDir, opts); err != nil {
		log.Printf("Purge failed: %v", err)
		fmt.Fprintf(os.Stderr, "Purge failed: %v\n", err)
		os.Exit(1)
	}

	if opts.DestroyEverything {
		fmt.Printf("\n✓ Full destruction completed - %s has been deleted\n", configDir)
	} else {
		fmt.Printf("\n✓ Purge completed successfully\n")
		fmt.Printf("  Items moved to %s/trash/\n", configDir)
	}
	os.Exit(0)
}
