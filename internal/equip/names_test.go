package equip

import (
	"slices"
	"testing"
)

func TestNormalizeName(t *testing.T) {
	cases := map[string]string{
		"Git":              "git",
		"git.profile.toml": "git",
		"git.profile":      "git",
		"git.toml":         "git",
		"  WORK  ":         "work",
	}
	for in, want := range cases {
		if got := NormalizeName(in); got != want {
			t.Errorf("NormalizeName(%q) = %q, want %q", in, got, want)
		}
	}
}

// Order is the one ordering rule shared by the grid and the inventory.
func TestOrder(t *testing.T) {
	tests := []struct {
		name         string
		items, order []string
		want         []string
	}{
		{"no order sorts by name", []string{"b", "a"}, nil, []string{"a", "b"}},
		{"filenames sort by deck name", []string{"git-extra.profile.toml", "git.profile.toml"}, nil,
			[]string{"git.profile.toml", "git-extra.profile.toml"}},
		{"saved names match loosely", []string{"alpha.profile.toml", "work.profile.toml"}, []string{"Work", "ghost"},
			[]string{"work.profile.toml", "alpha.profile.toml"}},
	}
	for _, tt := range tests {
		if got := Order(tt.items, tt.order); !slices.Equal(got, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}
