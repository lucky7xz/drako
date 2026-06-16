package config

import "testing"

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
