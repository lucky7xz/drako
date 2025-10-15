package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

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

func clampConfig(cfg *Config) {
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

func buildGrid(config Config) [][]string {
	clampConfig(&config)
	grid := make([][]string, config.Y)
	for i := range grid {
		grid[i] = make([]string, config.X)
	}
	for _, cmd := range config.Commands {
		row := cmd.Row
		col, err := letterToColumn(cmd.Col)
		if err != nil {
			log.Fatalf("invalid column value for command %q: %v", cmd.Name, err)
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

func copyCommands(src []Command) []Command {
	if len(src) == 0 {
		return []Command{}
	}
	dst := make([]Command, len(src))
	copy(dst, src)
	return dst
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return true
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func normalizeProfileName(name string) string {
	n := strings.TrimSpace(strings.ToLower(name))
	n = strings.TrimSuffix(n, ".toml")
	n = strings.TrimSuffix(n, ".profile")
	n = strings.TrimSuffix(n, ".profile")
	return n
}

func discoverProfiles(configDir string) []ProfileInfo {
	profiles := []ProfileInfo{{Name: "Default", Path: ""}}

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return profiles
	}

	var overlays []ProfileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".profile.toml") {
			continue
		}
		path := filepath.Join(configDir, name)
		var overlay profileOverlay
		if _, err := toml.DecodeFile(path, &overlay); err != nil {
			log.Printf("warning: could not decode profile overlay %s: %v", path, err)
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
	return profiles
}

func applyProfileOverlay(base Config, overlay profileOverlay) Config {

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

	if overlay.NumbModifier != nil {
		cfg.NumbModifier = *overlay.NumbModifier
	}

	if overlay.Behavior != nil {

		if overlay.Behavior.ExitConfirmation != nil {
			cfg.Behavior.ExitConfirmation = *overlay.Behavior.ExitConfirmation
		}

		if overlay.Behavior.AutoSave != nil {
			cfg.Behavior.AutoSave = *overlay.Behavior.AutoSave
		}

	}

	if overlay.Commands != nil {
		cfg.Commands = copyCommands(*overlay.Commands)
	}

	return cfg

}



const pivotProfileFilename = "pivot.profile"



func getConfigDir() (string, error) {

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



func readPivotProfile(configDir string) (string, error) {

	data, err := os.ReadFile(pivotProfilePath(configDir))

	if errors.Is(err, os.ErrNotExist) {

		return "", nil

	}

	if err != nil {

		return "", err

	}

	return strings.TrimSpace(string(data)), nil

}



func writePivotProfile(configDir, name string) error {

	if err := os.MkdirAll(configDir, 0o755); err != nil {

		return err

	}

	return os.WriteFile(pivotProfilePath(configDir), []byte(name+"\n"), 0o644)

}



func deletePivotProfile(configDir string) error {

	err := os.Remove(pivotProfilePath(configDir))

	if errors.Is(err, os.ErrNotExist) {

		return nil

	}

	return err

}


func loadConfig(profileOverride *string) configBundle {

	configDir, err := getConfigDir()

	if err != nil {

		log.Fatalf("could not resolve a config directory: %v", err)

	}

	configPath := filepath.Join(configDir, "config.toml")
	// First run: if config file is missing, ensure dir and copy embedded bootstrap assets
	if _, statErr := os.Stat(configPath); errors.Is(statErr, os.ErrNotExist) {
		if mkErr := os.MkdirAll(configDir, 0o755); mkErr != nil {
			log.Fatalf("could not create config directory: %v", mkErr)
		}
		if err := bootstrapCopy(configDir); err != nil {
			log.Printf("warning: bootstrap copy failed: %v", err)
		}
	}



	pivotName, err := readPivotProfile(configDir)

	if err != nil {

		log.Printf("warning: could not read pivot profile: %v", err)

		pivotName = ""

	}

	pivotRequested := false

	requestedPivot := strings.TrimSpace(pivotName)



	var base Config

	if _, err := os.Stat(configPath); os.IsNotExist(err) {

		if err := os.MkdirAll(configDir, 0o755); err != nil {

			log.Fatalf("could not create config directory: %v", err)

		}

		base.X = 3

		base.Y = 3

		base.Theme = "dracula"

		base.Behavior = DracoBehaviorConfig{ExitConfirmation: false, AutoSave: true}



		f, err := os.Create(configPath)

		if err != nil {

			log.Fatalf("could not create config file: %v", err)

		}

		defer f.Close()

		if err := toml.NewEncoder(f).Encode(base); err != nil {

			log.Fatalf("could not write to config file: %v", err)

		}


	} else {
		log.Printf("Loading config from: %s", configPath)
		configBytes, err := os.ReadFile(configPath)
		if err != nil {
			log.Fatalf("could not read config file: %v", err)
		}
		configString := os.ExpandEnv(string(configBytes))
		if _, err := toml.Decode(configString, &base); err != nil {
			log.Fatalf("could not decode config file: %v", err)
		}
		if strings.TrimSpace(base.NumbModifier) == "" {
			base.NumbModifier = "alt"
		}
		log.Printf("Loaded config: X=%d, Y=%d, Commands=%d", base.X, base.Y, len(base.Commands))
	}

	clampConfig(&base)

	profiles := discoverProfiles(configDir)

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

	target := normalizeProfileName(requested)
	activeIndex := 0
	pivotStillValid := requestedPivot != ""

	if target != "" && target != "default" {
		found := false
		for i := 1; i < len(profiles); i++ {
			if normalizeProfileName(profiles[i].Name) == target {
				activeIndex = i
				found = true
				break
			}
		}
		if !found && strings.TrimSpace(requested) != "" {
			log.Printf("profile not found, falling back to default: %s", requested)
			if pivotRequested {
				if err := deletePivotProfile(configDir); err != nil {
					log.Printf("warning: could not delete pivot profile: %v", err)
				}
				pivotStillValid = false
			}
		}
	}

	effective := base
	effective.Theme = base.Theme
	if activeIndex > 0 {
		info := profiles[activeIndex]
		effective = applyProfileOverlay(base, info.Overlay)
		log.Printf("Applied profile overlay: %s", info.Name)
	}

	clampConfig(&effective)
	effective.Commands = copyCommands(effective.Commands)

	return configBundle{
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
	}
}
