package profiles

import (
	"strings"

	"github.com/lucky7xz/drako/internal/paths"
)

// Result reports what Reconcile changed, by display name (suffix trimmed).
type Result struct {
	Equipped []string // equipped from the inventory
	Stashed  []string // stashed to the inventory
}

// Reconcile arranges profile files under configDir so that exactly the desired
// profiles are equipped: it equips desired
// profiles from the inventory and stashes equipped profiles not in desired.
func Reconcile(configDir string, desired []string) (Result, error) {
	equipped, err := List(configDir)
	if err != nil {
		return Result{}, err
	}
	inventory, err := List(paths.InventoryDir(configDir))
	if err != nil {
		return Result{}, err
	}

	var res Result
	for _, m := range planReconcile(fileNames(equipped), fileNames(inventory), desired) {
		if err := Move(configDir, m.File, m.From, m.To); err != nil {
			return res, err
		}
		name := strings.TrimSuffix(m.File, ProfileSuffix)
		if m.To == Equipped {
			res.Equipped = append(res.Equipped, name)
		} else {
			res.Stashed = append(res.Stashed, name)
		}
	}
	return res, nil
}

// fileNames extracts the on-disk filenames from a slice of entries.
func fileNames(entries []Entry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.File
	}
	return names
}

// move is one file relocation between locations.
type move struct {
	File string
	From Location
	To   Location
}

// planReconcile computes the moves needed so that exactly the desired profiles
// are equipped. equipped and inventory are the current filenames in each
// location; desired is a list of profile references (any form NormalizeName
// accepts). It performs no I/O.
func planReconcile(equipped, inventory, desired []string) []move {
	want := make(map[string]bool, len(desired))
	for _, d := range desired {
		want[NormalizeName(d)] = true
	}

	var moves []move
	for _, file := range inventory {
		if want[NormalizeName(file)] {
			moves = append(moves, move{File: file, From: Inventory, To: Equipped})
		}
	}
	for _, file := range equipped {
		if !want[NormalizeName(file)] {
			moves = append(moves, move{File: file, From: Equipped, To: Inventory})
		}
	}
	return moves
}
