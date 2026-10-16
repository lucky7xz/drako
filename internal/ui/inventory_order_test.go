package ui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// The inventory lists equipped decks in the grid's order, or an Apply with no
// changes would save a different one.
func TestInventoryOrderMatchesTheGrid(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		pivot string
		want  []string
	}{
		{"saved names match loosely", []string{"alpha", "work"}, "equipped_order = [\"Work\", \"alpha.profile.toml\"]\n",
			[]string{"work", "alpha"}},
		{"the rest sort by deck name", []string{"git-extra", "git"}, "",
			[]string{"git", "git-extra"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, f+".profile.toml"), []byte("x = 1\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(dir, "pivot.toml"), []byte(tt.pivot), 0o644); err != nil {
				t.Fatal(err)
			}
			var want []string
			for _, n := range tt.want {
				want = append(want, n+".profile.toml")
			}
			if got := InitInventoryModel(dir).State.Visible; !slices.Equal(got, want) {
				t.Errorf("Visible = %v, want %v", got, want)
			}
		})
	}
}
