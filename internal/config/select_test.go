package config

import (
	"strings"
	"testing"
)

func infos(names ...string) []ProfileInfo {
	out := make([]ProfileInfo, len(names))
	for i, n := range names {
		out[i] = ProfileInfo{Name: n}
	}
	return out
}

func names(profiles []ProfileInfo) []string {
	out := make([]string, len(profiles))
	for i, p := range profiles {
		out[i] = p.Name
	}
	return out
}

func TestReorderByPivot(t *testing.T) {
	tests := []struct {
		name     string
		profiles []string
		order    []string
		want     []string
	}{
		{"empty order is a no-op", []string{"a", "b"}, nil, []string{"a", "b"}},
		{"full reorder", []string{"a", "b", "c"}, []string{"c", "a", "b"}, []string{"c", "a", "b"}},
		{"unknown names ignored", []string{"a", "b"}, []string{"ghost", "b"}, []string{"b", "a"}},
		{"leftovers appended name-sorted", []string{"zeta", "beta", "work"}, []string{"work"}, []string{"work", "beta", "zeta"}},
		{"matching is normalized", []string{"Work"}, []string{"work.profile.toml"}, []string{"Work"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := names(reorderByPivot(infos(tt.profiles...), tt.order))
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestResolveRequested(t *testing.T) {
	override := func(s string) *string { return &s }
	tests := []struct {
		name                     string
		override                 *string
		pivot, env, cfg          string
		want                     string
		wantFromPivot            bool
	}{
		{"override wins over everything", override("alpha"), "work", "env", "cfg", "alpha", false},
		{"empty override still wins (verbatim)", override(""), "work", "env", "cfg", "", false},
		{"pivot beats env and cfg", nil, "work", "env", "cfg", "work", true},
		{"env beats cfg", nil, "", "envprofile", "cfg", "envprofile", false},
		{"cfg is the last resort", nil, "", "", "cfgprofile", "cfgprofile", false},
		{"whitespace pivot is no pivot", nil, "   ", " env ", "", "env", false},
		{"nothing requested", nil, "", "", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, fromPivot := resolveRequested(tt.override, tt.pivot, tt.env, tt.cfg)
			if got != tt.want || fromPivot != tt.wantFromPivot {
				t.Errorf("resolveRequested = (%q, %v), want (%q, %v)", got, fromPivot, tt.want, tt.wantFromPivot)
			}
		})
	}
}

func validProfile(name string) ProfileInfo {
	return ProfileInfo{Name: name, Path: name + ".profile.toml", Profile: ProfileFile{
		X: 2, Y: 2,
		Commands: []Command{{Name: name + "-cmd", Command: "true", Row: 0, Col: "a"}},
	}}
}

func invalidProfile(name string) ProfileInfo {
	return ProfileInfo{Name: name, Path: name + ".profile.toml", Profile: ProfileFile{
		X: 1, Y: 1,
		Commands: []Command{{Name: "out-of-bounds", Command: "true", Row: 5, Col: "a"}},
	}}
}

func TestBuildEffective_AppliesSelectedProfile(t *testing.T) {
	profiles := []ProfileInfo{validProfile("alpha"), validProfile("work")}

	effective, active, extraBroken, fellBack := buildEffective(Config{}, profiles, 1, false)
	if len(extraBroken) != 0 || fellBack {
		t.Fatalf("clean profile should not fall back: broken=%+v fellBack=%v", extraBroken, fellBack)
	}
	if active != 1 {
		t.Errorf("active index should be untouched, got %d", active)
	}
	if len(effective.Commands) != 1 || effective.Commands[0].Name != "work-cmd" {
		t.Errorf("work overlay not applied: %+v", effective.Commands)
	}
}

func TestBuildEffective_FactoryDefaultsSkipOverlay(t *testing.T) {
	profiles := []ProfileInfo{validProfile("alpha")}

	effective, _, extraBroken, fellBack := buildEffective(Config{}, profiles, 0, true)
	if len(extraBroken) != 0 || fellBack {
		t.Fatalf("factory defaults are not a fallback: broken=%+v fellBack=%v", extraBroken, fellBack)
	}
	if len(effective.Commands) == 0 {
		t.Error("factory defaults should carry the rescue helper commands")
	}
}

func TestBuildEffective_NoProfilesForcesFactoryDefaults(t *testing.T) {
	effective, active, extraBroken, fellBack := buildEffective(Config{}, nil, 0, false)
	if len(extraBroken) != 0 || fellBack || active != 0 {
		t.Fatalf("no profiles is not a fallback: broken=%+v fellBack=%v active=%d", extraBroken, fellBack, active)
	}
	if len(effective.Commands) == 0 {
		t.Error("factory defaults should carry the rescue helper commands")
	}
}

func TestBuildEffective_InvalidProfileFallsBackToFirst(t *testing.T) {
	profiles := []ProfileInfo{validProfile("alpha"), invalidProfile("broken")}

	effective, active, extraBroken, fellBack := buildEffective(Config{}, profiles, 1, false)
	if !fellBack {
		t.Fatal("invalid profile must report fellBack")
	}
	if len(extraBroken) != 1 || !strings.Contains(extraBroken[0].Err, "Grid validation failed") {
		t.Fatalf("want one Grid validation failed entry, got %+v", extraBroken)
	}
	if active != 0 {
		t.Errorf("fallback should land on the first profile, got index %d", active)
	}
	if len(effective.Commands) != 1 || effective.Commands[0].Name != "alpha-cmd" {
		t.Errorf("first-profile overlay not applied: %+v", effective.Commands)
	}
}

func TestBuildEffective_AllInvalidLandsOnRescue(t *testing.T) {
	profiles := []ProfileInfo{invalidProfile("broken")}

	effective, active, extraBroken, fellBack := buildEffective(Config{}, profiles, 0, false)
	if !fellBack || active != 0 {
		t.Fatalf("want fellBack at index 0, got fellBack=%v active=%d", fellBack, active)
	}
	if len(extraBroken) != 1 {
		t.Fatalf("want one broken entry for the selected profile, got %+v", extraBroken)
	}
	if len(effective.Commands) == 0 {
		t.Error("rescue config should carry its helper commands")
	}
}

func TestSelectProfile(t *testing.T) {
	profiles := infos("alpha", "beta", "work")
	tests := []struct {
		name      string
		requested string
		wantIdx   int
		wantFound bool
	}{
		{"empty request means first profile", "", 0, true},
		{"finds by name", "work", 2, true},
		{"normalized matching", "WORK.profile.toml", 2, true},
		{"missing profile reports not found", "ghost", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, found := selectProfile(profiles, tt.requested)
			if idx != tt.wantIdx || found != tt.wantFound {
				t.Errorf("selectProfile(%q) = (%d, %v), want (%d, %v)", tt.requested, idx, found, tt.wantIdx, tt.wantFound)
			}
		})
	}
}
