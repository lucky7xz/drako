// Package profiles owns drako's profile/spec file naming convention and the
// parse-free listing of those files on disk.
//
// It deliberately does no TOML parsing: the file-moving commands (stash,
// apply, strip, purge) must see every matching file, including profiles that
// fail to parse. Parsing and validation is the heavier job of
// lair.DiscoverProfiles.
package profiles

import (
	"os"
	"sort"
	"strings"

	"github.com/lucky7xz/drako/internal/equip"
)

// List returns every *.profile.toml file in dir, sorted by Name. A missing
// dir yields (nil, nil); any other read error is returned.
func List(dir string) ([]equip.Entry, error) {
	return listSuffix(dir, equip.ProfileSuffix, equip.NormalizeName)
}

// ListSpecs returns every *.spec.toml file in dir, sorted by Name. Specs are
// not normalized, so Norm == Name. A missing dir yields (nil, nil).
func ListSpecs(dir string) ([]equip.Entry, error) {
	return listSuffix(dir, equip.SpecSuffix, func(s string) string { return s })
}

// listSuffix is the shared scan: read dir, keep non-dir files ending in
// suffix, trim the suffix for Name, derive Norm via norm, sort by Name.
func listSuffix(dir, suffix string, norm func(string) string) ([]equip.Entry, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var entries []equip.Entry
	for _, de := range dirEntries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), suffix) {
			continue
		}
		name := strings.TrimSuffix(de.Name(), suffix)
		entries = append(entries, equip.Entry{File: de.Name(), Name: name, Norm: norm(name)})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries, nil
}
