package lair

import (
	"testing"

	"github.com/lucky7xz/drako/internal/profile"
	"github.com/lucky7xz/drako/internal/settings"
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
	all := []ProfileInfo{
		validProfile("work"), validProfile("env"), validProfile("envprofile"),
		validProfile("cfg"), validProfile("cfgprofile"),
	}
	tests := []struct {
		name            string
		pivot, env, cfg string
		available       []ProfileInfo
		want            profileRequest
	}{
		{"pivot beats env and cfg", "work", "env", "cfg", all,
			profileRequest{Name: "work", FromPivot: true}},
		{"env beats cfg", "", "envprofile", "cfg", all,
			profileRequest{Name: "envprofile"}},
		{"cfg is the last resort", "", "", "cfgprofile", all,
			profileRequest{Name: "cfgprofile"}},
		{"whitespace pivot is no pivot", "   ", " env ", "", all,
			profileRequest{Name: "env"}},
		{"nothing requested", "", "", "", all,
			profileRequest{}},

		// Stashing the deck a source names must fall through to the next
		// source, not be honoured and then fail downstream.
		{"stashed lock falls through to env", "gone", "env", "cfg", all,
			profileRequest{Name: "env", Dropped: "gone", StaleLock: true}},
		{"stashed session falls through to cfg", "", "gone", "cfg", all,
			profileRequest{Name: "cfg", Dropped: "gone"}},
		{"only the first casualty is reported", "gone", "alsogone", "cfg", all,
			profileRequest{Name: "cfg", Dropped: "gone", StaleLock: true}},
		{"nothing equipped leaves no name", "gone", "", "", nil,
			profileRequest{Dropped: "gone", StaleLock: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveRequested(tt.pivot, tt.env, tt.cfg, tt.available, nil)
			if got != tt.want {
				t.Errorf("resolveRequested = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func validProfile(name string) ProfileInfo {
	return ProfileInfo{Name: name, Path: name + ".profile.toml", Profile: profile.File{
		X: 2, Y: 2,
		Commands: []profile.Command{{Name: name + "-cmd", Command: "true", Row: 0, Col: "a"}},
	}}
}

func invalidProfile(name string) ProfileInfo {
	return ProfileInfo{Name: name, Path: name + ".profile.toml", Profile: profile.File{
		X: 1, Y: 1,
		Commands: []profile.Command{{Name: "out-of-bounds", Command: "true", Row: 5, Col: "a"}},
	}}
}

func TestBuildEffective_AppliesSelectedProfile(t *testing.T) {
	profiles := []ProfileInfo{validProfile("alpha"), validProfile("work")}

	effective, rescue := buildEffective(settings.Config{}, profiles, 1, "")
	if rescue {
		t.Error("a deck was selected; this is not the rescue grid")
	}
	if len(effective.Commands) != 1 || effective.Commands[0].Name != "work-cmd" {
		t.Errorf("work overlay not applied: %+v", effective.Commands)
	}
}

func TestBuildEffective_NoProfilesIsRescue(t *testing.T) {
	effective, rescue := buildEffective(settings.Config{}, nil, 0, "")
	if !rescue {
		t.Error("no profiles means the rescue grid")
	}
	if len(effective.Commands) == 0 {
		t.Error("the rescue grid should carry its helper commands")
	}
}

// A deck with a cell outside its grid is still the deck that was asked for:
// it loads, and the cell that fits nowhere is simply not drawn (I-03).
func TestBuildEffective_FlawedDeckStillLoads(t *testing.T) {
	profiles := []ProfileInfo{validProfile("alpha"), invalidProfile("flawed")}

	effective, rescue := buildEffective(settings.Config{}, profiles, 1, "")
	if rescue {
		t.Fatal("a flawed deck must not send the user to the rescue grid")
	}
	if len(effective.Commands) != 1 || effective.Commands[0].Name != "out-of-bounds" {
		t.Errorf("the selected deck was not applied: %+v", effective.Commands)
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

// Two decks whose names differ only in case are two decks on a case-sensitive
// filesystem; ordering must not drop one of them.
func TestReorderByPivot_KeepsCaseVariants(t *testing.T) {
	in := []ProfileInfo{{Name: "Git"}, {Name: "alpha"}, {Name: "git"}}
	got := reorderByPivot(in, []string{"alpha"})
	if len(got) != len(in) {
		t.Fatalf("got %d profiles, want %d: %v", len(got), len(in), got)
	}
}
