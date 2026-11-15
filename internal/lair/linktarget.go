package lair

import (
	"io/fs"
	"os"
	"path/filepath"
)

// What is a link, and to what: asked before drako removes or trashes
// something, so it can say that only the link goes. Nothing here follows a
// link or changes a file.

// LinkTarget is the full path that the file called name in configDir points
// to when name is itself a link, and "" when it is an ordinary file or is
// not there.
func LinkTarget(configDir, name string) string {
	l, err := Open(configDir)
	if err != nil {
		return ""
	}
	defer l.Close()
	target, err := l.root.Readlink(name)
	if err != nil {
		return ""
	}
	return aimed(filepath.Dir(filepath.Join(configDir, name)), target)
}

// DirLinkTarget is the full path configDir points to when the config folder
// is itself a link, and "" when it is an ordinary folder or is not there.
func DirLinkTarget(configDir string) string {
	target, err := os.Readlink(configDir)
	if err != nil {
		return ""
	}
	return aimed(filepath.Dir(configDir), target)
}

// CountLinks is how many links are in configDir, at any depth.
func CountLinks(configDir string) int {
	l, err := Open(configDir)
	if err != nil {
		return 0
	}
	defer l.Close()
	n := 0
	fs.WalkDir(l.root.FS(), ".", func(_ string, d fs.DirEntry, err error) error {
		if err == nil && d.Type()&fs.ModeSymlink != 0 {
			n++
		}
		return nil
	})
	return n
}

// aimed is where a link standing in dir points, as a full path.
func aimed(dir, target string) string {
	if filepath.IsAbs(target) {
		return filepath.Clean(target)
	}
	return filepath.Join(dir, target)
}
