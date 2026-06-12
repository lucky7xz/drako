package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// CopyCommands returns an independent copy of a command slice so callers
// can mutate it without affecting the source.
func CopyCommands(src []Command) []Command {
	if len(src) == 0 {
		return []Command{}
	}
	dst := make([]Command, len(src))
	copy(dst, src)
	return dst
}

// ValidateProfileFile checks if the profile has minimum required fields.
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

// NormalizeProfileName lowercases a profile reference and strips the known
// file suffixes, so "Git", "git.profile.toml" and "git.profile" all
// address the same profile.
func NormalizeProfileName(name string) string {
	n := strings.TrimSpace(strings.ToLower(name))
	n = strings.TrimSuffix(n, ".profile.toml")
	n = strings.TrimSuffix(n, ".toml")
	n = strings.TrimSuffix(n, ".profile")
	return n
}

// DiscoverProfilesWithErrors scans configDir for *.profile.toml files and
// returns the parseable profiles (sorted by name) plus a parse-error entry
// for every file that could not be loaded or fails validation.
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
		fullPath := filepath.Join(configDir, name)
		profileName := strings.TrimSuffix(name, ".profile.toml")

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

// DiscoverProfiles is DiscoverProfilesWithErrors without the error report.
func DiscoverProfiles(configDir string) []ProfileInfo {
	profiles, _ := DiscoverProfilesWithErrors(configDir)
	return profiles
}

// ApplyProfileOverlay layers a profile on top of the base settings: grid
// size and commands always come from the profile; theme, header art and
// shell only when the profile sets them.
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
	cfg.Commands = CopyCommands(profile.Commands)

	return cfg
}
