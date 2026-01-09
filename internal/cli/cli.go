package cli

import (
	"bufio"
	"flag"
	"fmt"
	"io"
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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		printPurgeUsage()
		os.Exit(1)
	}
}

// ExecutePurge parses flags and executes the purge logic.
// It is exported so internal commands can call it without spawning a subprocess.
func ExecutePurge(args []string) error {
	opts, interactive, err := ParsePurgeFlags(args)
	if err != nil {
		return err
	}

	if interactive {
		configDir, err := config.GetConfigDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not get config dir: %v\n", err)
			return err
		}
		// Pass stdin/stdout for interactive mode
		if err := runInteractivePurgeSelection(configDir, opts, os.Stdin, os.Stdout); err != nil {
			return err
		}
	}

	// Logging setup
	setupPurgeLogging(opts.DestroyEverything)

	log.Printf("Purge command invoked: %+v", opts)

	// Confirmations
	confirmMsg := ""
	configDir, _ := config.GetConfigDir() // Ignore error as we checked above or it will fail in PurgeConfig

	if opts.DestroyEverything {
		confirmMsg = fmt.Sprintf("💀 This will DESTROY EVERYTHING in %s.\n   NO UNDO. NO TRASH.\n   Are you absolutely sure?", configDir)
	} else if opts.TargetConfig {
		confirmMsg = "⚠️  This will reset your Core Configuration (config.toml). Proceed?"
	} else if len(opts.TargetProfiles) > 0 {
		confirmMsg = fmt.Sprintf("⚠️  This will remove %d profile(s): %s. Proceed?", len(opts.TargetProfiles), strings.Join(opts.TargetProfiles, ", "))
	} else {
		// Strict Safety: If no target is specified, PurgeConfig will error.
		// We catch this case below to provide a helpful usage message.
	}

	if !opts.DestroyEverything && !opts.TargetConfig && len(opts.TargetProfiles) == 0 {
		printPurgeUsage()
		return fmt.Errorf("no target specified")
	}

	// Interactive mode handles its own confirmations per item, unless it's a bulk action from flags.
	// If flags were used, we confirm once here.
	if !interactive {
		if !ConfirmAction(confirmMsg) {
			log.Printf("Purge cancelled by user")
			return nil
		}
	}

	if err := PurgeConfig(configDir, *opts); err != nil {
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
	fmt.Println("Purge Usage:")
	fmt.Println("To purge a specific profile, use:    `drako purge --target <name>`")
	fmt.Println("To purge Core config, use:           `drako purge --config`")
	fmt.Println("To select interactively, use:        `drako purge --interactive`")
	fmt.Println("Destroy ~/.config/drako directory:   `drako purge --destroyeverything`")
}

func setupPurgeLogging(destroyEverything bool) {
	if !destroyEverything {
		configDir, err := config.GetConfigDir()
		if err != nil {
			return
		}
		logPath := filepath.Join(configDir, "drako.log")
		logFile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not open log file: %v\n", err)
		} else {
			// Note: We leave the log file open for the duration of the command.
			log.SetOutput(logFile)
		}
	}
}

// IO Dependencies injected
func runInteractivePurgeSelection(configDir string, opts *PurgeOptions, input io.Reader, output io.Writer) error {
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

	if len(validProfiles) == 0 {
		fmt.Fprintln(output, "(No profiles found)")
		return nil // No-op
	}

	fmt.Fprintln(output, "Select profile(s) to purge:")
	for i, p := range validProfiles {
		fmt.Fprintf(output, "%d. %s\n", i+1, p.DisplayName)
	}

	fmt.Fprint(output, "\nEnter numbers (e.g. '1, 3', '1-5'): ")

	// Read full line
	bufReader := bufio.NewReader(input)
	line, err := bufReader.ReadString('\n')
	if err != nil {
		fmt.Fprintln(output, "\nInput cancelled.")
		return nil
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return fmt.Errorf("no selection made")
	}

	// Parse Batch Input
	// Supports: "1", "1,3", "1, 3", "1-3", "1, 3-5"
	parts := strings.Split(line, ",")
	var selectedIndices []int

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check for range
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				fmt.Fprintf(output, "Invalid range format: %s\n", part)
				continue
			}
			start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err1 != nil || err2 != nil {
				fmt.Fprintf(output, "Invalid numbers in range: %s\n", part)
				continue
			}
			if start > end {
				start, end = end, start // swap
			}
			for i := start; i <= end; i++ {
				selectedIndices = append(selectedIndices, i)
			}
		} else {
			// Single number
			num, err := strconv.Atoi(part)
			if err != nil {
				fmt.Fprintf(output, "Invalid number: %s\n", part)
				continue
			}
			selectedIndices = append(selectedIndices, num)
		}
	}

	// Process selections
	count := 0
	// Deduplicate indices using map
	seen := make(map[int]bool)

	for _, idx := range selectedIndices {
		if seen[idx] {
			continue
		}
		seen[idx] = true

		if idx < 1 || idx > len(validProfiles) {
			fmt.Fprintf(output, "Warning: %d is out of range (1-%d)\n", idx, len(validProfiles))
			continue
		}

		profile := validProfiles[idx-1]

		// Individual Confirmation
		fmt.Fprintf(output, "Delete %s? [y/N]: ", profile.DisplayName)
		confirmRaw, _ := bufReader.ReadString('\n')
		confirm := strings.ToLower(strings.TrimSpace(confirmRaw))

		if confirm == "y" || confirm == "yes" {
			opts.TargetProfiles = append(opts.TargetProfiles, profile.RelativePath)
			count++
		} else {
			fmt.Fprintln(output, "Skipped.")
		}
	}

	if count == 0 {
		fmt.Fprintln(output, "No profiles selected for deletion.")
	}

	return nil
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

// ParsePurgeFlags processes raw arguments and returns PurgeOptions and interactive state.
// This separates valid parsing from execution logic, enabling easier testing.
func ParsePurgeFlags(args []string) (*PurgeOptions, bool, error) {
	purgeCmd := flag.NewFlagSet("purge", flag.ContinueOnError)

	var target string
	purgeCmd.StringVar(&target, "target", "", "Target profile to purge")
	purgeCmd.StringVar(&target, "t", "", "Alias for --target")

	var targetConfig bool
	purgeCmd.BoolVar(&targetConfig, "config", false, "Purge config.toml")
	purgeCmd.BoolVar(&targetConfig, "c", false, "Alias for --config")

	var interactive bool
	purgeCmd.BoolVar(&interactive, "interactive", false, "Interactive mode")
	purgeCmd.BoolVar(&interactive, "i", false, "Alias for --interactive")

	destroyEverything := purgeCmd.Bool("destroyeverything", false, "Destroy everything")

	if err := purgeCmd.Parse(args); err != nil {
		return nil, false, err
	}

	if purgeCmd.NArg() > 0 {
		return nil, false, fmt.Errorf("unrecognized arguments: %v", purgeCmd.Args())
	}

	opts := &PurgeOptions{
		DestroyEverything: *destroyEverything,
		TargetConfig:      targetConfig,
		TargetProfiles:    []string{},
	}

	if target != "" {
		opts.TargetProfiles = append(opts.TargetProfiles, target)
	}

	return opts, interactive, nil
}
