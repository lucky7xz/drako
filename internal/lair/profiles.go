package lair

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lucky7xz/drako/internal/equip"
)

// This file lists and moves profile files. It deliberately does no TOML
// parsing: the file-moving commands (stash, apply, strip, purge) must see
// every matching file, including profiles that fail to parse.

// folder is where loc's profiles sit, as a name inside the config directory.
func folder(loc equip.Location) string {
	switch loc {
	case equip.Inventory:
		return InventoryDirName
	case equip.Trash:
		return TrashDirName
	default:
		return "."
	}
}

// LocationDir is the on-disk directory for loc under configDir, for messages
// and paths shown to the user.
func LocationDir(configDir string, loc equip.Location) string {
	return filepath.Join(configDir, folder(loc))
}

// List returns every *.profile.toml file at loc, sorted by Name. A location
// that does not exist yet yields (nil, nil); any other read error is returned.
func List(configDir string, loc equip.Location) ([]equip.Entry, error) {
	l, err := Open(configDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer l.Close()
	return l.list(folder(loc), equip.ProfileSuffix, equip.NormalizeName)
}

// list is the shared scan: read the folder called name, keep non-dir files
// ending in suffix, trim the suffix for Name, derive Norm via norm, sort by Name.
func (l *Lair) list(name, suffix string, norm func(string) string) ([]equip.Entry, error) {
	dirEntries, err := l.entries(name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var entries []equip.Entry
	for _, de := range dirEntries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), suffix) {
			continue
		}
		base := strings.TrimSuffix(de.Name(), suffix)
		entries = append(entries, equip.Entry{File: de.Name(), Name: base, Norm: norm(base)})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries, nil
}

// Move relocates file (a bare filename) from one location to another under
// configDir.
func Move(configDir, file string, from, to equip.Location) error {
	l, err := Open(configDir)
	if err != nil {
		return err
	}
	defer l.Close()
	return l.move(file, from, to)
}

func (l *Lair) move(file string, from, to equip.Location) error {
	src := filepath.Join(folder(from), file)
	dstDir := folder(to)
	dst := filepath.Join(dstDir, file)
	if _, err := l.root.Lstat(dst); err == nil {
		return fmt.Errorf("destination already exists: %s", filepath.Join(l.dir, dst))
	}
	if err := l.mkdirAll(dstDir); err != nil {
		return err
	}
	// A relative link (stow makes these) resolves from its own folder, so re-aim it.
	if target, err := l.root.Readlink(src); err == nil && !filepath.IsAbs(target) {
		if target, err = filepath.Rel(dstDir, filepath.Join(filepath.Dir(src), target)); err != nil {
			return err
		}
		if err := l.root.Symlink(target, dst); err != nil {
			return l.explain(dst, err)
		}
		return l.explain(src, l.root.Remove(src))
	}
	return l.explain(src, l.root.Rename(src, dst))
}

// MoveToTrash moves file (a path inside configDir, e.g.
// "inventory/git.profile.toml") into the trash folder, under a timestamped
// name so nothing already there is ever overwritten.
func MoveToTrash(configDir, file string) error {
	// Said in drako's own words before the directory refuses it in the system's.
	if rel := filepath.Clean(file); filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path traversal detected: %s", file)
	}
	l, err := Open(configDir)
	if err != nil {
		return err
	}
	defer l.Close()
	return l.moveToTrash(filepath.Clean(file))
}

func (l *Lair) moveToTrash(src string) error {
	if _, err := l.stat(src); err != nil {
		return err
	}

	trash := folder(equip.Trash)
	if err := l.mkdirAll(trash); err != nil {
		return err
	}
	base := filepath.Join(trash, filepath.Base(src)+"."+time.Now().Format("20060102-150405"))
	dst := base
	for n := 2; ; n++ {
		_, err := l.root.Lstat(dst)
		if errors.Is(err, fs.ErrNotExist) {
			break
		}
		if err != nil {
			return l.explain(dst, err)
		}
		dst = fmt.Sprintf("%s-%d", base, n)
	}
	return l.explain(src, l.root.Rename(src, dst))
}

// Reconcile arranges profile files under configDir so that exactly the desired
// profiles are equipped: it equips desired
// profiles from the inventory and stashes equipped profiles not in desired.
// The result may not exceed equip.MaxEquipped; see ReconcileOverCap.
func Reconcile(configDir string, desired []string) (equip.Result, error) {
	return reconcile(configDir, desired, true)
}

// ReconcileOverCap is Reconcile without the equip.MaxEquipped ceiling, for callers
// that have taken explicit consent from the user — today only the spec CLI's
// confirmed path. Prefer Reconcile everywhere else.
func ReconcileOverCap(configDir string, desired []string) (equip.Result, error) {
	return reconcile(configDir, desired, false)
}

// PlannedEquippedCount reports how many profiles would be equipped after
// reconciling configDir to desired. Read-only: it lists both locations and
// plans the moves, but performs none of them, so callers can warn about an
// outcome before committing to it.
func PlannedEquippedCount(configDir string, desired []string) (int, error) {
	equipped, inventory, err := listBothLocations(configDir)
	if err != nil {
		return 0, err
	}
	moves := equip.PlanReconcile(equip.Files(equipped), equip.Files(inventory), desired)
	return equip.FinalCount(len(equipped), moves), nil
}

// Unmatched returns the names in desired that match no profile file in
// either location.
func Unmatched(configDir string, desired []string) ([]string, error) {
	equipped, inventory, err := listBothLocations(configDir)
	if err != nil {
		return nil, err
	}
	have := make(map[string]bool, len(equipped)+len(inventory))
	for _, e := range append(equipped, inventory...) {
		have[e.Norm] = true
	}
	var missing []string
	for _, d := range desired {
		if !have[equip.NormalizeName(d)] {
			missing = append(missing, d)
		}
	}
	return missing, nil
}

func listBothLocations(configDir string) (equipped, inventory []equip.Entry, err error) {
	equipped, err = List(configDir, equip.Equipped)
	if err != nil {
		return nil, nil, err
	}
	inventory, err = List(configDir, equip.Inventory)
	if err != nil {
		return nil, nil, err
	}
	return equipped, inventory, nil
}

func reconcile(configDir string, desired []string, enforceCap bool) (equip.Result, error) {
	equipped, inventory, err := listBothLocations(configDir)
	if err != nil {
		return equip.Result{}, err
	}
	moves := equip.PlanReconcile(equip.Files(equipped), equip.Files(inventory), desired)
	if enforceCap {
		if err := equip.CheckCap(len(equipped), moves); err != nil {
			return equip.Result{}, err
		}
	}

	var res equip.Result
	for _, m := range moves {
		if err := Move(configDir, m.File, m.From, m.To); err != nil {
			return res, err
		}
		name := strings.TrimSuffix(m.File, equip.ProfileSuffix)
		if m.To == equip.Equipped {
			res.Equipped = append(res.Equipped, name)
		} else {
			res.Stashed = append(res.Stashed, name)
		}
	}
	return res, nil
}
