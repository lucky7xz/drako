package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
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

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		X:            3,
		Y:            3,
		Theme:        "dracula",
		NumbModifier: "alt",
		DefaultShell: "bash",
		Keys: InputConfig{
			Explain:      "e",
			Inventory:    "i",
			PathGridMode: "tab",
			Lock:         "r",
			ProfilePrev:  "o",
			ProfileNext:  "p",
		},
	}
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

func BuildGrid(config Config) [][]string {
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

func ExpandCommandTokens(s string, cfg Config) string {
	if strings.TrimSpace(s) == "" {
		return s
	}
	s = strings.ReplaceAll(s, "{dR4ko_path}", cfg.DR4koPath)
	return s
}

func FileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return true
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// OverlayIsEmpty returns true if no settings are provided in the overlay.
func OverlayIsEmpty(ov ProfileOverlay) bool {
	if ov.DR4koPath != nil {
		return false
	}
	if ov.X != nil {
		return false
	}
	if ov.Y != nil {
		return false
	}
	if ov.Theme != nil {
		return false
	}
	if ov.DefaultShell != nil {
		return false
	}
	if ov.NumbModifier != nil {
		return false
	}
	if ov.LockTimeoutMinutes != nil {
		return false
	}
	if ov.Commands != nil && len(*ov.Commands) > 0 {
		return false
	}
	return true
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
	profiles := []ProfileInfo{{Name: "Default", Path: ""}}

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return profiles, nil
	}

	var overlays []ProfileInfo
	var broken []ProfileParseError
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".profile.toml") {
			continue
		}
		path := filepath.Join(configDir, name)
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			broken = append(broken, ProfileParseError{
				Name: strings.TrimSuffix(name, ".profile.toml"),
				Path: path,
				Err:  rerr.Error(),
			})
			continue
		}
		if strings.TrimSpace(string(data)) == "" {
			broken = append(broken, ProfileParseError{
				Name: strings.TrimSuffix(name, ".profile.toml"),
				Path: path,
				Err:  "empty profile file (no content)",
			})
			continue
		}
		var overlay ProfileOverlay
		if _, err := toml.Decode(string(data), &overlay); err != nil {
			broken = append(broken, ProfileParseError{
				Name: strings.TrimSuffix(name, ".profile.toml"),
				Path: path,
				Err:  err.Error(),
			})
			continue
		}
		if OverlayIsEmpty(overlay) {
			broken = append(broken, ProfileParseError{
				Name: strings.TrimSuffix(name, ".profile.toml"),
				Path: path,
				Err:  "no settings found in profile",
			})
			continue
		}
		overlays = append(overlays, ProfileInfo{
			Name:    strings.TrimSuffix(name, ".profile.toml"),
			Path:    path,
			Overlay: overlay,
		})
	}

	sort.Slice(overlays, func(i, j int) bool {
		return overlays[i].Name < overlays[j].Name
	})

	profiles = append(profiles, overlays...)
	return profiles, broken
}

func DiscoverProfiles(configDir string) []ProfileInfo {
	profiles, _ := DiscoverProfilesWithErrors(configDir)
	return profiles
}

func ApplyProfileOverlay(base Config, overlay ProfileOverlay) Config {
	cfg := base

	if overlay.DR4koPath != nil {
		cfg.DR4koPath = *overlay.DR4koPath
	}

	if overlay.X != nil {
		cfg.X = *overlay.X
	}

	if overlay.Y != nil {
		cfg.Y = *overlay.Y
	}

	if overlay.Theme != nil {
		cfg.Theme = *overlay.Theme
	}

	if overlay.HeaderArt != nil {
		cfg.HeaderArt = overlay.HeaderArt
	}

	if overlay.DefaultShell != nil {
		cfg.DefaultShell = *overlay.DefaultShell
	}

	if overlay.NumbModifier != nil {
		cfg.NumbModifier = *overlay.NumbModifier
	}

	if overlay.LockTimeoutMinutes != nil {
		cfg.LockTimeoutMinutes = overlay.LockTimeoutMinutes
	}

	if overlay.Commands != nil {
		cfg.Commands = CopyCommands(*overlay.Commands)
	}

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

func readPivotProfile(configDir string) (pivotFile, error) {
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
	pf, _ := readPivotProfile(configDir)
	pf.Locked = strings.TrimSpace(name)
	return writePivotFile(configDir, pf)
}

func WritePivotEquippedOrder(configDir string, order []string) error {
	pf, _ := readPivotProfile(configDir)
	pf.EquippedOrder = order
	return writePivotFile(configDir, pf)
}

func DeletePivotProfile(configDir string) error {
	// Preserve equipped_order; only clear the lock
	pf, _ := readPivotProfile(configDir)
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

	pf, err := readPivotProfile(configDir)
	if err != nil {
		log.Printf("warning: could not read pivot profile: %v", err)
		pf = pivotFile{}
	}
	pivotRequested := false
	requestedPivot := strings.TrimSpace(pf.Locked)

	var base Config

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			fatalf("could not create config directory: %v", err)
		}

		base = DefaultConfig()
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
		if _, err := toml.Decode(configString, &base); err != nil {
			fatalf("could not decode config file: %v", err)
		}
		// Apply defaults for any missing fields
		defaults := DefaultConfig()
		if strings.TrimSpace(base.NumbModifier) == "" {
			base.NumbModifier = defaults.NumbModifier
		}
		if strings.TrimSpace(base.DefaultShell) == "" {
			base.DefaultShell = defaults.DefaultShell
		}
		if strings.TrimSpace(base.Theme) == "" {
			base.Theme = defaults.Theme
		}
		// Apply key defaults if missing
		if strings.TrimSpace(base.Keys.Explain) == "" {
			base.Keys.Explain = defaults.Keys.Explain
		}
		if strings.TrimSpace(base.Keys.Inventory) == "" {
			base.Keys.Inventory = defaults.Keys.Inventory
		}
		if strings.TrimSpace(base.Keys.PathGridMode) == "" {
			base.Keys.PathGridMode = defaults.Keys.PathGridMode
		}
		if strings.TrimSpace(base.Keys.Lock) == "" {
			base.Keys.Lock = defaults.Keys.Lock
		}
		if strings.TrimSpace(base.Keys.ProfilePrev) == "" {
			base.Keys.ProfilePrev = defaults.Keys.ProfilePrev
		}
		if strings.TrimSpace(base.Keys.ProfileNext) == "" {
			base.Keys.ProfileNext = defaults.Keys.ProfileNext
		}
		log.Printf("Loaded config: X=%d, Y=%d, Commands=%d", base.X, base.Y, len(base.Commands))
	}

	ClampConfig(&base)
	// Compute the navigation sets based on flags
	base.Keys.InitControls()

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
	selected := profiles[activeIndex]
	if useFactoryDefaults || (len(broken) > 0 && NormalizeProfileName(selected.Name) == "default") {
		// Either an explicitly requested profile was missing/broken, or there are broken overlays present
		// and we are using Default. Fall back to factory defaults (3x3).
		effective = DefaultConfig()
		// Ensure controls are initialized for factory default
		effective.Keys.InitControls()
	} else if NormalizeProfileName(selected.Name) != "default" {
		effective = ApplyProfileOverlay(base, selected.Overlay)
		// Controls might have been updated if overlay affects base config (which it does)
		// Re-init controls to be safe, although keys aren't currently overridable per profile
		effective.Keys.InitControls()
		log.Printf("Applied profile overlay: %s", selected.Name)
	}

	ClampConfig(&effective)
	effective.Commands = CopyCommands(effective.Commands)

	return ConfigBundle{
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
