package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	profilepkg "github.com/lucky7xz/drako/internal/profiles"
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

// CheckProfileFile reports whether the file at path parses and validates as
// a profile. Used for immediate feedback after in-app edits.
func CheckProfileFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not read file: %w", err)
	}
	var pf ProfileFile
	meta, err := toml.Decode(string(data), &pf)
	if err != nil {
		return fmt.Errorf("invalid TOML: %w", err)
	}
	if ok, problems := ValidateProfileFile(pf, data, meta); !ok {
		return fmt.Errorf("%s", strings.Join(problems, "; "))
	}
	return nil
}

// ValidateProfileFile checks a profile has at least one command, x/y in 1-9,
// and no unrecognized top-level keys (meta names the typo, e.g. "ssy").
func ValidateProfileFile(pf ProfileFile, raw []byte, meta toml.MetaData) (bool, []string) {
	var problems []string
	if len(pf.Commands) == 0 {
		problems = append(problems, "needs at least one command")
	}
	problems = append(problems, validateGridDim("x", pf.X, raw)...)
	problems = append(problems, validateGridDim("y", pf.Y, raw)...)
	if unknown := undecodedTopLevelKeys(meta); len(unknown) > 0 {
		problems = append(problems, fmt.Sprintf("unrecognized key(s): %s (check for a typo)", strings.Join(unknown, ", ")))
	}
	return len(problems) == 0, problems
}

// undecodedTopLevelKeys returns top-level keys that matched no field at all
// — a longer path (e.g. "commands.5.foo") means the top level DID match.
func undecodedTopLevelKeys(meta toml.MetaData) []string {
	var keys []string
	for _, k := range meta.Undecoded() {
		if len(k) == 1 {
			keys = append(keys, k[0])
		}
	}
	return keys
}

// validateGridDim distinguishes "key absent" from "present but out of
// range" instead of reporting the zero-default as if it were typed.
func validateGridDim(key string, val int, raw []byte) []string {
	if val >= 1 && val <= 9 {
		return nil
	}
	line := findKeyLine(raw, key)
	if line == 0 {
		return []string{fmt.Sprintf("%s is missing (must be 1-9)", key)}
	}
	return []string{fmt.Sprintf("%s = %d is invalid (must be 1-9) (line %d)", key, val, line)}
}

// findKeyLine returns the 1-based line of a `key =` / `key=` assignment, or 0.
func findKeyLine(raw []byte, key string) int {
	for i, line := range strings.Split(string(raw), "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, key+"=") || strings.HasPrefix(t, key+" =") {
			return i + 1
		}
	}
	return 0
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
		if !strings.HasSuffix(name, profilepkg.ProfileSuffix) {
			continue
		}
		fullPath := filepath.Join(configDir, name)
		profileName := strings.TrimSuffix(name, profilepkg.ProfileSuffix)

		raw, err := os.ReadFile(fullPath)
		if err != nil {
			log.Printf("Failed to read profile %s: %v", entry.Name(), err)
			broken = append(broken, ProfileParseError{Name: profileName, Path: fullPath, Err: err.Error()})
			continue
		}

		var profileFile ProfileFile
		meta, err := toml.Decode(string(raw), &profileFile)
		if err != nil {
			log.Printf("Failed to parse profile %s: %v", entry.Name(), err)
			broken = append(broken, ProfileParseError{Name: profileName, Path: fullPath, Err: err.Error()})
			continue
		}

		if ok, problems := ValidateProfileFile(profileFile, raw, meta); !ok {
			broken = append(broken, ProfileParseError{
				Name: profileName,
				Path: fullPath,
				Err:  strings.Join(problems, "; "),
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
