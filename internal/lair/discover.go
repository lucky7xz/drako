package lair

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/lucky7xz/drako/internal/profile"
	profilepkg "github.com/lucky7xz/drako/internal/profiles"
)

// CheckProfileFile reports whether the file at path parses and validates as
// a profile. Used for immediate feedback after in-app edits.
func CheckProfileFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not read file: %w", err)
	}
	_, problems, err := profile.Decode(data, nil) // validating only: no platform needed
	if err != nil {
		return fmt.Errorf("invalid TOML: %w", err)
	}
	if len(problems) > 0 {
		return fmt.Errorf("%s", strings.Join(problems, "; "))
	}
	return nil
}

// undecodedKeys returns every key path that matched no field, e.g. "keys.lader".
func undecodedKeys(meta toml.MetaData) []string {
	var keys []string
	for _, k := range meta.Undecoded() {
		keys = append(keys, k.String())
	}
	return keys
}

// DiscoverProfilesWithErrors scans configDir for *.profile.toml files and
// returns the parseable profiles (sorted by name) plus a parse-error entry
// for every file that could not be loaded or fails validation. Commands are
// resolved for targets, the platform's variant keys.
func DiscoverProfilesWithErrors(configDir string, targets []string) ([]ProfileInfo, []ProfileParseError) {
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

		profileFile, problems, err := profile.Decode(raw, targets)
		if err != nil {
			log.Printf("Failed to parse profile %s: %v", entry.Name(), err)
			broken = append(broken, ProfileParseError{Name: profileName, Path: fullPath, Err: err.Error()})
			continue
		}
		if len(problems) > 0 {
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
