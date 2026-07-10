package core

import (
	"slices"
	"testing"
)

func TestCommandEnv_WhitelistFilters(t *testing.T) {
	base := []string{"PATH=/usr/bin", "SECRET=hunter2"}
	env := commandEnv(base, []string{"PATH"}, "")
	if !slices.Contains(env, "PATH=/usr/bin") {
		t.Error("whitelisted PATH should survive")
	}
	if slices.Contains(env, "SECRET=hunter2") {
		t.Error("SECRET is not whitelisted and must be dropped")
	}
}

func TestCommandEnv_ProfileInjectedDespiteWhitelist(t *testing.T) {
	base := []string{"PATH=/usr/bin"}
	env := commandEnv(base, []string{"PATH"}, "work")
	if !slices.Contains(env, "DRAKO_PROFILE=work") {
		t.Errorf("drako's own var must reach children regardless of whitelist, got %v", env)
	}
}

func TestCommandEnv_StaleProfileReplaced(t *testing.T) {
	base := []string{"DRAKO_PROFILE=old", "PATH=/usr/bin"}
	env := commandEnv(base, nil, "new")
	if slices.Contains(env, "DRAKO_PROFILE=old") {
		t.Errorf("stale inherited DRAKO_PROFILE must be replaced, got %v", env)
	}
	count := 0
	for _, e := range env {
		if e == "DRAKO_PROFILE=new" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("want exactly one DRAKO_PROFILE=new, got %v", env)
	}
}

func TestCommandEnv_NoProfileKeepsInherited(t *testing.T) {
	base := []string{"DRAKO_PROFILE=external", "PATH=/usr/bin"}
	env := commandEnv(base, nil, "")
	if !slices.Contains(env, "DRAKO_PROFILE=external") {
		t.Errorf("without an active profile, an externally set value passes through, got %v", env)
	}
}
