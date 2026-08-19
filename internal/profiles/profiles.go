// Package profiles owns drako's profile/spec file naming convention and the
// parse-free listing of those files on disk.
//
// It deliberately does no TOML parsing: the file-moving commands (stash,
// apply, strip, purge) must see every matching file, including profiles that
// fail to parse. Parsing and validation is the heavier job of
// config.DiscoverProfiles.
package profiles

import (
	"os"
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

// List returns every *.profile.toml file in dir, sorted by Name. A missing
// dir yields (nil, nil); any other read error is returned.
func List(dir string) ([]Entry, error) {
	return listSuffix(dir, ProfileSuffix, NormalizeName)
}

// ListSpecs returns every *.spec.toml file in dir, sorted by Name. Specs are
// not normalized, so Norm == Name. A missing dir yields (nil, nil).
func ListSpecs(dir string) ([]Entry, error) {
	return listSuffix(dir, SpecSuffix, func(s string) string { return s })
}

// listSuffix is the shared scan: read dir, keep non-dir files ending in
// suffix, trim the suffix for Name, derive Norm via norm, sort by Name.
func listSuffix(dir, suffix string, norm func(string) string) ([]Entry, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var entries []Entry
	for _, de := range dirEntries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), suffix) {
			continue
		}
		name := strings.TrimSuffix(de.Name(), suffix)
		entries = append(entries, Entry{File: de.Name(), Name: name, Norm: norm(name)})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries, nil
}
