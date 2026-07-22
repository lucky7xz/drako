package config

import (
	"fmt"
	"runtime"

	"github.com/lucky7xz/drako/internal/paths"
)

// RescueConfig returns a minimal "Safe Mode" configuration.
// It provides tools to help the user fix a broken configuration.
func RescueConfig() Config {
	isWindows := runtime.GOOS == "windows"
	defaultShell := "bash"

	if isWindows {
		defaultShell = "powershell" // 5.1 ships with Windows; users with PowerShell 7 can set "pwsh"
	}

	// Used only to build the "Edit Config" / "Open Config Dir" helper
	// commands; resolution has already succeeded by the time rescue mode
	// can run.
	configDir, _ := paths.ConfigDir()
	configPath := paths.ConfigFile(configDir)

	editCmd := fmt.Sprintf("drako open %s", configPath)
	openDirCmd := fmt.Sprintf("drako open %s", configDir)

	return Config{
		X:                  3,
		Y:                  3,
		Theme:              "dracula", // A safe, dark theme
		NumbModifier:       "alt",
		DefaultShell:       defaultShell,
		LockTimeoutMinutes: func() *int { i := 5; return &i }(),
		AutoLockEnabled:    func() *bool { b := true; return &b }(),
		Keys: InputConfig{
			Explain:      "e",
			Inventory:    "i",
			PathGridMode: "tab",
			Lock:         "r",
			ProfilePrev:  "o",
			ProfileNext:  "p",
			EditFile:     "e",
			Leader:       "m",
		},
		Commands: []Command{
			{
				Name:        "Exit Rescue Mode",
				Command:     "true", // Intercepted by UI
				Description: "Reloads your profiles and returns to whatever is equipped.\n\n• If no profiles are equipped, you stay in Rescue mode.",
				Row:         0,
				Col:         "a", // Left top — the cell the cursor lands on when rescue opens
			},
			{
				Name:        "Remove Core Profile",
				Command:     "drako purge --target core",
				Description: "Removes your core.profile.toml.\n\n• Use this if the core profile layout is broken.",
				Row:         1,
				Col:         "a", // Left below Exit
			},
			{
				Name:        "Remove Another Profile",
				Command:     "drako purge --interactive",
				Description: "Select a profile to remove.\n\n• Useful if a specific profile is broken and crashing Drako.\n• The profile will be moved to trash/.",
				Row:         2,
				Col:         "a", // Left below Reset Core Profile
			},
			{
				Name:        "Edit Config",
				Command:     editCmd,
				Description: "Opens the main configuration file in your default editor.\n\n• Use this to fix syntax errors in config.toml.\n• If this file is broken, Drako falls back to this Rescue mode.\n\nTip: You can switch to a working profile right now with 'o' (prev) or 'p' (next).",
				Row:         0,
				Col:         "b", // Center
			},
			{
				Name:        "Documentation",
				Command:     "drako open https://github.com/lucky7xz/drako",
				Description: "Opens the Drako documentation in your browser.\n\n• Check the syntax reference.\n• Find examples of valid profiles.\n\nTip: You can switch to a working profile right now with 'o' (prev) or 'p' (next).",
				Row:         0,
				Col:         "c", // Right
			},
			{
				Name:        "Open Config Dir",
				Command:     openDirCmd,
				Description: "Opens the configuration directory.\n\n• Delete or fix broken profiles here.\n• Move unfinished profiles to a 'collection' subfolder to hide them.\n\nTip: You can switch to a working profile right now with 'o' (prev) or 'p' (next).",
				Row:         1,
				Col:         "b", // Center below Edit
			},
			{
				Name:        "Reload Config",
				Command:     "true", // No-op, but triggers an update loop because execution finishes
				Description: "Forces a reload of the configuration.\nDrako automatically reloads on file save, but you can use this to manually retry.\n\nTip: You can switch to a working profile right now with 'o' (prev) or 'p' (next).",
				Row:         1,
				Col:         "c", // Right below Docs
			},
			{
				Name:        "Reset Core (config & profile)",
				Command:     "drako purge --config",
				Description: "Resets config.toml to defaults.\n\n• Your old config.toml will be moved to trash/. Note that if the Core profile has been removed, this will reinitialize it too\n• Use this to fix syntax errors in config.toml.\n• Drako will exit after this operation.",
				Row:         2,
				Col:         "b", // Center bottom — the most destructive cell, farthest from the entry cursor
			},
			{
				Name:        "Restore Bootstrap Files",
				Command:     "drako restore-bootstrap",
				Description: "Restores any bootstrap files that are missing — the core deck, starter decks like ssh-utils, themes, and specs.\n\n• Use this if you stashed or deleted files shipped with drako and want them back.\n• Existing files are never overwritten.",
				Row:         2,
				Col:         "c", // Right bottom
			},
		},
	}
}
