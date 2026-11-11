package lair

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"strings"

	"github.com/lucky7xz/drako/internal/profile"
	"github.com/lucky7xz/drako/internal/settings"
)

// loadBaseConfig reads config.toml into the base Config. Every failure path
// lands on the rescue config: a missing file gets the settings template written
// (best effort), while read and syntax errors keep the broken file and
// report it as a ProfileParseError. Commands are always empty — those come
// from profile overlays.
func (l *Lair) loadBaseConfig() (settings.Config, []ProfileParseError) {
	configDir := l.dir
	configPath := ConfigFile(configDir)

	if _, err := l.stat(ConfigFileName); errors.Is(err, fs.ErrNotExist) {
		// Bootstrap failed to produce a config file; run on rescue and write the
		// settings template so the next run starts from something valid.
		if _, err := l.writeSettingsTemplate(); err != nil {
			log.Printf("warning: could not write config.toml: %v", err)
		}
		return settings.RescueConfig(configDir), nil
	}

	log.Printf("Loading config from: %s", configPath)
	configBytes, err := l.read(ConfigFileName)
	if err != nil {
		log.Printf("could not read config file: %v", err)
		return settings.RescueConfig(configDir), []ProfileParseError{{
			Name: ConfigFileName,
			Path: configPath,
			Err:  fmt.Sprintf("Could not read config: %v", err),
		}}
	}

	base, unknown, err := settings.Decode(configBytes)
	if err != nil {
		log.Printf("could not decode config file: %v", err)
		return settings.RescueConfig(configDir), []ProfileParseError{{
			Name: ConfigFileName,
			Path: configPath,
			Err:  fmt.Sprintf("SYNTAX ERROR: %v", err),
		}}
	}

	if len(unknown) > 0 { // drako check reports them
		log.Printf("config.toml: ignoring unrecognized key(s): %s", strings.Join(unknown, ", "))
	}

	// Settings become the base config; commands only ever
	// come from profile overlays.
	log.Printf("Loaded base settings")
	return base, nil
}

// LoadConfig assembles the full configuration bundle from configDir: it
// bootstraps a missing config on first run, loads the base settings (falling
// back to rescue mode on read or syntax errors), discovers and orders the
// profiles, and applies the requested profile as an overlay.
//
// Everything it needs to know is handed in. targets are the platform's variant
// keys (host.Targets), which decide each command's text. wanted is the profile
// asked for from outside: the one a running session has active, else the
// DRAKO_PROFILE environment variable; the pivot lock still wins over it, and
// the profile named in config.toml is the fallback.
//
// It errors only when the config directory cannot be made or opened;
// everything else degrades to the rescue config and Broken entries.
func LoadConfig(configDir string, targets []string, wanted string) (ConfigBundle, error) {
	// One handle for the whole load: every file below is reached through it.
	// The directory always exists once a load is over, so make it now.
	l, err := Create(configDir)
	if err != nil {
		return ConfigBundle{}, err
	}
	defer l.Close()

	// First run: if config file is missing, copy embedded bootstrap assets
	if _, statErr := l.stat(ConfigFileName); errors.Is(statErr, fs.ErrNotExist) {
		log.Printf("bootstrap: restoring missing embedded files into %s", configDir)
		if restored, err := l.restoreBootstrap(); err != nil {
			log.Printf("warning: bootstrap copy failed: %v", err)
		} else {
			log.Printf("bootstrap: restored %d file(s): %v", len(restored), restored)
		}
	}

	pf, err := l.readPivot()
	if err != nil {
		log.Printf("warning: could not read pivot profile: %v", err)
		pf = pivotFile{}
	}
	requestedPivot := strings.TrimSpace(pf.Locked)

	base, broken := l.loadBaseConfig()

	base.ApplyDefaults()

	if err := profile.ValidateGrid(base.X, base.Y, base.Commands); err != nil {
		log.Printf("Error: Base config is invalid: %v", err)
	}

	profiles, profileErrors := l.discoverProfiles(targets)
	broken = append(broken, profileErrors...)

	profiles = reorderByPivot(profiles, pf.EquippedOrder)

	envProfile := wanted

	req := resolveRequested(pf.Locked, envProfile, base.Profile, profiles, profileErrors)

	pivotStillValid := requestedPivot != ""

	droppedProfile := req.Dropped
	if droppedProfile != "" {
		log.Printf("requested profile is not equipped, falling through: %s", droppedProfile)
	}
	if req.StaleLock {
		if err := l.lockProfile(""); err != nil {
			log.Printf("warning: could not clear pivot lock: %v", err)
		}
		pivotStillValid = false
	}

	activeIndex, _ := selectProfile(profiles, req.Name) // resolveRequested only names equipped decks
	effective, activeIndex, extraBroken, fellBack, rescue := buildEffective(base, profiles, activeIndex, configDir)
	broken = append(broken, extraBroken...)
	if fellBack {
		pivotStillValid = false
	}

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
		Rescue:         rescue,
	}, nil
}
