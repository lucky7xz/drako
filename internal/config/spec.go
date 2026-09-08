package config

import (
	"fmt"
	"path/filepath"

	"github.com/BurntSushi/toml"
	profilepkg "github.com/lucky7xz/drako/internal/profiles"
)

// specFile is the on-disk shape of a *.spec.toml: a named set of profiles to
// equip. Unexported because nothing outside this file needs the struct — the
// two functions below return the profile list itself.
type specFile struct {
	Profiles []string `toml:"profiles"`
}

// SpecInfo is one spec found on disk. Err is set when the file does not parse,
// in which case Profiles is empty; the entry is still returned so callers can
// show the spec and say why it is unusable.
type SpecInfo struct {
	Name     string // "dev"
	File     string // "dev.spec.toml"
	Profiles []string
	Err      error
}

// LoadSpec reads the profile list out of one spec file. It is the single
// decoder for the format: the CLI commands and the TUI inventory both go
// through here, so "what a spec is" is stated once.
func LoadSpec(path string) ([]string, error) {
	var sf specFile
	if _, err := toml.DecodeFile(path, &sf); err != nil {
		return nil, err
	}
	return sf.Profiles, nil
}

// DiscoverSpecs returns every *.spec.toml in specsDir, sorted by name, with
// each file's parse error carried on its own entry rather than failing the
// scan — one unreadable spec must not hide the rest. A missing directory
// yields no specs and no error.
func DiscoverSpecs(specsDir string) ([]SpecInfo, error) {
	entries, err := profilepkg.ListSpecs(specsDir)
	if err != nil {
		return nil, fmt.Errorf("could not read specs dir: %w", err)
	}

	specs := make([]SpecInfo, 0, len(entries))
	for _, e := range entries {
		info := SpecInfo{Name: e.Name, File: e.File}
		info.Profiles, info.Err = LoadSpec(filepath.Join(specsDir, e.File))
		specs = append(specs, info)
	}
	return specs, nil
}
