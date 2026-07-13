package cli

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lucky7xz/drako/internal/config" // drako.chronyx.xyz
	"github.com/lucky7xz/drako/internal/paths"
)

// HandleCLI checks if the program was invoked with CLI arguments (not TUI
// mode). handled=false means proceed to the TUI; otherwise code is the
// process exit code, which app.Run carries back to the sole os.Exit in main.
func HandleCLI(args []string) (handled bool, code int) {
	if len(args) <= 1 {
		return false, 0
	}

	command := args[1]

	switch command {
	case "ls", "list":
		return true, HandleLsCommand(args)
	case "summon", "--summon":
		return true, HandleSummonCommand(args)
	case "purge", "--purge":
		return true, HandlePurgeCommand(args)
	case "spec", "--spec":
		return true, HandleSpecCommand(args)
	case "stash", "--stash":
		return true, HandleStashCommand(args)
	case "strip", "--strip":
		return true, HandleStripCommand(args)
	case "restore-bootstrap", "--restore-bootstrap",
		"restore-core", "--restore-core": // legacy alias
		return true, HandleRestoreBootstrapCommand()
	case "explain", "--explain":
		return true, HandleExplainCommand(args)
	case "check", "--check":
		return true, HandleCheckCommand(args)
	case "open", "--open":
		return true, HandleOpenCLI(args)
	case "version", "--version", "-v":
		fmt.Printf("%s %s\n", config.AppName, config.Version())
		return true, 0
	case "help", "--help", "-h":
		PrintUsage()
		return true, 0
	default:
		// An argument that isn't a known command is an error.
		fmt.Printf("Unknown command or argument: '%s'\n\n", command)
		PrintUsage()
		return true, 1
	}
}

func PrintUsage() {
	fmt.Printf("%s %s\n", config.AppName, config.Version())
	fmt.Printf("Usage: drako <command> [arguments]\n\n")
	table(os.Stdout, []string{"Command", "Description"}, [][]string{
		{"ls", "List equipped decks and their cell addresses"},
		{"explain [profile:]<addr>", "Show one cell's command, description, flags"},
		{"check [path ...]", "Validate profile files for authoring errors"},
		{"summon <url>", "Summon profile(s) from a URL"},
		{"spec list", "Show available specs"},
		{"spec <name>", "Equip a spec's profiles; stash the rest"},
		{"stash <name>", "Stash a spec's profiles to the inventory"},
		{"strip", "Move all equipped profiles to the inventory"},
		{"restore-bootstrap", "Restore any missing bootstrap files"},
		{"purge <name>", "Remove profiles or config (try 'purge -i')"},
		{"version", "Show version information"},
		{"help", "Show this help message"},
		{"open <path/url>", "Open a file, dir, or URL with the OS default"},
	})
}

// HandleSummonCommand processes the 'drako summon <url>' command.
// Returns the process exit code.
func HandleSummonCommand(args []string) int {
	if len(args) < 3 {
		PrintSummonUsage()
		return 1
	}

	configDir, err := paths.ConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get config dir: %v\n", err)
		return 1
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "could not create config dir: %v\n", err)
		return 1
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
		return 1
	}
	log.Printf("Attempting to summon profile from: %s", sourceURL)

	summoner := NewSummoner(configDir)
	summoner.SHA256 = sha
	summoner.Rev = rev
	if err := summoner.Summon(sourceURL); err != nil {
		log.Printf("Summon failed: %v", err)
		fmt.Fprintf(os.Stderr, "Summon failed: %v\n", err)
		return 1
	}

	inventoryDir := paths.InventoryDir(configDir)
	fmt.Printf("\n✓ Profile summoned successfully to %s\n", inventoryDir)
	return 0
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

// HandleRestoreBootstrapCommand restores every embedded bootstrap file that is
// missing from the config dir — the core deck, starter decks like ssh-utils,
// themes, and specs — without ever overwriting an existing file. It is the way
// back after files have been stashed or deleted; the rescue grid offers it as a
// cell. Returns the process exit code.
func HandleRestoreBootstrapCommand() int {
	quietLogs() // keep the internal bootstrap log lines out of the user's terminal
	configDir, err := paths.ConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get config dir: %v\n", err)
		return 1
	}
	restored, err := config.RestoreBootstrap(configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "restore-bootstrap failed: %v\n", err)
		return 1
	}
	if len(restored) == 0 {
		fmt.Println("All bootstrap files present — nothing to restore.")
		return 0
	}
	fmt.Printf("✓ Restored %d file(s) to %s:\n", len(restored), configDir)
	for _, f := range restored {
		fmt.Printf("  %s\n", f)
	}
	return 0
}

// HandleOpenCLI processes the 'drako open <path>' command from the shell.
// Returns the process exit code.
func HandleOpenCLI(args []string) int {
	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: drako open <path>\n")
		return 1
	}

	// Reconstruct the argument or take the last one.
	path := args[2]

	if err := OpenPath(path); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening '%s': %v\n", path, err)
		return 1
	}

	return 0
}
