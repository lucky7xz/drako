package cli

import (
	"fmt"
	"log"
	"os"
	"strings"

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
	case "ls", "list":
		HandleLsCommand(args)
		return true
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
	case "restore-core", "--restore-core":
		HandleRestoreCoreCommand()
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
		{"ls", "List equipped profiles and their commands with cell addresses"},
		{"summon <url>", "Summon profile(s) from a URL"},
		{"spec list", "Shows available specs"},
		{"spec <name>", "Apply a spec: move related profiles files to inventory."},
		{"stash <name>", "Stash the related prifle files to inventory."},
		{"strip", "Move all equipped profiles to inventory."},
		{"restore-core", "Regenerate the default core profile for this platform"},
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

	sourceURL, sha, rev, err := parseSummonArgs(args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		PrintSummonUsage()
		os.Exit(1)
	}
	log.Printf("Attempting to summon profile from: %s", sourceURL)

	summoner := NewSummoner(configDir)
	summoner.SHA256 = sha
	summoner.Rev = rev
	if err := summoner.Summon(sourceURL); err != nil {
		log.Printf("Summon failed: %v", err)
		fmt.Fprintf(os.Stderr, "Summon failed: %v\n", err)
		os.Exit(1)
	}

	inventoryDir := paths.InventoryDir(configDir)
	fmt.Printf("\n✓ Profile summoned successfully to %s\n", inventoryDir)
	os.Exit(0)
}

// parseSummonArgs extracts the source URL and the optional verification
// pins from everything after "drako summon". Flags may appear before or
// after the URL, in --flag value or --flag=value form.
func parseSummonArgs(args []string) (sourceURL, sha, rev string, err error) {
	flagValue := func(i *int, name string) (string, error) {
		a := args[*i]
		if a == name {
			*i++
			if *i >= len(args) {
				return "", fmt.Errorf("%s requires a value", name)
			}
			return args[*i], nil
		}
		return strings.TrimPrefix(a, name+"="), nil
	}

	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--sha256" || strings.HasPrefix(a, "--sha256="):
			if sha, err = flagValue(&i, "--sha256"); err != nil {
				return "", "", "", err
			}
		case a == "--rev" || strings.HasPrefix(a, "--rev="):
			if rev, err = flagValue(&i, "--rev"); err != nil {
				return "", "", "", err
			}
		case strings.HasPrefix(a, "-"):
			return "", "", "", fmt.Errorf("unknown flag: %s", a)
		case sourceURL != "":
			return "", "", "", fmt.Errorf("more than one source URL given (%q and %q)", sourceURL, a)
		default:
			sourceURL = a
		}
	}

	if sourceURL == "" {
		return "", "", "", fmt.Errorf("no source URL given")
	}
	// Only full-length pins verify anything: a truncated hash weakens the
	// check, and branch/tag names are mutable — reject both loudly.
	if sha != "" && !isHexString(sha, 64) {
		return "", "", "", fmt.Errorf("--sha256 must be the full 64-character hex digest (got %d characters)", len(sha))
	}
	if rev != "" && !isHexString(rev, 40) {
		return "", "", "", fmt.Errorf("--rev must be a full 40-character commit hash; branch and tag names are mutable and verify nothing")
	}
	return sourceURL, sha, rev, nil
}

// PrintSummonUsage prints the usage information for the summon command
func PrintSummonUsage() {
	fmt.Fprintf(os.Stderr, "Usage: drako summon <url> [--sha256 <hex64> | --rev <commit40>]\n")
	fmt.Fprintf(os.Stderr, "\nSummoned profiles are saved to ~/.config/drako/inventory/\n")
	fmt.Fprintf(os.Stderr, "Pins verify that what arrived is what the author published;\n")
	fmt.Fprintf(os.Stderr, "without one, drako warns and proceeds unverified.\n")
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  # Summon a single profile file (pin with its sha256):\n")
	fmt.Fprintf(os.Stderr, "  drako summon https://example.com/deck.profile.toml --sha256 9f2a...\n")
	fmt.Fprintf(os.Stderr, "\n  # Summon from a git repository (pin with a full commit hash):\n")
	fmt.Fprintf(os.Stderr, "  drako summon git@github.com:user/repo.git --rev 3f81c2d...\n")
	fmt.Fprintf(os.Stderr, "  drako summon https://github.com/user/repo.git\n")
}

// HandleRestoreCoreCommand regenerates core.profile.toml from the embedded
// template. This is the way back after the core deck has been stashed or
// deleted — the rescue grid offers it as a cell.
func HandleRestoreCoreCommand() {
	configDir, err := paths.ConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get config dir: %v\n", err)
		os.Exit(1)
	}
	created, err := config.RestoreCoreProfile(configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "restore-core failed: %v\n", err)
		os.Exit(1)
	}
	if created {
		fmt.Printf("✓ Core profile restored to %s\n", configDir)
	} else {
		fmt.Println("Core profile already present — nothing to do.")
	}
	os.Exit(0)
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
