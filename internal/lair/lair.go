// Package lair is drako's config directory: every file drako owns is reached
// through a Lair, and a Lair cannot be led outside its directory. A name with
// ".." in it, or a link pointing elsewhere, is refused by the operating
// system (os.Root), not by a check someone has to remember.
package lair

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// Lair is the config directory, opened. Names passed to its methods are
// relative to that directory.
type Lair struct {
	dir  string
	root *os.Root
}

// Open opens dir as drako's config directory. A directory that does not exist
// yet is an error (fs.ErrNotExist): Open never creates anything.
func Open(dir string) (*Lair, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("could not open config dir: %w", err)
	}
	return &Lair{dir: dir, root: root}, nil
}

// Create is Open for a first run: it makes dir when it is missing.
func Create(dir string) (*Lair, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("could not create config dir: %w", err)
	}
	return Open(dir)
}

// Close lets go of the directory. Files already opened through the Lair stay
// usable.
func (l *Lair) Close() error { return l.root.Close() }

// Dir is the directory's path, for messages and for code that has not moved
// behind the Lair yet.
func (l *Lair) Dir() string { return l.dir }

// Destroy deletes the config folder and everything in it. A folder cannot
// remove itself through its own root, so this is the one removal by path. A
// folder that is a link loses only the link; what it points to stays.
func Destroy(dir string) error {
	l, err := Open(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("config directory does not exist: %s", dir)
	}
	if err != nil {
		return err
	}
	l.Close()
	return os.RemoveAll(dir)
}
