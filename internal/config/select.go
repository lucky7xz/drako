package config

import (
	"fmt"
	"log"
	"sort"
	"strings"

	profilepkg "github.com/lucky7xz/drako/internal/profiles"
)

// Profile selection policy, factored out of LoadConfig so each rule is a
// pure function: same inputs, same choice, no filesystem.

// reorderByPivot arranges profiles to match the pivot's equipped order;
// profiles not named in order are appended sorted by name. An empty order
// returns profiles unchanged.
func reorderByPivot(profiles []ProfileInfo, order []string) []ProfileInfo {
	if len(order) == 0 {
		return profiles
	}
	remaining := map[string]ProfileInfo{}
	for _, p := range profiles {
		remaining[profilepkg.NormalizeName(p.Name)] = p
	}
	var ordered []ProfileInfo
	for _, n := range order {
		norm := profilepkg.NormalizeName(n)
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
	return ordered
}

// resolveRequested picks which profile name to activate:
// override > pivot lock > DRAKO_PROFILE env > config.toml profile.
// A non-nil override is used verbatim; the rest are trimmed.
func resolveRequested(override *string, pivotLocked, envProfile, cfgProfile string) (requested string, fromPivot bool) {
	if override != nil {
		return *override, false
	}
	if pivot := strings.TrimSpace(pivotLocked); pivot != "" {
		return pivot, true
	}
	if env := strings.TrimSpace(envProfile); env != "" {
		return env, false
	}
	return strings.TrimSpace(cfgProfile), false
}

// buildEffective produces the effective config by overlaying the active
// profile on base. An invalid selected profile is reported in extraBroken and
// triggers the "first available" fallback policy: try the first profile, and
// if that fails too, land on the rescue config (always a valid grid).
// fellBack is true whenever the selected profile had to be abandoned, so the
// caller can drop a pivot lock that points at it. With no profiles, or with
// useFactoryDefaults set, the rescue config is used directly.
func buildEffective(base Config, profiles []ProfileInfo, activeIndex int, useFactoryDefaults bool) (effective Config, newActive int, extraBroken []ProfileParseError, fellBack bool) {
	newActive = activeIndex

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
		return effective, newActive, nil, false
	}

	effective, err := applyAndValidate(selected)
	if err == nil {
		log.Printf("Applied profile overlay: %s", selected.Name)
		return effective, newActive, nil, false
	}

	log.Printf("Selected profile %q is invalid: %v. Falling back to defaults.", selected.Name, err)
	extraBroken = append(extraBroken, ProfileParseError{
		Name: selected.Name,
		Path: selected.Path,
		Err:  fmt.Sprintf("Grid validation failed: %v", err),
	})
	// "First available" fallback policy: try the first profile;
	// if that fails too, fall back to rescue mode (a valid grid).
	effective = RescueConfig()
	effective.ApplyDefaults()
	newActive = 0
	selected = profiles[0]
	log.Printf("Falling back to first available profile: %s", selected.Name)
	fallbackConfig, err2 := applyAndValidate(selected)
	if err2 == nil {
		effective = fallbackConfig
	} else {
		log.Printf("Fallback profile %s is also invalid: %v", selected.Name, err2)
	}
	return effective, newActive, extraBroken, true
}

// selectProfile finds requested among profiles by normalized name. An empty
// request means the first profile; found is false only when a non-empty
// request matches nothing.
func selectProfile(profiles []ProfileInfo, requested string) (activeIndex int, found bool) {
	target := profilepkg.NormalizeName(requested)
	if target == "" {
		return 0, true
	}
	for i := range profiles {
		if profilepkg.NormalizeName(profiles[i].Name) == target {
			return i, true
		}
	}
	return 0, false
}
