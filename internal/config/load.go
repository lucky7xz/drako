package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/lucky7xz/drako/internal/paths"
)

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	log.Fatalf(format, args...)
}

// LoadConfig assembles the full configuration bundle: it bootstraps a
// missing config on first run, loads the base settings (falling back to
// rescue mode on read or syntax errors), discovers and orders the profiles,
// and applies the requested profile as an overlay. profileOverride takes
// precedence over the pivot lock, the DRAKO_PROFILE env var, and the
// profile named in config.toml, in that order.
func LoadConfig(profileOverride *string) ConfigBundle {
	configDir, err := paths.ConfigDir()
	if err != nil {
		fatalf("could not resolve a config directory: %v", err)
	}

	configPath := paths.ConfigFile(configDir)
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
	var broken []ProfileParseError

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Bootstrap failed to produce a config file; write a rescue
		// config so the next run starts from something valid.
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
			log.Printf("could not read config file: %v", err)
			base = RescueConfig()
			broken = append(broken, ProfileParseError{
				Name: paths.ConfigFileName,
				Path: configPath,
				Err:  fmt.Sprintf("Could not read config: %v", err),
			})
		} else {
			configString := os.ExpandEnv(string(configBytes))

			var settings AppSettings
			if _, err := toml.Decode(configString, &settings); err != nil {
				log.Printf("could not decode config file: %v", err)
				base = RescueConfig()
				broken = append(broken, ProfileParseError{
					Name: paths.ConfigFileName,
					Path: configPath,
					Err:  fmt.Sprintf("SYNTAX ERROR: %v", err),
				})
			} else {
				// Settings become the base config; commands only ever
				// come from profile overlays.
				base = Config{
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
				}
				log.Printf("Loaded base settings")
			}
		}
	}

	base.ApplyDefaults()

	if err := ValidateConfig(base); err != nil {
		log.Printf("Error: Base config is invalid: %v", err)
	}

	profiles, profileErrors := DiscoverProfilesWithErrors(configDir)
	broken = append(broken, profileErrors...)

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
	var selected ProfileInfo

	if len(profiles) > 0 {
		selected = profiles[activeIndex]
	} else {
		// No profiles found; enforce factory defaults with a stand-in
		// profile for logging and selection logic.
		useFactoryDefaults = true
		selected = ProfileInfo{Name: "Rescue", Path: "internal", Profile: ProfileFile{}}
	}

	applyAndValidate := func(p ProfileInfo) (Config, error) {
		temp := ApplyProfileOverlay(base, p.Profile)
		temp.ApplyDefaults()
		if err := ValidateConfig(temp); err != nil {
			return temp, err
		}
		return temp, nil
	}

	if useFactoryDefaults {
		effective = RescueConfig()
		effective.ApplyDefaults()
	} else {
		var err error
		effective, err = applyAndValidate(selected)
		if err != nil {
			log.Printf("Selected profile %q is invalid: %v. Falling back to defaults.", selected.Name, err)
			broken = append(broken, ProfileParseError{
				Name: selected.Name,
				Path: selected.Path,
				Err:  fmt.Sprintf("Grid validation failed: %v", err),
			})
			// "First available" fallback policy: try the first profile;
			// if that fails too, stay on the base settings.
			effective = base
			activeIndex = 0
			pivotStillValid = false
			if len(profiles) > 0 {
				selected = profiles[0]
				log.Printf("Falling back to first available profile: %s", selected.Name)
				fallbackConfig, err2 := applyAndValidate(selected)
				if err2 == nil {
					effective = fallbackConfig
					activeIndex = 0
				} else {
					log.Printf("Fallback profile %s is also invalid: %v", selected.Name, err2)
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
