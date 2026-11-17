package lair

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ErrLinkedFolder says a subfolder of the config directory is a link that
// leaves it. Drako follows a linked file when reading, never a linked folder.
var ErrLinkedFolder = errors.New("is a link that leaves drako's folder; drako does not read or write through linked subfolders")

// CheckLayout says whether configDir is laid out the way drako supports. A
// subfolder drako keeps (inventory, specs, assets, trash) may not be a link
// that leaves it; the error names every one that is, and what to do instead.
// A folder that does not exist yet, or cannot be opened, has nothing to check.
func CheckLayout(configDir string) error {
	l, err := Open(configDir)
	if err != nil {
		return nil
	}
	defer l.Close()
	var linked []string
	for _, name := range []string{InventoryDirName, SpecsDirName, assetsDirName, TrashDirName} {
		if l.linksOut(name) {
			linked = append(linked, name)
		}
	}
	switch len(linked) {
	case 0:
		return nil
	case 1:
		return fmt.Errorf("%s is a linked folder, and drako does not support linked folders.\n%s", linked[0], layoutHint)
	default:
		names := strings.Join(linked[:len(linked)-1], ", ") + " and " + linked[len(linked)-1]
		return fmt.Errorf("%s are linked folders, and drako does not support linked folders.\n%s", names, layoutHint)
	}
}

const layoutHint = `Link the whole drako folder or single files instead. See https://chronyx.xyz/docs/drako/first-run/#linking-your-config`

// linksOut reports whether name is a link to a folder outside the directory.
// A link that stays inside is fine: the directory follows it.
func (l *Lair) linksOut(name string) bool {
	if !l.isLink(name) {
		return false
	}
	if _, err := l.root.Stat(name); err == nil {
		return false
	}
	info, err := os.Stat(filepath.Join(l.dir, name))
	return err == nil && info.IsDir()
}

// read returns the content of a file drako loads. It is the one place a path
// may leave the directory: a file that is itself a link pointing elsewhere
// (stow and home-manager make these from a dotfiles repository) is followed,
// because the user put it there to be read. Nothing is ever written this way.
func (l *Lair) read(name string) ([]byte, error) {
	data, err := l.root.ReadFile(name)
	if err != nil && l.isLink(name) {
		return os.ReadFile(filepath.Join(l.dir, name))
	}
	return data, l.explain(name, err)
}

// stat is read's twin: it sees through a linked file the same way, so "is it
// there?" and "what does it hold?" never disagree.
func (l *Lair) stat(name string) (fs.FileInfo, error) {
	info, err := l.root.Stat(name)
	if err != nil && l.isLink(name) {
		return os.Stat(filepath.Join(l.dir, name))
	}
	return info, l.explain(name, err)
}

// Exists reports whether the file called name (e.g.
// "inventory/git.profile.toml") is in configDir. A file the user linked there
// counts, like for read.
func Exists(configDir, name string) bool {
	l, err := Open(configDir)
	if err != nil {
		return false
	}
	defer l.Close()
	_, err = l.stat(name)
	return err == nil
}

// isLink reports whether name itself is a symbolic link. A name that passes
// through a linked folder, or steps out with "..", is not one: the directory
// refuses to look.
func (l *Lair) isLink(name string) bool {
	info, err := l.root.Lstat(name)
	return err == nil && info.Mode()&fs.ModeSymlink != 0
}

// explain turns the failure of an operation on name into ErrLinkedFolder when
// the reason is a linked subfolder on the way, so the user is told what to
// change. Any other error, and no error, pass through.
func (l *Lair) explain(name string, err error) error {
	if err == nil {
		return nil
	}
	first, _, _ := strings.Cut(filepath.ToSlash(name), "/")
	if first == "" || first == "." || !l.isLink(first) {
		return err
	}
	if target, statErr := os.Stat(filepath.Join(l.dir, first)); statErr != nil || !target.IsDir() {
		return err
	}
	return fmt.Errorf("%s %w", first, ErrLinkedFolder)
}

// entries lists the directory called name ("." for the config directory
// itself), sorted by filename.
func (l *Lair) entries(name string) ([]fs.DirEntry, error) {
	list, err := fs.ReadDir(l.root.FS(), name)
	return list, l.explain(name, err)
}

// mkdirAll makes the folder called name, and its parents, inside the directory.
func (l *Lair) mkdirAll(name string) error {
	return l.explain(name, l.root.MkdirAll(name, 0o755))
}

// writeNew creates name holding data, or reports false when anything is there
// already; O_EXCL keeps "never overwrite" true under races and links.
func (l *Lair) writeNew(name string, data []byte) (bool, error) {
	f, err := l.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, l.explain(name, err)
	}
	_, err = f.Write(data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err == nil, err
}

// replace puts data in name; see replaceFrom.
func (l *Lair) replace(name string, data []byte) error {
	return l.replaceFrom(name, bytes.NewReader(data), 0o644)
}

// replaceFrom puts what r holds in name through a temp file next to it and a
// rename, so a crash mid-write never leaves name truncated. A name that was a
// link becomes a plain file; what the link pointed at is not touched. The
// file ends up with exactly the mode perm.
func (l *Lair) replaceFrom(name string, r io.Reader, perm fs.FileMode) error {
	base := filepath.Base(name)
	tmp := filepath.Join(filepath.Dir(name), "."+strings.TrimSuffix(base, filepath.Ext(base))+"-"+rand.Text()+".tmp")
	f, err := l.root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return l.explain(name, err)
	}
	defer l.root.Remove(tmp) // gone after the rename; cleans up on failure
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return err
	}
	if err := f.Chmod(perm); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return l.explain(name, l.root.Rename(tmp, name))
}

// remove deletes name; one already gone is fine.
func (l *Lair) remove(name string) error {
	if err := l.root.Remove(name); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return l.explain(name, err)
	}
	return nil
}

// ReadFile is the content of the file called name in configDir, read the way
// the loader reads it: a file linked into the folder is followed.
func ReadFile(configDir, name string) ([]byte, error) {
	l, err := Open(configDir)
	if err != nil {
		return nil, err
	}
	defer l.Close()
	return l.read(name)
}

// SameFile reports whether the file called name in configDir is src, one file
// on disk: a deck that is a link to the very file being summoned.
func SameFile(configDir, name string, src fs.FileInfo) bool {
	l, err := Open(configDir)
	if err != nil {
		return false
	}
	defer l.Close()
	info, err := l.stat(name)
	return err == nil && os.SameFile(info, src)
}
