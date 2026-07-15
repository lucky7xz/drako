package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/lucky7xz/drako/internal/paths"
)

// loadBaseConfig reads config.toml into the base Config. Every failure path
// lands on the rescue config: a missing file writes a fresh rescue config to
// disk (best effort), while read and syntax errors keep the broken file and
// report it as a ProfileParseError. Env vars in the file are expanded before
// decoding, and commands are always empty — those come from profile overlays.
func loadBaseConfig(configDir string) (Config, []ProfileParseError) {
	configPath := paths.ConfigFile(configDir)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Bootstrap failed to produce a config file; write a rescue
		// config so the next run starts from something valid.
		base := RescueConfig()
		writeRescue := func() error {
			f, err := os.Create(configPath)
			if err != nil {
				return err
			}
			defer f.Close()
			return toml.NewEncoder(f).Encode(base)
		}
		if err := writeRescue(); err != nil {
			log.Printf("warning: could not write rescue config: %v", err)
		}
		return base, nil
	}

	log.Printf("Loading config from: %s", configPath)
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("could not read config file: %v", err)
		return RescueConfig(), []ProfileParseError{{
			Name: paths.ConfigFileName,
			Path: configPath,
			Err:  fmt.Sprintf("Could not read config: %v", err),
		}}
	}

	configString := os.ExpandEnv(string(configBytes))

	var settings AppSettings
	if _, err := toml.Decode(configString, &settings); err != nil {
		log.Printf("could not decode config file: %v", err)
		return RescueConfig(), []ProfileParseError{{
			Name: paths.ConfigFileName,
			Path: configPath,
			Err:  fmt.Sprintf("SYNTAX ERROR: %v", err),
		}}
	}

	// Settings become the base config; commands only ever
	// come from profile overlays.
	log.Printf("Loaded base settings")
	return Config{
		DefaultShell:       settings.DefaultShell,
		NumbModifier:       settings.NumbModifier,
		Profile:            settings.Profile,
		LockTimeoutMinutes: settings.LockTimeoutMinutes,
		AutoLockEnabled:    settings.AutoLockEnabled,
		EnvWhitelist:       settings.EnvWhitelist,
		EnvBlocklist:       settings.EnvBlocklist,
		Theme:              settings.Theme,
		Keys:               settings.Keys,
		Commands:           []Command{},
	}, nil
}

// LoadConfig assembles the full configuration bundle: it bootstraps a
// missing config on first run, loads the base settings (falling back to
// rescue mode on read or syntax errors), discovers and orders the profiles,
// and applies the requested profile as an overlay. profileOverride takes
// precedence over the pivot lock, the DRAKO_PROFILE env var, and the
// profile named in config.toml, in that order. It errors only when the
// environment is unusable — no config directory can be resolved or created;
// everything else degrades to the rescue config and Broken entries.
func LoadConfig(profileOverride *string) (ConfigBundle, error) {
	return loadConfig(profileOverride, "")
}

// ReloadConfig is LoadConfig for a running session: sessionProfile is the
// profile the session currently has active, and it takes the env var's slot
// in the selection precedence (pivot lock still snaps back over it). An empty
// sessionProfile behaves exactly like LoadConfig(nil).
func ReloadConfig(sessionProfile string) (ConfigBundle, error) {
	return loadConfig(nil, sessionProfile)
}

func loadConfig(profileOverride *string, sessionProfile string) (ConfigBundle, error) {
	configDir, err := paths.ConfigDir()
	if err != nil {
		return ConfigBundle{}, fmt.Errorf("could not resolve a config directory: %w", err)
	}

	configPath := paths.ConfigFile(configDir)
	// First run: if config file is missing, ensure dir and copy embedded bootstrap assets
	if _, statErr := os.Stat(configPath); errors.Is(statErr, os.ErrNotExist) {
		if mkErr := os.MkdirAll(configDir, 0o755); mkErr != nil {
			return ConfigBundle{}, fmt.Errorf("could not create config directory: %w", mkErr)
		}
		if restored, err := RestoreBootstrap(configDir); err != nil {
			log.Printf("warning: bootstrap copy failed: %v", err)
		} else {
			log.Printf("bootstrap: restored %d file(s): %v", len(restored), restored)
		}
	}

	pf, err := ReadPivotProfile(configDir)
	if err != nil {
		log.Printf("warning: could not read pivot profile: %v", err)
		pf = pivotFile{}
	}
	pivotRequested := false
	requestedPivot := strings.TrimSpace(pf.Locked)

	base, broken := loadBaseConfig(configDir)

	base.ApplyDefaults()

	if err := ValidateConfig(base); err != nil {
		log.Printf("Error: Base config is invalid: %v", err)
	}

	profiles, profileErrors := DiscoverProfilesWithErrors(configDir)
	broken = append(broken, profileErrors...)

	profiles = reorderByPivot(profiles, pf.EquippedOrder)

	// The session's active profile occupies the env var's precedence slot:
	// the env var seeds the first load, the session carries it from there.
	envProfile := sessionProfile
	if envProfile == "" {
		envProfile = os.Getenv("DRAKO_PROFILE")
	}

	requested, fromPivot := resolveRequested(profileOverride, pf.Locked, envProfile, base.Profile)
	pivotRequested = fromPivot

	pivotStillValid := requestedPivot != ""
	useFactoryDefaults := false

	droppedProfile := ""
	activeIndex, found := selectProfile(profiles, requested)
	if !found {
		log.Printf("profile not found (possibly broken), falling back to factory defaults: %s", requested)
		useFactoryDefaults = true
		// The requested profile (from the lock, the session, or config.toml)
		// vanished — deleted or moved to inventory. Record its name so the UI
		// can explain the otherwise-silent rescue drop.
		droppedProfile = requested
		if pivotRequested {
			// It was the locked profile: clear the now-stale lock too.
			if err := WritePivotLocked(configDir, ""); err != nil {
				log.Printf("warning: could not clear pivot lock: %v", err)
			}
			pivotStillValid = false
		}
	}

	effective, activeIndex, extraBroken, fellBack := buildEffective(base, profiles, activeIndex, useFactoryDefaults)
	broken = append(broken, extraBroken...)
	if fellBack {
		pivotStillValid = false
	}

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
		Broken:         broken,
		DroppedProfile: droppedProfile,
	}, nil
}
