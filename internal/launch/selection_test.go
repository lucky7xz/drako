package launch

import (
	"testing"

	"github.com/lucky7xz/drako/internal/profile"
)

func TestSelectionResolve(t *testing.T) {
	cells := []profile.Command{
		{Name: "A", Command: "echo A"},
		{Name: "B", Items: []profile.CommandItem{{Name: "Same", Command: "echo B1"}}},
		{Name: "C", Items: []profile.CommandItem{{Name: "Same", Command: "echo C1"}}},
		{Name: "Same", Command: "echo cell"},
	}
	tests := []struct {
		name string
		sel  Selection
		want string // the resolved command, "" for no match
	}{
		{"cell", Selection{N: 1}, "echo A"},
		{"item in its own folder", Selection{N: 3, Item: 1}, "echo C1"},
		{"cell sharing an item's name", Selection{N: 4}, "echo cell"},
		{"item past the folder's end", Selection{N: 2, Item: 2}, ""},
		{"no such cell", Selection{N: 9}, ""},
		{"nothing picked", Selection{}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cell, item, ok := tc.sel.Resolve(cells)
			got := ""
			if ok {
				got = cell.Command
				if item != nil {
					got = item.Command
				}
			}
			if got != tc.want {
				t.Errorf("Resolve(%+v) = %q, want %q", tc.sel, got, tc.want)
			}
		})
	}
}

// Two cells may share a name; each is picked by where it stands.
func TestSelectionTellsSameNamedCellsApart(t *testing.T) {
	cells := []profile.Command{{Name: "same", Command: "echo first"}, {Name: "same", Command: "echo second"}}
	for n, want := range map[int]string{1: "echo first", 2: "echo second"} {
		cell, _, ok := Selection{N: n}.Resolve(cells)
		if !ok || cell.Command != want {
			t.Errorf("N=%d: got %+v, want %q", n, cell, want)
		}
	}
}
