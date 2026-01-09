package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
	case "strip", "--strip":
		HandleStripCommand()
		return true
	case "open", "--open":
		HandleOpenCLI()
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

// HandlePurgeCommand processes the 'drako purge' command from os.Args
func HandlePurgeCommand() {
	// Parse args starting from index 2 (skipping "drako" and "purge")
	if err := ExecutePurge(os.Args[2:]); err != nil {
		os.Exit(1)
	}
}

// ExecutePurge parses flags and executes the purge logic.
// It is exported so internal commands can call it without spawning a subprocess.
func ExecutePurge(args []string) error {
	configDir, err := config.GetConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get config dir: %v\n", err)
		return err
	}

	// Parse purge specific flags
	purgeCmd := flag.NewFlagSet("purge", flag.ContinueOnError)

	var target string
	purgeCmd.StringVar(&target, "target", "", "Target profile to purge (e.g. 'core' for core.profile.toml)")
	purgeCmd.StringVar(&target, "t", "", "Alias for --target")

	var targetConfig bool
	purgeCmd.BoolVar(&targetConfig, "config", false, "Purge config.toml (Core Configuration)")
	purgeCmd.BoolVar(&targetConfig, "c", false, "Alias for --config")

	var interactive bool
	purgeCmd.BoolVar(&interactive, "interactive", false, "Interactively select a profile to purge")
	purgeCmd.BoolVar(&interactive, "i", false, "Alias for --interactive")

	// destroyEverything is dangerous
	destroyEverything := purgeCmd.Bool("destroyeverything", false, "DANGEROUS: Delete entire config directory (no trash)")

	if err := purgeCmd.Parse(args); err != nil {
		return err
	}

	// Safety Check: Reject unrecognized positional arguments
	if purgeCmd.NArg() > 0 {
		fmt.Printf("Error: Unrecognized argument(s): %v\n", purgeCmd.Args())
		printPurgeUsage()
		return fmt.Errorf("unrecognized arguments")
	}

	// Setup options
	opts := PurgeOptions{
		DestroyEverything: *destroyEverything,
		TargetConfig:      targetConfig,
		TargetProfile:     target,
	}

	if interactive {
		if err := runInteractivePurgeSelection(configDir, &opts); err != nil {
			return err
		}
	}

	// Logging setup
	setupPurgeLogging(configDir, opts.DestroyEverything)

	log.Printf("Purge command invoked: %+v", opts)

	// Confirmations
	confirmMsg := ""
	if opts.DestroyEverything {
		confirmMsg = fmt.Sprintf("💀 This will DESTROY EVERYTHING in %s.\n   NO UNDO. NO TRASH.\n   Are you absolutely sure?", configDir)
	} else if opts.TargetConfig {
		confirmMsg = "⚠️  This will reset your Core Configuration (config.toml). Proceed?"
	} else if opts.TargetProfile != "" {
		confirmMsg = fmt.Sprintf("⚠️  This will remove profile '%s'. Proceed?", opts.TargetProfile)
	} else {
		// Strict Safety: If no target, PurgeConfig will error, but we can catch it here too or let it fall through.
		// However, PurgeConfig returns error "no target specified".
		// We should checking opts Validness? No, let PurgeConfig handle it.
		// But wait, if we don't have a target, we verify before asking confirmation?
		// Actually PurgeConfig checks it.
	}

	// If no options set, don't even ask for confirmation, just run (and fail)
	// Or check here to avoid "Confirm action ?" with empty msg?
	if !opts.DestroyEverything && !opts.TargetConfig && opts.TargetProfile == "" {
		printPurgeUsage()
		return fmt.Errorf("no target specified")
	}

	if !ConfirmAction(confirmMsg) {
		log.Printf("Purge cancelled by user")
		return nil
	}

	if err := PurgeConfig(configDir, opts); err != nil {
		log.Printf("Purge failed: %v", err)
		fmt.Fprintf(os.Stderr, "Purge failed: %v\n", err)
		return err
	}

	if opts.DestroyEverything {
		fmt.Printf("\n✓ Full destruction completed - %s has been deleted\n", configDir)
	} else {
		fmt.Printf("\n✓ Purge completed successfully\n")
		fmt.Printf("  Items moved to %s/trash/\n", configDir)
	}
	return nil
}

func printPurgeUsage() {
	fmt.Println("Purge Usage:\n")
	fmt.Println("To purge a specific profile, use:    `drako purge --target <name>`")
	fmt.Println("To purge Core config, use:           `drako purge --config`")
	fmt.Println("To select interactively, use:        `drako purge --interactive`")
	fmt.Println("Destroy ~/.config/drako directory:   `drako purge --destroyeverything`")
}

func setupPurgeLogging(configDir string, destroyEverything bool) {
	if !destroyEverything {
		logPath := filepath.Join(configDir, "drako.log")
		logFile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not open log file: %v\n", err)
		} else {
			// Lealing fd? Ideally we close it. But for a CLI command it's fine.
			// Ideally we return the closer.
			// For now, simplify.
			log.SetOutput(logFile)
		}
	}
}

// runInteractivePurgeSelection handles the UI for selecting a profile
func runInteractivePurgeSelection(configDir string, opts *PurgeOptions) error {
	// Struct to hold profile info
	type startProfile struct {
		DisplayName  string
		RelativePath string
	}
	var validProfiles []startProfile

	// Helper to scan a directory
	scanDir := func(dir string, isInventory bool) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".profile.toml") {
				name := strings.TrimSuffix(e.Name(), ".profile.toml")

				relPath := e.Name()
				label := name
				if isInventory {
					relPath = filepath.Join("inventory", e.Name())
					label = fmt.Sprintf("%s (Inventory)", name)
				} else {
					label = fmt.Sprintf("%s (Equipped)", name)
				}

				validProfiles = append(validProfiles, startProfile{
					DisplayName:  label,
					RelativePath: relPath,
				})
			}
		}
	}

	// 1. Scan Root (Equipped)
	scanDir(configDir, false)

	// 2. Scan Inventory
	scanDir(filepath.Join(configDir, "inventory"), true)

	fmt.Println("Select profile to purge:")
	for i, p := range validProfiles {
		fmt.Printf("%d. %s\n", i+1, p.DisplayName)
	}
	if len(validProfiles) == 0 {
		fmt.Println("(No profiles found)")
		return nil // No-op
	}

	fmt.Print("\nEnter number: ")
	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		fmt.Println("\nInput cancelled.")
		return nil
	}

	// Parse number
	if num, err := strconv.Atoi(input); err == nil {
		if num >= 1 && num <= len(validProfiles) {
			selected := validProfiles[num-1]
			opts.TargetProfile = selected.RelativePath
			return nil
		} else {
			fmt.Println("Invalid selection.")
			return fmt.Errorf("invalid selection")
		}
	} else {
		fmt.Println("Invalid input. Please enter a number.")
		return fmt.Errorf("invalid input")
	}
}

// HandleOpenCLI processes the 'drako open <path>' command from the shell.
func HandleOpenCLI() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: drako open <path>\n")
		os.Exit(1)
	}

	// Reconstruct the argument or take the last one.
	path := os.Args[2]

	if err := OpenPath(path); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening '%s': %v\n", path, err)
		os.Exit(1)
	}

	// Success
	os.Exit(0)
}
