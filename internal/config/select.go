package config

import (
	"fmt"
	"log"
	"slices"
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
	rest := slices.Clone(profiles) // a list, not a map: Git and git are two decks
	var ordered []ProfileInfo
	for _, n := range order {
		norm := profilepkg.NormalizeName(n)
		if i := slices.IndexFunc(rest, func(p ProfileInfo) bool { return profilepkg.NormalizeName(p.Name) == norm }); i >= 0 {
			ordered = append(ordered, rest[i])
			rest = slices.Delete(rest, i, i+1)
		}
	}
	sort.Slice(rest, func(i, j int) bool { return rest[i].Name < rest[j].Name })
	return append(ordered, rest...)
}

type profileRequest struct {
	Name      string // profile to activate; empty means "the first one"
	Dropped   string // a higher-priority source named this, but it is not equipped
	FromPivot bool
	StaleLock bool // the lock names something unequipped; clear it on disk
}

// resolveRequested picks which profile to activate:
// override > pivot lock > DRAKO_PROFILE env > config.toml profile.
// A source naming an unequipped profile is skipped rather than honoured and
// then failed — falling through is what a precedence list is for. A non-nil
// override is used verbatim.
func resolveRequested(override *string, pivotLocked, envProfile, cfgProfile string, available []ProfileInfo, broken []ProfileParseError) profileRequest {
	if override != nil {
		return profileRequest{Name: *override}
	}

	equipped := func(name string) bool {
		_, ok := selectProfile(available, name)
		return ok
	}
	unparsable := func(name string) bool {
		for _, b := range broken {
			if profilepkg.NormalizeName(b.Name) == profilepkg.NormalizeName(name) {
				return true
			}
		}
		return false
	}

	req := profileRequest{}
	for _, src := range []struct {
		name  string
		pivot bool
	}{
		{strings.TrimSpace(pivotLocked), true},
		{strings.TrimSpace(envProfile), false},
		{strings.TrimSpace(cfgProfile), false},
	} {
		if src.name == "" {
			continue
		}
		if equipped(src.name) {
			req.Name = src.name
			req.FromPivot = src.pivot
			return req
		}
		if unparsable(src.name) {
			continue // still equipped, reported as broken; a lock on it waits for the fix
		}
		if req.Dropped == "" {
			req.Dropped = src.name
		}
		if src.pivot {
			req.StaleLock = true
		}
	}
	return req
}

// buildEffective produces the effective config by overlaying the active
// profile on base. An invalid selected profile is reported in extraBroken and
// triggers the "first available" fallback policy: try the first profile, and
// if that fails too, land on the rescue config (always a valid grid).
// fellBack is true whenever the selected profile had to be abandoned, so the
// caller can drop a pivot lock that points at it. With no profiles, or with
// useFactoryDefaults set, the rescue config is used directly; rescue reports it.
func buildEffective(base Config, profiles []ProfileInfo, activeIndex int, useFactoryDefaults bool) (effective Config, newActive int, extraBroken []ProfileParseError, fellBack, rescue bool) {
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
		return effective, newActive, nil, false, true
	}

	effective, err := applyAndValidate(selected)
	if err == nil {
		log.Printf("Applied profile overlay: %s", selected.Name)
		return effective, newActive, nil, false, false
	}

	gridBroken := func(p ProfileInfo, err error) ProfileParseError {
		return ProfileParseError{Name: p.Name, Path: p.Path, Err: fmt.Sprintf("Grid validation failed: %v", err)}
	}
	log.Printf("Selected profile %q is invalid: %v. Falling back to defaults.", selected.Name, err)
	extraBroken = append(extraBroken, gridBroken(selected, err))
	// "First available" fallback policy: try the first profile;
	// if that fails too, fall back to rescue mode (a valid grid).
	selected = profiles[0]
	log.Printf("Falling back to first available profile: %s", selected.Name)
	fallbackConfig, err2 := applyAndValidate(selected)
	if err2 == nil {
		return fallbackConfig, 0, extraBroken, true, false
	}
	log.Printf("Fallback profile %s is also invalid: %v", selected.Name, err2)
	if activeIndex != 0 { // profile 0 was the selected one, already reported
		extraBroken = append(extraBroken, gridBroken(selected, err2))
	}
	effective = RescueConfig()
	effective.ApplyDefaults()
	return effective, 0, extraBroken, true, true
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
