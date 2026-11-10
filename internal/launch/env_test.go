package launch

import (
	"slices"
	"testing"
)

// Commands inherit drako's whole environment; only DRAKO_PROFILE is added.
func TestEnv_KeepsTheInheritedEnvironment(t *testing.T) {
	env := Env([]string{"PATH=/usr/bin", "SECRET=hunter2"}, "work")
	for _, want := range []string{"PATH=/usr/bin", "SECRET=hunter2", "DRAKO_PROFILE=work"} {
		if !slices.Contains(env, want) {
			t.Errorf("env %v is missing %s", env, want)
		}
	}
}

func TestEnv_StaleProfileReplaced(t *testing.T) {
	base := []string{"DRAKO_PROFILE=old", "PATH=/usr/bin"}
	env := Env(base, "new")
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

func TestEnv_NoProfileKeepsInherited(t *testing.T) {
	base := []string{"DRAKO_PROFILE=external", "PATH=/usr/bin"}
	env := Env(base, "")
	if !slices.Contains(env, "DRAKO_PROFILE=external") {
		t.Errorf("without an active profile, an externally set value passes through, got %v", env)
	}
}
