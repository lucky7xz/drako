package config

import (
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

func writePivotFile(configDir string, pf pivotFile) error {
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	path := paths.PivotFile(configDir)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(pf)
}

// WritePivotLocked records name as the locked profile, preserving the
// equipped order.
func WritePivotLocked(configDir, name string) error {
	pf, _ := ReadPivotProfile(configDir)
	pf.Locked = strings.TrimSpace(name)
	return writePivotFile(configDir, pf)
}

// WritePivotEquippedOrder records the equipped-profile order, preserving
// the lock.
func WritePivotEquippedOrder(configDir string, order []string) error {
	pf, _ := ReadPivotProfile(configDir)
	pf.EquippedOrder = order
	return writePivotFile(configDir, pf)
}

// DeletePivotProfile clears the lock, preserving the equipped order. When
// neither lock nor order remain, the file itself is removed.
func DeletePivotProfile(configDir string) error {
	pf, _ := ReadPivotProfile(configDir)
	if pf.Locked == "" && len(pf.EquippedOrder) == 0 {
		return os.Remove(paths.PivotFile(configDir))
	}
	pf.Locked = ""
	return writePivotFile(configDir, pf)
}
