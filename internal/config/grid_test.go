package config

import (
	"strings"
	"testing"
)

// TestResolveCell pins the shared cell-resolution that ValidateConfig and
// BuildGrid both rely on, including the "-1 / 'z' means last" rule.
func TestResolveCell(t *testing.T) {
	cfg := Config{X: 3, Y: 4} // columns a,b,c ; rows 0..3

	cases := []struct {
		name             string
		cmd              Command
		wantRow, wantCol int
		wantErr          bool
	}{
		{"plain letter+row", Command{Col: "a", Row: 0}, 0, 0, false},
		{"last letter", Command{Col: "c", Row: 2}, 2, 2, false},
		{"'z' resolves to last column", Command{Col: "z", Row: 1}, 1, 2, false},  // X-1
		{"row -1 resolves to last row", Command{Col: "a", Row: -1}, 3, 0, false}, // Y-1
		{"unparseable column errors", Command{Col: "9", Row: 0}, 0, 0, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row, col, err := resolveCell(tc.cmd, cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error for column %q", tc.cmd.Col)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if row != tc.wantRow || col != tc.wantCol {
				t.Errorf("got (row=%d, col=%d), want (row=%d, col=%d)", row, col, tc.wantRow, tc.wantCol)
			}
		})
	}
}

// BuildGrid drops a row below -1 silently, so validation must reject it.
func TestValidateConfigRejectsNegativeRow(t *testing.T) {
	cfg := Config{X: 1, Y: 1, Commands: []Command{{Name: "gone", Command: "true", Row: -2, Col: "a"}}}
	if err := ValidateConfig(cfg); err == nil || !strings.Contains(err.Error(), "invalid row -2") {
		t.Errorf("want an invalid row error, got %v", err)
	}
	cfg.Commands[0].Row = -1
	if err := ValidateConfig(cfg); err != nil {
		t.Errorf("row -1 is the last row, got %v", err)
	}
}

func TestBuildGridSkipsInvalidCell(t *testing.T) {
	cfg := Config{
		X: 2, Y: 1,
		Commands: []Command{
			{Name: "good", Command: "true", Row: 0, Col: "a"},
			{Name: "bad", Command: "true", Row: 0, Col: "!"},
		},
	}
	// Must not kill the process: ValidateConfig already reports this at
	// load time; rendering skips the cell.
	grid := BuildGrid(cfg)
	if grid[0][0] != "good" {
		t.Errorf("valid cell should be placed, got %q", grid[0][0])
	}
	if grid[0][1] != "" {
		t.Errorf("invalid cell should be skipped, got %q", grid[0][1])
	}
}

func TestCellAddresses(t *testing.T) {
	c := Command{Name: "x", Col: "c", Row: 2}
	if got := c.Address(); got != "C2" {
		t.Errorf("Address = %q, want C2", got)
	}
	if got := c.ItemAddress(6); got != "C2.6" {
		t.Errorf("ItemAddress(6) = %q, want C2.6", got)
	}

	cmds := []Command{
		{Name: "shadowed", Col: "a", Row: 0},
		{Name: "last", Col: "A", Row: 0},
		{Name: "other", Col: "b", Row: 1},
	}
	if got := FindCell(cmds, "a0"); got == nil || got.Name != "last" {
		t.Errorf("FindCell(a0) = %+v, want the last claimant, as on the grid", got)
	}
	if got := FindCell(cmds, "B1"); got == nil || got.Name != "other" {
		t.Errorf("FindCell(B1) = %+v, want other", got)
	}
	if got := FindCell(cmds, "C9"); got != nil {
		t.Errorf("FindCell(C9) = %+v, want nil", got)
	}
}
