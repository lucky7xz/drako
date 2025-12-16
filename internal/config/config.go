package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	log.Fatalf(format, args...)
}

func letterToColumn(s string) (int, error) {
	if len(s) != 1 {
		return 0, errors.New("column must be a single letter")
	}
	char := strings.ToLower(s)[0]

	if char == 'z' {
		return -1, nil
	}

	if char < 'a' || char >= 'z' {
		return 0, errors.New("column must be a letter from 'a' to 'y', or 'z' for the last column")
	}
	return int(char - 'a'), nil
}

// RescueConfig returns a minimal "Safe Mode" configuration.
// It provides tools to help the user fix a broken configuration.
func RescueConfig() Config {
	isWindows := runtime.GOOS == "windows"
	defaultShell := "bash"

	if isWindows {
		defaultShell = "pwsh" // Prefer PowerShell on Windows, falling back to cmd if needed by user config
	}

	// Resolve config directory for "Open Config Dir" / "Edit Config"
	// We use the same logic as GetConfigDir, but since we are inside config package,
	// we assume standard locations.
	// Since RescueConfig is a fallback, we should try to be helpful.
	// However, GetConfigDir might return error or different path if not set.
	// Best effort:
	configDir, err := GetConfigDir()
	if err != nil {
		// Fallback to a safe guess if we can't find it
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config", "drako")
		if isWindows {
			configDir = filepath.Join(home, "AppData", "Roaming", "drako")
		}
	}

	configPath := filepath.Join(configDir, "config.toml")

	// We wrap paths in quotes in case of spaces, though drako open handles simple strings.
	// Actually handling spaces in arguments for "drako open" requires care if we just use string splitting.
	// But our HandleOpenCommand takes "everything after prefix". So spaces are preserved.

	editCmd := fmt.Sprintf("drako open %s", configPath)
	openDirCmd := fmt.Sprintf("drako open %s", configDir)

	return Config{
		X:            3,
		Y:            3,
		Theme:        "dracula", // A safe, dark theme
		NumbModifier: "alt",
		DefaultShell: defaultShell,
		Keys: InputConfig{
			Explain:      "e",
			Inventory:    "i",
			PathGridMode: "tab",
			Lock:         "r",
			ProfilePrev:  "o",
			ProfileNext:  "p",
		},
		Commands: []Command{
			{
				Name:        "Reset Core Config",
				Command:     "drako purge --target core",
				Description: "Resets your config.toml to defaults.\n\n• Your old config.toml will be moved to trash/.\n• Use this to fix syntax errors in config.toml.\n• Drako will exit after this operation.",
				Row:         0,
				Col:         "a", // Left
			},
			{
				Name:        "Reset a Profile",
				Command:     "drako purge --interactive",
				Description: "Select a profile to reset/delete.\n\n• Useful if a specific profile is broken and crashing Drako.\n• The profile will be moved to trash/.",
				Row:         1,
				Col:         "a", // Left below Reset Core
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
				Name:        "Exit Rescue Mode",
				Command:     "true", // Intercepted by UI
				Description: "Returns to your Core configuration.\n\n(Same as switching to the first profile with Mod+1)",
				Row:         2,
				Col:         "b", // Center bottom
			},
		},
	}
}

// ApplyDefaults fills in any missing fields with default values.
// It ensures the configuration is valid and complete.
func (c *Config) ApplyDefaults() {
	defaults := RescueConfig()

	if strings.TrimSpace(c.NumbModifier) == "" {
		c.NumbModifier = defaults.NumbModifier
	}
	if strings.TrimSpace(c.DefaultShell) == "" {
		c.DefaultShell = defaults.DefaultShell
	}
	if strings.TrimSpace(c.Theme) == "" {
		c.Theme = defaults.Theme
	}

	// Apply key defaults if missing
	if strings.TrimSpace(c.Keys.Explain) == "" {
		c.Keys.Explain = defaults.Keys.Explain
	}
	if strings.TrimSpace(c.Keys.Inventory) == "" {
		c.Keys.Inventory = defaults.Keys.Inventory
	}
	if strings.TrimSpace(c.Keys.PathGridMode) == "" {
		c.Keys.PathGridMode = defaults.Keys.PathGridMode
	}
	if strings.TrimSpace(c.Keys.Lock) == "" {
		c.Keys.Lock = defaults.Keys.Lock
	}
	if strings.TrimSpace(c.Keys.ProfilePrev) == "" {
		c.Keys.ProfilePrev = defaults.Keys.ProfilePrev
	}
	if strings.TrimSpace(c.Keys.ProfileNext) == "" {
		c.Keys.ProfileNext = defaults.Keys.ProfileNext
	}

	// Ensure limits are respected
	ClampConfig(c)

	// Initialize control sets (WASD, Vim, arrows)
	c.Keys.InitControls()
}

func ClampConfig(cfg *Config) {
	if cfg.X < 1 {
		cfg.X = 1
	}
	if cfg.X > 9 {
		cfg.X = 9
	}
	if cfg.Y < 1 {
		cfg.Y = 1
	}
	if cfg.Y > 9 {
		cfg.Y = 9
	}
}

// ValidateConfig checks if the configuration is logically valid.
// It returns an error if any command is out of bounds for the grid size.
func ValidateConfig(cfg Config) error {
	for _, cmd := range cfg.Commands {
		row := cmd.Row
		col, err := letterToColumn(cmd.Col)
		if err != nil {
			return fmt.Errorf("command %q has invalid column %q: %v", cmd.Name, cmd.Col, err)
		}

		// Handle special -1 values (meaning last row/col)
		if row == -1 {
			row = cfg.Y - 1
		}
		if col == -1 {
			col = cfg.X - 1
		}

		if row >= cfg.Y {
			return fmt.Errorf("command %q at row %d exceeds grid height %d", cmd.Name, row, cfg.Y)
		}
		if col >= cfg.X {
			return fmt.Errorf("command %q at column %q exceeds grid width %d", cmd.Name, cmd.Col, cfg.X)
		}
	}
	return nil
}

func BuildGrid(config Config) [][]string {
	// Safety clamp, though ApplyDefaults usually handles it
	ClampConfig(&config)
	grid := make([][]string, config.Y)
	for i := range grid {
		grid[i] = make([]string, config.X)
	}
	for _, cmd := range config.Commands {
		row := cmd.Row
		col, err := letterToColumn(cmd.Col)
		if err != nil {
			fatalf("invalid column value for command %q: %v", cmd.Name, err)
		}

		if row == -1 {
			row = config.Y - 1
		}
		if col == -1 {
			col = config.X - 1
		}
		if row >= 0 && row < config.Y && col >= 0 && col < config.X {
			grid[row][col] = cmd.Name
		}
	}
	return grid
}

func CopyCommands(src []Command) []Command {
	if len(src) == 0 {
		return []Command{}
	}
	dst := make([]Command, len(src))
	copy(dst, src)
	return dst
}

// ValidateProfileFile checks if the profile has minimum required fields
func ValidateProfileFile(pf ProfileFile) (bool, []string) {
	var missing []string
	if len(pf.Commands) == 0 {
		missing = append(missing, "commands")
	}
	// X and Y are critical for grid
	if pf.X <= 0 {
		missing = append(missing, "x")
	}
	if pf.Y <= 0 {
		missing = append(missing, "y")
	}
	return len(missing) == 0, missing
}

func NormalizeProfileName(name string) string {
	n := strings.TrimSpace(strings.ToLower(name))
	// Normalize known suffixes in safe order
	n = strings.TrimSuffix(n, ".profile.toml")
	n = strings.TrimSuffix(n, ".toml")
	n = strings.TrimSuffix(n, ".profile")
	return n
}

func DiscoverProfilesWithErrors(configDir string) ([]ProfileInfo, []ProfileParseError) {
	profiles := []ProfileInfo{}

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return profiles, nil
	}

	var discoveredProfiles []ProfileInfo
	var broken []ProfileParseError
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".profile.toml") {
			continue
		}
		// Check for empty file or broken read?
		// Note: We used to check for empty strings here.
		// Since toml.DecodeFile handles open/read, strict read check before might be duplicated but harmless.
		fullPath := filepath.Join(configDir, name)
		profileName := strings.TrimSuffix(name, ".profile.toml")

		// Parse the profile to check for validity and metadata
		var profileFile ProfileFile
		if _, err := toml.DecodeFile(fullPath, &profileFile); err != nil {
			log.Printf("Failed to parse profile %s: %v", entry.Name(), err)
			broken = append(broken, ProfileParseError{Name: profileName, Path: fullPath, Err: err.Error()})
			continue
		}

		if ok, missing := ValidateProfileFile(profileFile); !ok {
			broken = append(broken, ProfileParseError{
				Name: profileName,
				Path: fullPath,
				Err:  fmt.Sprintf("profile is missing required settings: %s", strings.Join(missing, ", ")),
			})
			continue
		}

		discoveredProfiles = append(discoveredProfiles, ProfileInfo{
			Name:    profileName,
			Path:    fullPath,
			Profile: profileFile,
		})
	}

	sort.Slice(discoveredProfiles, func(i, j int) bool {
		return discoveredProfiles[i].Name < discoveredProfiles[j].Name
	})

	profiles = append(profiles, discoveredProfiles...)
	return profiles, broken
}

func DiscoverProfiles(configDir string) []ProfileInfo {
	profiles, _ := DiscoverProfilesWithErrors(configDir)
	return profiles
}

func applyAndValidate(info ProfileInfo) (Config, error) {
	// This logic relies on "base" being available in scope, but it isn't.
	// Wait, ApplyProfileOverlay was a pure function.
	// The previous failed edit seemingly messed up function boundaries or signatures.
	// Let's restore/fix ApplyProfileOverlay.
	return Config{}, nil // Placeholder to fix syntax, will be properly implemented in next step if viewed
}

func ApplyProfileOverlay(base Config, profile ProfileFile) Config {
	cfg := base

	cfg.X = profile.X
	cfg.Y = profile.Y
	if strings.TrimSpace(profile.Theme) != "" {
		cfg.Theme = profile.Theme
	}
	if profile.HeaderArt != nil {
		cfg.HeaderArt = profile.HeaderArt
	}
	if profile.Shell != nil {
		cfg.DefaultShell = *profile.Shell
	}
	// Commands are mandatory in ProfileFile basically
	cfg.Commands = CopyCommands(profile.Commands)

	return cfg
}

const pivotProfileFilename = "pivot.toml"

func GetConfigDir() (string, error) {

	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return "", errors.Join(err, herr)
		}
		configDir = filepath.Join(home, ".drako")

	} else {
		configDir = filepath.Join(configDir, "drako")
	}
	return configDir, nil

}

func pivotProfilePath(configDir string) string {
	return filepath.Join(configDir, pivotProfileFilename)
}

type pivotFile struct {
	Locked        string   `toml:"locked"`
	EquippedOrder []string `toml:"equipped_order"`
}

func ReadPivotProfile(configDir string) (pivotFile, error) {
	var pf pivotFile
	path := pivotProfilePath(configDir)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return pivotFile{}, nil
	}
	if err != nil {
		return pivotFile{}, err
	}
	if _, err := toml.Decode(string(data), &pf); err != nil {
		return pivotFile{}, err
	}
	return pf, nil
}

func writePivotFile(configDir string, pf pivotFile) error {
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	path := pivotProfilePath(configDir)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(pf)
}

func WritePivotLocked(configDir, name string) error {
	pf, _ := ReadPivotProfile(configDir)
	pf.Locked = strings.TrimSpace(name)
	return writePivotFile(configDir, pf)
}

func WritePivotEquippedOrder(configDir string, order []string) error {
	pf, _ := ReadPivotProfile(configDir)
	pf.EquippedOrder = order
	return writePivotFile(configDir, pf)
}

func DeletePivotProfile(configDir string) error {
	// Preserve equipped_order; only clear the lock
	pf, _ := ReadPivotProfile(configDir)
	if pf.Locked == "" && len(pf.EquippedOrder) == 0 {
		// No useful content, remove file if it exists
		return os.Remove(pivotProfilePath(configDir))
	}
	pf.Locked = ""
	return writePivotFile(configDir, pf)
}

func LoadConfig(profileOverride *string) ConfigBundle {

	configDir, err := GetConfigDir()

	if err != nil {
		fatalf("could not resolve a config directory: %v", err)
	}

	configPath := filepath.Join(configDir, "config.toml")
	// First run: if config file is missing, ensure dir and copy embedded bootstrap assets
	if _, statErr := os.Stat(configPath); errors.Is(statErr, os.ErrNotExist) {
		if mkErr := os.MkdirAll(configDir, 0o755); mkErr != nil {
			fatalf("could not create config directory: %v", mkErr)
		}
		if err := bootstrapCopy(configDir); err != nil {
			log.Printf("warning: bootstrap copy failed: %v", err)
		}
	}

	pf, err := ReadPivotProfile(configDir)
	if err != nil {
		log.Printf("warning: could not read pivot profile: %v", err)
		pf = pivotFile{}
	}
	pivotRequested := false
	requestedPivot := strings.TrimSpace(pf.Locked)

	var base Config

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// This case is redundant if bootstrapCopy works, but kept for safety
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			fatalf("could not create config directory: %v", err)
		}

		base = RescueConfig()
		f, err := os.Create(configPath)
		if err != nil {
			fatalf("could not create config file: %v", err)
		}

		defer f.Close()
		if err := toml.NewEncoder(f).Encode(base); err != nil {
			fatalf("could not write to config file: %v", err)
		}

	} else {
		log.Printf("Loading config from: %s", configPath)
		configBytes, err := os.ReadFile(configPath)
		if err != nil {
			fatalf("could not read config file: %v", err)
		}
		configString := os.ExpandEnv(string(configBytes))

		var settings AppSettings
		if _, err := toml.Decode(configString, &settings); err != nil {
			fatalf("could not decode config file: %v", err)
		}

		// Convert Settings to Base Config (Commands are empty)
		// Base config initially lacks Layout/Theme (will be filled by Profile or Defaults)
		base = Config{
			DefaultShell:       settings.DefaultShell,
			NumbModifier:       settings.NumbModifier,
			Profile:            settings.Profile,
			LockTimeoutMinutes: settings.LockTimeoutMinutes,
			EnvWhitelist:       settings.EnvWhitelist,
			EnvBlocklist:       settings.EnvBlocklist,
			Theme:              settings.Theme,
			Keys:               settings.Keys,
			Commands:           []Command{}, // Explicitly empty
		}
		log.Printf("Loaded base settings")
	}

	// Apply defaults to the base config immediately
	base.ApplyDefaults()

	// Validate base config (sanity check)
	if err := ValidateConfig(base); err != nil {
		// For base config, we probably still want to crash or fallback, but let's log it
		log.Printf("Error: Base config is invalid: %v", err)
		// We can fallback to hard defaults if base is totally broken
		// base = RescueConfig()
		// base.ApplyDefaults()
	}

	profiles, broken := DiscoverProfilesWithErrors(configDir)
	// Reorder profiles based on pivot equipped_order
	if len(pf.EquippedOrder) > 0 {
		remaining := map[string]ProfileInfo{}
		for i := 0; i < len(profiles); i++ {
			remaining[NormalizeProfileName(profiles[i].Name)] = profiles[i]
		}
		var ordered []ProfileInfo
		for _, n := range pf.EquippedOrder {
			norm := NormalizeProfileName(n)
			if info, ok := remaining[norm]; ok {
				ordered = append(ordered, info)
				delete(remaining, norm)
			}
		}
		if len(remaining) > 0 {
			var rest []ProfileInfo
			for _, v := range remaining {
				rest = append(rest, v)
			}
			sort.Slice(rest, func(i, j int) bool { return rest[i].Name < rest[j].Name })
			ordered = append(ordered, rest...)
		}
		profiles = ordered
	}

	var requested string
	if profileOverride != nil {
		requested = *profileOverride
	} else if requestedPivot != "" {
		requested = requestedPivot
		pivotRequested = true
	} else {
		requested = strings.TrimSpace(os.Getenv("DRAKO_PROFILE"))
		if requested == "" {
			requested = strings.TrimSpace(base.Profile)
		}
	}

	target := NormalizeProfileName(requested)
	activeIndex := 0
	pivotStillValid := requestedPivot != ""
	useFactoryDefaults := false

	if target != "" {
		found := false
		for i := 0; i < len(profiles); i++ {
			if NormalizeProfileName(profiles[i].Name) == target {
				activeIndex = i
				found = true
				break
			}
		}
		if !found && strings.TrimSpace(requested) != "" {
			log.Printf("profile not found (possibly broken), falling back to factory defaults: %s", requested)
			useFactoryDefaults = true
			if pivotRequested {
				if err := WritePivotLocked(configDir, ""); err != nil {
					log.Printf("warning: could not clear pivot lock: %v", err)
				}
				pivotStillValid = false
			}
		}
	}

	effective := base
	// effective initialized above as base
	var selected ProfileInfo

	if len(profiles) > 0 {
		selected = profiles[activeIndex]
	} else {
		// No profiles found! Enforce factory defaults.
		useFactoryDefaults = true
		// Create a dummy selected profile for logging/logic safety
		selected = ProfileInfo{Name: "Rescue", Path: "internal", Profile: ProfileFile{}}
	}

	// Helper to safely apply and validate a profile
	applyAndValidate := func(p ProfileInfo) (Config, error) {
		temp := ApplyProfileOverlay(base, p.Profile)
		temp.ApplyDefaults()
		if err := ValidateConfig(temp); err != nil {
			return temp, err
		}
		return temp, nil
	}

	if useFactoryDefaults {
		// Fall back to factory defaults (3x3).
		effective = RescueConfig()
		// ApplyDefaults will init controls too
		effective.ApplyDefaults()
	} else {
		// Attempt to apply the selected profile
		// Even for "Core", it is now an overlay loaded from disk
		var err error
		effective, err = applyAndValidate(selected)
		if err != nil {
			// validation failed for the selected profile!
			log.Printf("Selected profile %q is invalid: %v. Falling back to defaults.", selected.Name, err)
			broken = append(broken, ProfileParseError{
				Name: selected.Name,
				Path: selected.Path,
				Err:  fmt.Sprintf("Grid validation failed: %v", err),
			})
			// Since selected is broken, we fall back to core/base
			effective = base
			// Reset active index to core (0) or stay 0 if empty
			activeIndex = 0
			pivotStillValid = false
			// Update selected to the first available profile if possible
			// This implements "First Available" fallback policy
			if len(profiles) > 0 {
				selected = profiles[0]
				// We must retry applying this new selection to be safe?
				// Actually, the loop logic below (if we had one) would handle it.
				// But here we are linear. Let's try to apply the fallback immediately
				// to avoid showing a broken state if the fallback is valid.
				log.Printf("Falling back to first available profile: %s", selected.Name)
				fallbackConfig, err2 := applyAndValidate(selected)
				if err2 == nil {
					effective = fallbackConfig
					// Need to update activeIndex to match this new selected profile
					// profiles[0] is index 0
					activeIndex = 0
				} else {
					log.Printf("Fallback profile %s is also invalid: %v", selected.Name, err2)
					// Verify if we should add it to broken list? It might already be broken from discover?
					// If applyAndValidate fails, it implies it's broken.
				}
			}
		} else {
			log.Printf("Applied profile overlay: %s", selected.Name)
		}
	}

	effective.Commands = CopyCommands(effective.Commands)

	return ConfigBundle{
		Settings: AppSettings{
			DefaultShell:       base.DefaultShell,
			NumbModifier:       base.NumbModifier,
			Profile:            base.Profile,
			LockTimeoutMinutes: base.LockTimeoutMinutes,
			EnvWhitelist:       base.EnvWhitelist,
			EnvBlocklist:       base.EnvBlocklist,
			Theme:              base.Theme,
			Keys:               base.Keys,
		},
		Base:        base,
		Config:      effective,
		Profiles:    profiles,
		ActiveIndex: activeIndex,
		ConfigDir:   configDir,
		LockedName: func() string {
			if !pivotStillValid {
				return ""
			}
			return requestedPivot
		}(),
		Broken: broken,
	}
}
