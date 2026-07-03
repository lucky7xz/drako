package ui

import (
	"testing"

	"github.com/lucky7xz/drako/internal/config"
)

// boolPtr is a tiny helper for the optional execution flags.
func boolPtr(b bool) *bool { return &b }

// explainModel builds a grid model sitting on one command cell, with the
// explain key bound, so the explain path can be driven directly.
func explainModel(cmd config.Command) Model {
	return Model{
		mode:    gridMode,
		gridNav: gridNav{grid: [][]string{{cmd.Name}}},
		Config: config.Config{
			Keys:     config.InputConfig{Explain: "e"},
			Commands: []config.Command{cmd},
		},
	}
}

// Characterization: pins the info-popup content produced by the grid
// explain path, including the *bool default resolution that the extraction
// must preserve.
func TestExplainDetail_Characterization(t *testing.T) {
	cases := []struct {
		name          string
		cmd           config.Command
		wantValue     string
		wantExec      string
		wantAutoClose string
	}{
		{
			name:          "defaults: nil flags mean live + auto-close",
			cmd:           config.Command{Name: "Build", Command: "make", Description: "d"},
			wantValue:     "make",
			wantExec:      "live",
			wantAutoClose: "true",
		},
		{
			name:          "debug flag set",
			cmd:           config.Command{Name: "Build", Command: "make", DebugExecution: boolPtr(true)},
			wantValue:     "make",
			wantExec:      "debug",
			wantAutoClose: "true",
		},
		{
			name:          "auto-close disabled",
			cmd:           config.Command{Name: "Build", Command: "make", AutoCloseExecution: boolPtr(false)},
			wantValue:     "make",
			wantExec:      "live",
			wantAutoClose: "false",
		},
		{
			name:          "empty command shows the folder hint",
			cmd:           config.Command{Name: "Tools", Command: ""},
			wantValue:     "Error: no command. ( This might be a folder of commands!)",
			wantExec:      "live",
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
			if meta["Exec"] != tc.wantExec {
				t.Errorf("Exec = %q, want %q", meta["Exec"], tc.wantExec)
			}
			if meta["Auto-close"] != tc.wantAutoClose {
				t.Errorf("Auto-close = %q, want %q", meta["Auto-close"], tc.wantAutoClose)
			}
		})
	}
}
