package profile

import (
	"strings"
	"testing"
)

// TestPlace pins the one rule for where a cell stands, including "-1 / 'z'
// means last" and every way of standing nowhere.
func TestPlace(t *testing.T) {
	const x, y = 3, 4 // columns a,b,c ; rows 0..3

	cases := []struct {
		name             string
		cmd              Command
		wantRow, wantCol int
		wantErr          string
	}{
		{"plain letter+row", Command{Col: "a", Row: 0}, 0, 0, ""},
		{"last letter", Command{Col: "c", Row: 2}, 2, 2, ""},
		{"'z' resolves to last column", Command{Col: "z", Row: 1}, 1, 2, ""},
		{"row -1 resolves to last row", Command{Col: "a", Row: -1}, 3, 0, ""},
		{"unparseable column", Command{Col: "9", Row: 0}, 0, 0, "invalid col"},
		{"no column", Command{Row: 0}, 0, 0, "no col"},
		{"row below -1", Command{Col: "a", Row: -2}, 0, 0, "invalid row -2"},
		{"column past the grid", Command{Col: "d", Row: 0}, 0, 0, "only 3 wide"},
		{"row past the grid", Command{Col: "a", Row: 4}, 0, 0, "only 4 tall"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row, col, err := place(tc.cmd, x, y)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("want an error holding %q, got %v", tc.wantErr, err)
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

// The grid leaves out a cell that stands nowhere, and never panics on it.
func TestBuildGridSkipsUnplacedCells(t *testing.T) {
	cmds := []Command{
		{Name: "good", Command: "true", Row: 0, Col: "a"},
		{Name: "bad", Command: "true", Row: 0, Col: "!"},
		{Name: "far", Command: "true", Row: 7, Col: "a"},
		{Name: "deep", Command: "true", Row: -2, Col: "b"},
	}
	grid := BuildGrid(2, 1, cmds)
	if grid[0][0] != "good" || grid[0][1] != "" {
		t.Errorf("want only the cell that fits, got %q", grid)
	}
}

// Flaws is the list the loader carries and drako check prints: one home.
func TestFlaws(t *testing.T) {
	p := File{X: 2, Y: 2, Commands: []Command{
		{Name: "same", Command: "a", Row: 0, Col: "a"},
		{Name: "same", Command: "b", Row: 0, Col: "b"},
		{Name: "under", Command: "c", Row: 1, Col: "a"},
		{Name: "over", Command: "d", Row: 1, Col: "a"},
		{Name: "far", Command: "e", Row: 5, Col: "d"},
	}}
	fs := Flaws(p)
	want := []struct{ level, part string }{
		{"warning", `duplicate command name "same"`},
		{"error", `command "over" and command "under" share cell A1`},
		{"error", `command "far" sits in column "d"`},
	}
	if len(fs) != len(want) {
		t.Fatalf("want %d flaws, got %d: %+v", len(want), len(fs), fs)
	}
	for i, w := range want {
		if fs[i].Level != w.level || !strings.Contains(fs[i].Msg, w.part) {
			t.Errorf("flaw %d: want %s holding %q, got %+v", i, w.level, w.part, fs[i])
		}
	}
	if clean := Flaws(File{X: 1, Y: 1, Commands: p.Commands[:1]}); len(clean) != 0 {
		t.Errorf("a clean deck has no flaws, got %+v", clean)
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

func TestParseAddress(t *testing.T) {
	tests := []struct {
		ref         string
		wantProfile string
		wantAddr    string
		wantItem    int // 0 = no item
		wantErr     bool
	}{
		{"A0", "", "A0", 0, false},
		{"b12", "", "B12", 0, false},
		{"work:A1.2", "work", "A1", 2, false},
		{":A0", "", "A0", 0, false},
		{"Work:B2", "Work", "B2", 0, false},
		{"a1.10", "", "A1", 10, false},
		{"a-1", "", "A-1", 0, false},
		{"work:C-1.2", "work", "C-1", 2, false},
		{"A", "", "", 0, true},
		{"1A", "", "", 0, true},
		{"A1.0", "", "", 0, true},
		{"A1.2.3", "", "", 0, true},
		{"work:", "", "", 0, true},
		{"", "", "", 0, true},
		{"AA1", "", "", 0, true},
		{"A1.x", "", "", 0, true},
		{"A-", "", "", 0, true},
		{"A--1", "", "", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			name, addr, item, err := ParseAddress(tt.ref)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseAddress(%q) should fail, got (%q,%q,%d)", tt.ref, name, addr, item)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseAddress(%q) = %v", tt.ref, err)
			}
			if name != tt.wantProfile || addr != tt.wantAddr || item != tt.wantItem {
				t.Errorf("ParseAddress(%q) = (%q,%q,%d), want (%q,%q,%d)",
					tt.ref, name, addr, item, tt.wantProfile, tt.wantAddr, tt.wantItem)
			}
		})
	}
}

// The grid shows the last command placed on a square; ShownAt names that one.
func TestShownAt(t *testing.T) {
	cmds := []Command{
		{Name: "a", Col: "a", Row: 0},
		{Name: "under", Col: "b", Row: 0},
		{Name: "over", Col: "z", Row: 0}, // z: the last column, the same square as b
		{Name: "outside", Col: "d", Row: 5},
	}
	for _, tc := range []struct{ row, col, want int }{{0, 0, 0}, {0, 1, 2}, {1, 0, -1}, {5, 3, -1}, {-1, 0, -1}} {
		if got := ShownAt(cmds, 2, 2, tc.row, tc.col); got != tc.want {
			t.Errorf("ShownAt(row %d, col %d) = %d, want %d", tc.row, tc.col, got, tc.want)
		}
	}
}

// Everything a user can run has a name: a cell or a folder item without one
// is a flaw, an error, and the deck still loads.
func TestFlawsNameless(t *testing.T) {
	p := File{X: 2, Y: 1, Commands: []Command{
		{Name: "  ", Command: "a", Row: 0, Col: "a"},
		{Name: "folder", Row: 0, Col: "b", Items: []CommandItem{
			{Name: "ok", Command: "b"},
			{Name: "", Command: "c"},
		}},
	}}
	fs := Flaws(p)
	want := []string{
		`command #1 has no name`,
		`command "folder", item #2 has no name`,
	}
	if len(fs) != len(want) {
		t.Fatalf("want %d flaws, got %d: %+v", len(want), len(fs), fs)
	}
	for i, w := range want {
		if fs[i].Level != "error" || !strings.Contains(fs[i].Msg, w) {
			t.Errorf("flaw %d: want error holding %q, got %+v", i, w, fs[i])
		}
	}
}

// drako check says it once, as the loader does.
func TestLintNamelessOnce(t *testing.T) {
	src := []byte("x = 1\ny = 1\n[[commands]]\nname = \"\"\ncommand = \"a\"\ncol = \"a\"\nrow = 0\n")
	var hits []Finding
	for _, f := range Lint(src) {
		if strings.Contains(f.Msg, "has no name") {
			hits = append(hits, f)
		}
	}
	if len(hits) != 1 || hits[0].Level != "error" {
		t.Errorf("want one error about the missing name, got %+v", hits)
	}
}
