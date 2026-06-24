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

	"github.com/lucky7xz/drako/internal/paths"
	"github.com/lucky7xz/drako/internal/profiles"
)

// PurgeOptions defines the scope of the purge operation
type PurgeOptions struct {
	DestroyEverything bool     // Nuke ~/.config/drako entirely
	TargetProfiles    []string // Delete/Move specific profiles (e.g. "git", "core")
	TargetConfig      bool     // Reset config.toml
	TargetLogs        bool     // Purge logs (history.log and drako.log)
}

// PurgeConfig executes the purge operation based on the options.
// It moves files to ~/.config/drako/trash/ instead of deleting them,
// unless DestroyEverything is true.
func PurgeConfig(configDir string, opts PurgeOptions) error {
	if opts.DestroyEverything {
		log.Printf("Starting FULL purge (destroy everything) for: %s", configDir)
		return performFullNuke(configDir)
	}

	// Safety check: if no targets selected
	if len(opts.TargetProfiles) == 0 && !opts.TargetConfig && !opts.TargetLogs && !opts.DestroyEverything {
		// This should be caught by caller, but good to be safe
		return fmt.Errorf("no target specified (use --target, --config, --logs, or --destroyeverything)")
	}

	// Case 1: Reset Core Config (config.toml)
	if opts.TargetConfig {
		log.Printf("Purging Core config (config.toml)")
		if err := profiles.MoveToTrash(configDir, paths.ConfigFileName); err != nil {
			log.Printf("Failed to purge config.toml: %v", err)
		} else {
			fmt.Printf("  ✓ Moved %s to trash\n", paths.ConfigFileName)
		}
	}

	// Case 2: Purge Logs
	if opts.TargetLogs {
		log.Printf("Purging Logs (Permanent Delete)")
		for _, path := range paths.LogFiles(configDir) {
			name := filepath.Base(path)
			// Permanent deletion as requested
			if err := os.Remove(path); err != nil {
				// Don't log normal "not found" errors to avoid clutter, unless debugging
				if !os.IsNotExist(err) {
					log.Printf("Failed to delete log %s: %v", name, err)
				}
			} else {
				fmt.Printf("  💀 Deleted %s\n", name)
			}
		}
	}

	// Case 3: Target Specific Profiles
	for _, target := range opts.TargetProfiles {
		log.Printf("Purging Profile: %s", target)
		filename := target
		if filepath.Ext(filename) != ".toml" {
			filename = filename + ".profile.toml"
		}
		if err := profiles.MoveToTrash(configDir, filename); err != nil {
			log.Printf("Failed to purge %s: %v", target, err)
		} else {
			fmt.Printf("  ✓ Moved %s to trash\n", filename)
		}
	}

	return nil
}

// performFullNuke implements the "Destroy Everything" logic (Old --all)
func performFullNuke(configDir string) error {
	// Confirm existence
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return fmt.Errorf("config directory does not exist: %s", configDir)
	}

	fmt.Printf("\n💀 DESTROYING EVERYTHING in %s\n", configDir)
	// The caller (HandlePurgeCommand) should have asked for confirmation.

	if err := os.RemoveAll(configDir); err != nil {
		return fmt.Errorf("failed to destroy config directory: %w", err)
	}
	return nil
}

// HandlePurgeCommand processes the 'drako purge' command from args
func HandlePurgeCommand(args []string) {
	// Parse args starting from index 2 (skipping "drako" and "purge")
	if err := ExecutePurge(args[2:]); err != nil {
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
		configDir, err := paths.ConfigDir()
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
	configDir, _ := paths.ConfigDir() // Ignore error as we checked above or it will fail in PurgeConfig

	if opts.DestroyEverything {
		confirmMsg = fmt.Sprintf("💀 This will DESTROY EVERYTHING in %s.\n   NO UNDO. NO TRASH.\n   Are you absolutely sure?", configDir)
	} else if opts.TargetConfig {
		confirmMsg = "⚠️  This will reset your Core Configuration (config.toml). Proceed?"
	} else if opts.TargetLogs {
		confirmMsg = "💀 This will PERMANENTLY DELETE your log files (history.log, drako.log). Proceed?"
	} else if len(opts.TargetProfiles) > 0 {
		confirmMsg = fmt.Sprintf("⚠️  This will remove %d profile(s): %s. Proceed?", len(opts.TargetProfiles), strings.Join(opts.TargetProfiles, ", "))
	} else {
		// Strict Safety: If no target is specified, PurgeConfig will error.
		// We catch this case below to provide a helpful usage message.
	}

	if !opts.DestroyEverything && !opts.TargetConfig && !opts.TargetLogs && len(opts.TargetProfiles) == 0 {
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
		configDir, err := paths.ConfigDir()
		if err != nil {
			return
		}
		logPath := paths.LogFile(configDir)
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
		Name         string
		Location     string
		RelativePath string
	}
	var validProfiles []startProfile

	// Helper to scan a directory
	scanDir := func(dir, location string) {
		entries, err := profiles.List(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			relPath := e.File
			if location == "Inventory" {
				relPath = filepath.Join("inventory", e.File)
			}
			validProfiles = append(validProfiles, startProfile{
				Name:         e.Name,
				Location:     location,
				RelativePath: relPath,
			})
		}
	}

	// 1. Scan Root (Equipped)
	scanDir(configDir, "Equipped")

	// 2. Scan Inventory
	scanDir(paths.InventoryDir(configDir), "Inventory")

	if len(validProfiles) == 0 {
		fmt.Fprintln(output, "(No profiles found)")
		return nil // No-op
	}

	fmt.Fprintln(output, "Select profile(s) to purge:")
	selectRows := make([][]string, len(validProfiles))
	for i, p := range validProfiles {
		selectRows[i] = []string{strconv.Itoa(i + 1), p.Name, p.Location}
	}
	table(output, []string{"#", "Profile", "Location"}, selectRows)

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
		fmt.Fprintf(output, "Delete %s (%s)? [y/N]: ", profile.Name, profile.Location)
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

	var targetLogs bool
	purgeCmd.BoolVar(&targetLogs, "logs", false, "Purge logs")
	purgeCmd.BoolVar(&targetLogs, "l", false, "Alias for --logs")

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
		TargetLogs:        targetLogs,
		TargetProfiles:    []string{},
	}

	if target != "" {
		opts.TargetProfiles = append(opts.TargetProfiles, target)
	}

	return opts, interactive, nil
}
