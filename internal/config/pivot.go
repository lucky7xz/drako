package config

import (
	"bytes"
	"errors"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/lucky7xz/drako/internal/paths"
)

// pivotFile persists TUI state that must survive restarts: which profile is
// locked for launching, and the order of equipped profiles.
type pivotFile struct {
	Locked        string   `toml:"locked"`
	EquippedOrder []string `toml:"equipped_order"`
}

// ReadPivotProfile loads the pivot file; a missing file is an empty pivot,
// not an error.
func ReadPivotProfile(configDir string) (pivotFile, error) {
	var pf pivotFile
	path := paths.PivotFile(configDir)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return pivotFile{}, nil
	}
	if err != nil {
		return pivotFile{}, err
	}
	if _, err := toml.Decode(string(data), &pf); err != nil {
		return pivotFile{}, err
	}
	return pf, nil
}

// writePivotFile replaces pivot.toml through a temp file and a rename, so a
// crash mid-write never leaves it truncated.
func writePivotFile(configDir string, pf pivotFile) error {
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(pf); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(configDir, ".pivot-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // gone after the rename; cleans up on failure
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), paths.PivotFile(configDir))
}

// WritePivotLocked records name as the locked profile, preserving the equipped
// order. An unreadable pivot.toml is reported, never replaced.
func WritePivotLocked(configDir, name string) error {
	pf, err := ReadPivotProfile(configDir)
	if err != nil {
		return err
	}
	pf.Locked = strings.TrimSpace(name)
	return writePivotFile(configDir, pf)
}

// WritePivotEquippedOrder records the equipped-profile order, preserving the
// lock. An unreadable pivot.toml is reported, never replaced.
func WritePivotEquippedOrder(configDir string, order []string) error {
	pf, err := ReadPivotProfile(configDir)
	if err != nil {
		return err
	}
	pf.EquippedOrder = order
	return writePivotFile(configDir, pf)
}

// DeletePivotProfile clears the lock, preserving the equipped order. When no
// order remains, the file itself goes; one already gone is fine.
func DeletePivotProfile(configDir string) error {
	pf, err := ReadPivotProfile(configDir)
	if err != nil {
		return err
	}
	if len(pf.EquippedOrder) == 0 {
		if err := os.Remove(paths.PivotFile(configDir)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	pf.Locked = ""
	return writePivotFile(configDir, pf)
}
