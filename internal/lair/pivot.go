package lair

import (
	"bytes"
	"errors"
	"io/fs"
	"strings"

	"github.com/BurntSushi/toml"
)

// pivotFile persists TUI state that must survive restarts: which profile is
// locked for launching, and the order of equipped profiles.
type pivotFile struct {
	Locked        string   `toml:"locked"`
	EquippedOrder []string `toml:"equipped_order"`
}

// ReadPivotProfile loads the pivot file; a missing file, or a config
// directory that does not exist yet, is an empty pivot, not an error.
func ReadPivotProfile(configDir string) (pivotFile, error) {
	l, err := Open(configDir)
	if errors.Is(err, fs.ErrNotExist) {
		return pivotFile{}, nil
	}
	if err != nil {
		return pivotFile{}, err
	}
	defer l.Close()
	return l.readPivot()
}

// WritePivotLocked records name as the locked profile, preserving the equipped
// order. An unreadable pivot.toml is reported, never replaced.
func WritePivotLocked(configDir, name string) error {
	l, err := Create(configDir)
	if err != nil {
		return err
	}
	defer l.Close()
	return l.lockProfile(name)
}

// WritePivotEquippedOrder records the equipped-profile order, preserving the
// lock. An unreadable pivot.toml is reported, never replaced.
func WritePivotEquippedOrder(configDir string, order []string) error {
	l, err := Create(configDir)
	if err != nil {
		return err
	}
	defer l.Close()
	pf, err := l.readPivot()
	if err != nil {
		return err
	}
	pf.EquippedOrder = order
	return l.writePivot(pf)
}

// DeletePivotProfile clears the lock, preserving the equipped order. When no
// order remains, the file itself goes; one already gone is fine.
func DeletePivotProfile(configDir string) error {
	l, err := Open(configDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer l.Close()
	pf, err := l.readPivot()
	if err != nil {
		return err
	}
	if len(pf.EquippedOrder) == 0 {
		return l.remove(PivotFileName)
	}
	pf.Locked = ""
	return l.writePivot(pf)
}

// readPivot loads pivot.toml; a missing file is an empty pivot.
func (l *Lair) readPivot() (pivotFile, error) {
	var pf pivotFile
	data, err := l.read(PivotFileName)
	if errors.Is(err, fs.ErrNotExist) {
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

// writePivot replaces pivot.toml whole.
func (l *Lair) writePivot(pf pivotFile) error {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(pf); err != nil {
		return err
	}
	return l.replace(PivotFileName, buf.Bytes())
}

// lockProfile records name as the locked profile, preserving the equipped order.
func (l *Lair) lockProfile(name string) error {
	pf, err := l.readPivot()
	if err != nil {
		return err
	}
	pf.Locked = strings.TrimSpace(name)
	return l.writePivot(pf)
}
