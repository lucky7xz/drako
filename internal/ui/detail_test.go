package ui

import (
	"testing"

	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/profile"
	"github.com/lucky7xz/drako/internal/settings"
)

// boolPtr is a tiny helper for the optional execution flags.
func boolPtr(b bool) *bool { return &b }

// explainModel builds a grid model sitting on one command cell, with the
// explain key bound, so the explain path can be driven directly.
func explainModel(cmd profile.Command) Model {
	return Model{
		mode:    gridMode,
		gridNav: gridNav{grid: [][]string{{cmd.Name}}},
		Config: settings.Config{
			Keys:     settings.InputConfig{Explain: "e"},
			Commands: []profile.Command{cmd},
		},
	}
}

// Characterization: pins the info-popup content produced by the grid
// explain path, including the *bool default resolution that the extraction
// must preserve.
func TestExplainDetail_Characterization(t *testing.T) {
	cases := []struct {
		name          string
		cmd           profile.Command
		wantValue     string
		wantAutoClose string
	}{
		{
			name:          "defaults: nil flags mean auto-close",
			cmd:           profile.Command{Name: "Build", Command: "make", Description: "d"},
			wantValue:     "make",
			wantAutoClose: "true",
		},
		{
			name:          "debug flag set",
			cmd:           profile.Command{Name: "Build", Command: "make", DebugExecution: boolPtr(true)},
			wantValue:     "make",
			wantAutoClose: "true",
		},
		{
			name:          "auto-close disabled",
			cmd:           profile.Command{Name: "Build", Command: "make", AutoCloseExecution: boolPtr(false)},
			wantValue:     "make",
			wantAutoClose: "false",
		},
		{
			name:          "empty command shows the folder hint",
			cmd:           profile.Command{Name: "Tools", Command: ""},
			wantValue:     "Error: no command. ( This might be a folder of commands!)",
			wantAutoClose: "true",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := explainModel(tc.cmd)
			tm, _ := m.updateGridMode(keyRune("e"))
			got := tm.(Model)

			if got.mode != infoMode {
				t.Fatalf("mode = %v, want infoMode", got.mode)
			}
			if got.activeDetail == nil {
				t.Fatal("activeDetail = nil")
			}
			d := got.activeDetail
			if d.Value != tc.wantValue {
				t.Errorf("Value = %q, want %q", d.Value, tc.wantValue)
			}
			meta := map[string]string{}
			for _, mt := range d.Meta {
				meta[mt.Label] = mt.Value
			}
			if v, ok := meta["Exec"]; ok {
				t.Errorf("Exec row %q shown; it was dropped as noise", v)
			}
			if meta["Auto-close"] != tc.wantAutoClose {
				t.Errorf("Auto-close = %q, want %q", meta["Auto-close"], tc.wantAutoClose)
			}
		})
	}
}

// addressModel shows profile core's own grid: A0 a cell, B0 a two-item folder.
func addressModel() Model {
	cmds := []profile.Command{
		{Name: "Update", Command: "apt upgrade", Col: "a", Row: 0},
		{Name: "Tools", Col: "b", Row: 0, Items: []profile.CommandItem{
			{Name: "One", Command: "echo 1"},
			{Name: "Two", Command: "echo 2"},
		}},
	}
	cfg := settings.Config{X: 2, Y: 1, Keys: settings.InputConfig{Explain: "e"}, Commands: profile.CopyCommands(cmds)}
	return Model{
		mode:    gridMode,
		Config:  cfg,
		gridNav: gridNav{grid: profile.BuildGrid(cfg.X, cfg.Y, cfg.Commands)},
		profile: profileState{profiles: []config.ProfileInfo{
			{Name: "core", Profile: profile.File{X: 2, Y: 1, Commands: cmds}},
		}},
	}
}

func metaValue(t *testing.T, d *DetailState, label string) (string, bool) {
	t.Helper()
	if d == nil {
		t.Fatal("no popup opened")
	}
	for _, mt := range d.Meta {
		if mt.Label == label {
			return mt.Value, true
		}
	}
	return "", false
}

func TestExplainShowsRunAddress(t *testing.T) {
	tm, _ := addressModel().updateGridMode(keyRune("e"))
	d := tm.(Model).activeDetail
	if want := (DetailMeta{Label: "Address", Value: "core:A0"}); d == nil || len(d.Meta) == 0 || d.Meta[0] != want {
		t.Fatalf("first meta row: got %+v, want %+v", d, want)
	}

	m := addressModel()
	m.mode = dropdownMode
	m.dropdown = dropdownState{row: 0, col: 1, items: m.Config.Commands[1].Items, selectedIdx: 1}
	tm, _ = m.updateDropdownMode(keyRune("e"))
	if got, _ := metaValue(t, tm.(Model).activeDetail, "Address"); got != "core:B0.2" {
		t.Errorf("dropdown item address = %q, want core:B0.2", got)
	}
}

// An address drako run would resolve to some other command must not be shown.
func TestExplainHidesAddressItCannotReach(t *testing.T) {
	cases := map[string]func(*Model){
		"no profiles loaded": func(m *Model) { m.profile.profiles = nil },
		"rescue grid over loaded profiles": func(m *Model) {
			rescue := settings.RescueConfig("")
			m.Config.Commands = rescue.Commands
			m.gridNav.grid = profile.BuildGrid(rescue.X, rescue.Y, rescue.Commands)
		},
		"another command holds that address": func(m *Model) {
			m.Config.Commands[0].Command = "something else"
		},
	}
	for name, mutate := range cases {
		m := addressModel()
		mutate(&m)
		tm, _ := m.updateGridMode(keyRune("e"))
		if v, ok := metaValue(t, tm.(Model).activeDetail, "Address"); ok {
			t.Errorf("%s: Address row %q shown", name, v)
		}
	}
}
