package lair

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/lucky7xz/drako/internal/equip"
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

// LoadSpec reads the profile list out of the spec file called file (e.g.
// "dev.spec.toml") in configDir's specs folder. It is the single decoder for
// the format: the CLI commands and the TUI inventory both go through here, so
// "what a spec is" is stated once.
func LoadSpec(configDir, file string) ([]string, error) {
	l, err := Open(configDir)
	if err != nil {
		return nil, err
	}
	defer l.Close()
	return l.loadSpec(file)
}

func (l *Lair) loadSpec(file string) ([]string, error) {
	data, err := l.read(filepath.Join(SpecsDirName, file))
	if err != nil {
		return nil, err
	}
	var sf specFile
	meta, err := toml.Decode(string(data), &sf)
	if err != nil {
		return nil, err
	}
	// A typo like "profile" must not read as an empty list: applying that stashes every deck.
	if unknown := undecodedKeys(meta); len(unknown) > 0 {
		return nil, fmt.Errorf("unrecognized key(s): %s (check for a typo)", strings.Join(unknown, ", "))
	}
	return sf.Profiles, nil
}

// DiscoverSpecs returns every *.spec.toml in configDir's specs folder, sorted
// by name, with each file's parse error carried on its own entry rather than
// failing the scan — one unreadable spec must not hide the rest. A missing
// folder yields no specs and no error.
func DiscoverSpecs(configDir string) ([]SpecInfo, error) {
	l, err := Open(configDir)
	if errors.Is(err, fs.ErrNotExist) {
		return []SpecInfo{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read specs dir: %w", err)
	}
	defer l.Close()

	// Specs are not normalized, so Norm == Name.
	entries, err := l.list(SpecsDirName, equip.SpecSuffix, func(s string) string { return s })
	if err != nil {
		return nil, fmt.Errorf("could not read specs dir: %w", err)
	}

	specs := make([]SpecInfo, 0, len(entries))
	for _, e := range entries {
		info := SpecInfo{Name: e.Name, File: e.File}
		info.Profiles, info.Err = l.loadSpec(e.File)
		specs = append(specs, info)
	}
	return specs, nil
}
