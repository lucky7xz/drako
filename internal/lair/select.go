package lair

import (
	"log"
	"strings"

	"github.com/lucky7xz/drako/internal/equip"
	"github.com/lucky7xz/drako/internal/settings"
)

// Profile selection policy, factored out of LoadConfig so each rule is a
// pure function: same inputs, same choice, no filesystem.

// reorderByPivot arranges profiles in the pivot's equipped order, the same
// equip.Order the inventory uses, so the two never disagree.
func reorderByPivot(profiles []ProfileInfo, order []string) []ProfileInfo {
	byName := make(map[string]ProfileInfo, len(profiles))
	names := make([]string, len(profiles))
	for i, p := range profiles {
		byName[p.Name], names[i] = p, p.Name
	}
	out := make([]ProfileInfo, 0, len(profiles))
	for _, n := range equip.Order(names, order) {
		out = append(out, byName[n])
	}
	return out
}

type profileRequest struct {
	Name      string // profile to activate; empty means "the first one"
	Dropped   string // a higher-priority source named this, but it is not equipped
	FromPivot bool
	StaleLock bool // the lock names something unequipped; clear it on disk
}

// resolveRequested picks which profile to activate:
// pivot lock > DRAKO_PROFILE env > config.toml profile.
// A source naming an unequipped profile is skipped rather than honoured and
// then failed — falling through is what a precedence list is for.
func resolveRequested(pivotLocked, envProfile, cfgProfile string, available []ProfileInfo, broken []ProfileParseError) profileRequest {
	equipped := func(name string) bool {
		_, ok := selectProfile(available, name)
		return ok
	}
	unparsable := func(name string) bool {
		for _, b := range broken {
			if equip.NormalizeName(b.Name) == equip.NormalizeName(name) {
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
// profile on base. Every discovered deck has a grid (the loader refuses one
// without), so the overlay cannot fail; a flawed deck loads and carries its
// Flaws. With no profiles the rescue config is used; rescue reports it.
func buildEffective(base settings.Config, profiles []ProfileInfo, activeIndex int, configDir string) (effective settings.Config, rescue bool) {
	if len(profiles) == 0 {
		effective = settings.RescueConfig(configDir)
		effective.ApplyDefaults()
		return effective, true
	}
	selected := profiles[activeIndex]
	effective = settings.ApplyProfileOverlay(base, selected.Profile)
	effective.ApplyDefaults()
	log.Printf("Applied profile overlay: %s", selected.Name)
	return effective, false
}

// selectProfile finds requested among profiles by normalized name. An empty
// request means the first profile; found is false only when a non-empty
// request matches nothing.
func selectProfile(profiles []ProfileInfo, requested string) (activeIndex int, found bool) {
	target := equip.NormalizeName(requested)
	if target == "" {
		return 0, true
	}
	for i := range profiles {
		if equip.NormalizeName(profiles[i].Name) == target {
			return i, true
		}
	}
	return 0, false
}
