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
		{"cell", Selection{Cell: "A"}, "echo A"},
		{"item in its own folder", Selection{Cell: "C", Item: 1}, "echo C1"},
		{"cell sharing an item's name", Selection{Cell: "Same"}, "echo cell"},
		{"item past the folder's end", Selection{Cell: "B", Item: 2}, ""},
		{"no such cell", Selection{Cell: "X"}, ""},
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
