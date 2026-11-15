package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lucky7xz/drako/internal/equip"
	"github.com/lucky7xz/drako/internal/lair"
)

// ErrPurgeCancelled reports that the user declined or picked nothing.
var ErrPurgeCancelled = errors.New("purge cancelled")

var errNoPurgeTarget = errors.New("no target specified")

// PurgeOptions defines the scope of the purge operation
type PurgeOptions struct {
	DestroyEverything bool     // Nuke ~/.config/drako entirely
	TargetProfiles    []string // files to trash, relative to the config dir (see purgeFile)
	TargetConfig      bool     // Reset config.toml
	TargetLogs        bool     // Purge logs (history.log and drako.log)
}

// resetCore writes a fresh config.toml and restores the core deck if it is gone.
// It must not leave config.toml missing: that is a first run, restoring every deck.
func resetCore(configDir string) []error {
	var failed []error
	if _, err := lair.WriteSettingsTemplate(configDir); err != nil {
		failed = append(failed, fmt.Errorf("could not write a fresh %s: %w", lair.ConfigFileName, err))
	} else {
		fmt.Printf("  ✓ Wrote a fresh %s\n", lair.ConfigFileName)
	}
	if restored, err := lair.RestoreCoreProfile(configDir); err != nil {
		failed = append(failed, fmt.Errorf("could not restore core.profile.toml: %w", err))
	} else if restored {
		fmt.Println("  ✓ Restored core.profile.toml")
	}
	return failed
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

	var failed []error

	// Case 1: Reset Core Config (config.toml)
	if opts.TargetConfig {
		log.Printf("Purging Core config (config.toml)")
		target := lair.LinkTarget(configDir, lair.ConfigFileName)
		if err := lair.MoveToTrash(configDir, lair.ConfigFileName); err != nil {
			failed = append(failed, fmt.Errorf("could not move %s to trash: %w", lair.ConfigFileName, err))
		} else {
			fmt.Println(trashedLine(lair.ConfigFileName, target))
			failed = append(failed, resetCore(configDir)...)
		}
	}

	// Case 2: Purge Logs
	if opts.TargetLogs {
		log.Printf("Purging Logs (Permanent Delete)")
		targets := map[string]string{}
		for _, name := range lair.LogFileNames {
			targets[name] = lair.LinkTarget(configDir, name)
		}
		deleted, err := lair.DeleteLogs(configDir)
		for _, name := range deleted {
			fmt.Println(deletedLine(name, targets[name]))
		}
		if err != nil {
			failed = append(failed, err)
		}
	}

	// Case 3: Target Specific Profiles
	for _, file := range opts.TargetProfiles {
		log.Printf("Purging Profile: %s", file)
		target := lair.LinkTarget(configDir, file)
		if err := lair.MoveToTrash(configDir, file); err != nil {
			failed = append(failed, fmt.Errorf("could not move %s to trash: %w", file, err))
		} else {
			fmt.Println(trashedLine(file, target))
			// A purged profile is no longer equipped, so a lock still aimed at
			// it would drop the next launch into rescue. Covers --interactive
			// too: it appends into this same target list.
			unlockIfLocked(configDir, file)
		}
	}

	return errors.Join(failed...)
}

// trashedLine says that file went to trash; target is what it linked to, ""
// for an ordinary file.
func trashedLine(file, target string) string {
	if target != "" {
		return fmt.Sprintf("  ✓ Moved the link %s to trash (%s is untouched)", file, target)
	}
	return fmt.Sprintf("  ✓ Moved %s to trash", file)
}

// deletedLine is trashedLine for a file that is deleted for good.
func deletedLine(name, target string) string {
	if target != "" {
		return fmt.Sprintf("  💀 Removed the link %s (%s is untouched)", name, target)
	}
	return fmt.Sprintf("  💀 Deleted %s", name)
}

// destroyedLine says the config folder is gone; target is what it linked to,
// "" for an ordinary folder.
func destroyedLine(configDir, target string) string {
	if target != "" {
		return fmt.Sprintf("✓ Removed the link %s (%s and its files are untouched)", configDir, target)
	}
	return fmt.Sprintf("✓ Full destruction completed - %s has been deleted", configDir)
}

// performFullNuke implements the "Destroy Everything" logic (Old --all)
func performFullNuke(configDir string) error {
	if lair.DirLinkTarget(configDir) == "" {
		fmt.Printf("\n💀 DESTROYING EVERYTHING in %s\n", configDir)
	}
	// The caller (HandlePurgeCommand) should have asked for confirmation.

	if err := lair.Destroy(configDir); err != nil {
		return fmt.Errorf("failed to destroy config directory: %w", err)
	}
	return nil
}

// HandlePurgeCommand processes the 'drako purge' command from args.
// Returns the process exit code.
func HandlePurgeCommand(args []string) int {
	err := ExecutePurge(args[2:])
	switch {
	case err == nil:
		return 0
	case errors.Is(err, ErrPurgeCancelled):
		fmt.Println("Purge cancelled.")
	default:
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		if errors.Is(err, errNoPurgeTarget) {
			printPurgeUsage(os.Stderr)
		}
	}
	return 1
}

// ExecutePurge parses flags and executes the purge logic: nil once the purge
// ran, ErrPurgeCancelled when the user declined.
// It is exported so internal commands can call it without spawning a subprocess.
func ExecutePurge(args []string) error {
	opts, interactive, err := ParsePurgeFlags(args)
	if err != nil {
		return err
	}
	configDir, err := lair.ConfigDir()
	if err != nil {
		return fmt.Errorf("could not get config dir: %w", err)
	}
	// Resolved before the prompt, so it names the real file and a typo never asks.
	for i, target := range opts.TargetProfiles {
		if opts.TargetProfiles[i], err = purgeFile(configDir, target); err != nil {
			return err
		}
	}

	// Flag scopes get one confirmation; profiles picked with -i are confirmed one by one.
	confirmMsg := purgeConfirmMessage(configDir, *opts)
	if interactive {
		if err := runInteractivePurgeSelection(configDir, opts, confirmReader, os.Stdout); err != nil {
			return err
		}
	}
	if !opts.DestroyEverything && !opts.TargetConfig && !opts.TargetLogs && len(opts.TargetProfiles) == 0 {
		if interactive {
			return ErrPurgeCancelled
		}
		return errNoPurgeTarget
	}

	// A full nuke must not hold the log file open inside the dir it deletes.
	if !opts.DestroyEverything {
		quietLogs()
	}
	log.Printf("Purge command invoked: %+v", opts)

	if confirmMsg != "" && !ConfirmAction(confirmMsg) {
		log.Printf("Purge cancelled by user")
		return ErrPurgeCancelled
	}

	dirTarget := lair.DirLinkTarget(configDir) // asked while the folder is still there
	if err := PurgeConfig(configDir, *opts); err != nil {
		log.Printf("Purge failed: %v", err)
		return err
	}

	if opts.DestroyEverything {
		fmt.Printf("\n%s\n", destroyedLine(configDir, dirTarget))
	} else {
		fmt.Printf("\n✓ Purge completed successfully\n")
		fmt.Printf("  Items moved to %s/trash/\n", configDir)
	}
	return nil
}

// purgeFile finds the file a --target means, relative to configDir: the file it
// names (bare names get .profile.toml) if that exists, else the deck it names.
func purgeFile(configDir, target string) (string, error) {
	file := target
	if filepath.Ext(file) != ".toml" {
		file += equip.ProfileSuffix
	}
	if lair.Exists(configDir, file) {
		return file, nil
	}
	for _, loc := range []equip.Location{equip.Equipped, equip.Inventory} {
		dir := lair.LocationDir(configDir, loc)
		entries, err := lair.List(configDir, loc)
		if err != nil {
			return "", err
		}
		for _, e := range entries {
			if e.Norm == equip.NormalizeName(target) {
				return filepath.Rel(configDir, filepath.Join(dir, e.File))
			}
		}
	}
	return "", fmt.Errorf("no deck named %s", target)
}

// purgeConfirmMessage names every scope the flags selected; "" when none did.
func purgeConfirmMessage(configDir string, opts PurgeOptions) string {
	if opts.DestroyEverything {
		if target := lair.DirLinkTarget(configDir); target != "" {
			return fmt.Sprintf("💀 %s is a link to %s.\n   This removes the link only. Everything in %s stays.\n   Proceed?", configDir, target, target)
		}
		links := ""
		switch n := lair.CountLinks(configDir); {
		case n == 1:
			links = "   1 file in it is a link: it loses only the link, the file it points to stays.\n"
		case n > 1:
			links = fmt.Sprintf("   %d files in it are links: those lose only the link, the files they point to stay.\n", n)
		}
		return fmt.Sprintf("💀 This will DESTROY EVERYTHING in %s.\n   NO UNDO. NO TRASH.\n%s   Are you absolutely sure?", configDir, links)
	}
	// A link gets its own line: what goes is the link, and the file stays.
	var lines []string
	if opts.TargetConfig {
		if target := lair.LinkTarget(configDir, lair.ConfigFileName); target != "" {
			lines = append(lines, "⚠️  reset your Core Configuration",
				fmt.Sprintf("    %s is a link to %s.", lair.ConfigFileName, target),
				fmt.Sprintf("    The link goes to trash and a fresh %s takes its place.", lair.ConfigFileName),
				"    The linked file stays as it is; drako stops reading it.")
		} else {
			lines = append(lines, "⚠️  reset your Core Configuration (config.toml goes to trash)")
		}
	}
	if opts.TargetLogs {
		lines = append(lines, "💀 PERMANENTLY DELETE your log files (history.log, drako.log)")
		for _, name := range lair.LogFileNames {
			if target := lair.LinkTarget(configDir, name); target != "" {
				lines = append(lines, fmt.Sprintf("    %s is a link: only the link is removed, %s stays", name, target))
			}
		}
	}
	if n := len(opts.TargetProfiles); n > 0 {
		lines = append(lines, fmt.Sprintf("⚠️  move %d profile(s) to trash: %s", n, strings.Join(opts.TargetProfiles, ", ")))
		for _, file := range opts.TargetProfiles {
			if target := lair.LinkTarget(configDir, file); target != "" {
				lines = append(lines, fmt.Sprintf("    %s is a link: the link goes to trash, %s stays", file, target))
			}
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return "This will:\n   " + strings.Join(lines, "\n   ") + "\nProceed?"
}

// printPurgeUsage writes usage to w — os.Stderr on error paths, so stdout
// stays clean for pipes.
func printPurgeUsage(w io.Writer) {
	fmt.Fprintln(w, "Purge Usage:")
	fmt.Fprintln(w, "To purge a specific profile, use:    `drako purge --target <name>`")
	fmt.Fprintln(w, "To purge Core config, use:           `drako purge --config`")
	fmt.Fprintln(w, "To select interactively, use:        `drako purge --interactive`")
	fmt.Fprintln(w, "Destroy ~/.config/drako directory:   `drako purge --destroyeverything`")
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
	scanDir := func(loc equip.Location, location string) {
		entries, err := lair.List(configDir, loc)
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
	scanDir(equip.Equipped, "Equipped")

	// 2. Scan Inventory
	scanDir(equip.Inventory, "Inventory")

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
			// Clamp before expanding: a huge range must not allocate.
			if start < 1 || end > len(validProfiles) {
				fmt.Fprintf(output, "Warning: %s reaches outside 1-%d\n", part, len(validProfiles))
				start, end = max(start, 1), min(end, len(validProfiles))
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
		if target := lair.LinkTarget(configDir, profile.RelativePath); target != "" {
			fmt.Fprintf(output, "%s is a link: the link goes to trash, %s stays\n", profile.RelativePath, target)
		}
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
