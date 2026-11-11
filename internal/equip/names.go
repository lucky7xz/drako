package equip

import (
	"slices"
	"sort"
	"strings"
)

// File suffixes identifying drako's two TOML file kinds.
const (
	ProfileSuffix = ".profile.toml"
	SpecSuffix    = ".spec.toml"
)

// MaxEquipped mirrors drako's 1-9 idiom: every equipped profile stays
// reachable by one chord (leader 1-9, Alt+1-9).
const MaxEquipped = 9

// Entry is one matching file found on disk. No parsing is done, so invalid
// profiles are still listed.
type Entry struct {
	File string // on-disk filename, e.g. "git.profile.toml"
	Name string // display name, suffix trimmed, e.g. "git"
	Norm string // normalized name for matching, e.g. "git"
}

// NormalizeName lowercases a profile reference and strips the known file
// suffixes, so "Git", "git.profile.toml" and "git.profile" all address the
// same profile.
func NormalizeName(name string) string {
	n := strings.TrimSpace(strings.ToLower(name))
	n = strings.TrimSuffix(n, ".profile.toml")
	n = strings.TrimSuffix(n, ".toml")
	n = strings.TrimSuffix(n, ".profile")
	return n
}

// Order arranges deck names or filenames as the grid shows them: those named in
// order first (matched by NormalizeName), then the rest sorted by deck name.
func Order(items, order []string) []string {
	rest := slices.Clone(items)
	sort.Slice(rest, func(i, j int) bool {
		return strings.TrimSuffix(rest[i], ProfileSuffix) < strings.TrimSuffix(rest[j], ProfileSuffix)
	})
	var ordered []string
	for _, n := range order {
		norm := NormalizeName(n)
		if i := slices.IndexFunc(rest, func(r string) bool { return NormalizeName(r) == norm }); i >= 0 {
			ordered = append(ordered, rest[i])
			rest = slices.Delete(rest, i, i+1)
		}
	}
	return append(ordered, rest...)
}
