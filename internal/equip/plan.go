package equip

import (
	"fmt"
	"slices"
)

// Location is one of drako's profile-file locations under the config dir.
type Location int

const (
	Equipped  Location = iota // config dir root: profiles the TUI loads
	Inventory                 // …/inventory: stashed profiles
	Trash                     // …/trash: purged profiles
)

// Result reports what Reconcile changed, by display name (suffix trimmed).
type Result struct {
	Equipped []string // equipped from the inventory
	Stashed  []string // stashed to the inventory
}

// FinalCount reports the equipped total a plan would leave behind. Counting
// the plan's outcome rather than len(desired) keeps names that match no file
// in either location from inflating the total.
func FinalCount(equipped int, moves []Move) int {
	final := equipped
	for _, m := range moves {
		if m.To == Equipped {
			final++
		} else {
			final--
		}
	}
	return final
}

// CheckCap rejects a plan that would push the equipped count past MaxEquipped.
// The rule is monotonic rather than an invariant: someone already over the cap
// (profiles copied in by hand, a config predating the cap, or a confirmed
// over-cap spec) can still reorder and stash — only growth is refused, so the
// overflow drains but never returns.
func CheckCap(equipped int, moves []Move) error {
	if final := FinalCount(equipped, moves); final > MaxEquipped && final > equipped {
		return fmt.Errorf("at most %d profiles equipped (got %d); stash some first", MaxEquipped, final)
	}
	return nil
}

// Files extracts the on-disk filenames from a slice of entries.
func Files(entries []Entry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.File
	}
	return names
}

// Move is one file relocation between locations.
type Move struct {
	File string
	From Location
	To   Location
}

// PlanReconcile computes the moves needed so that exactly the desired profiles
// are equipped. equipped and inventory are the current filenames in each
// location; desired is a list of profile references (any form NormalizeName
// accepts). It performs no I/O. Stashes come first, so a run that stops
// partway never has more decks equipped than at its start or its end.
func PlanReconcile(equipped, inventory, desired []string) []Move {
	want := make(map[string]bool, len(desired))
	for _, d := range desired {
		want[NormalizeName(d)] = true
	}

	var moves []Move
	for _, file := range equipped {
		if !want[NormalizeName(file)] {
			moves = append(moves, Move{File: file, From: Equipped, To: Inventory})
		}
	}
	for _, file := range inventory {
		if want[NormalizeName(file)] {
			moves = append(moves, Move{File: file, From: Inventory, To: Equipped})
		}
	}
	return moves
}

// Place says where an arriving profile called file lands, and under which
// name: over the copy that is already there, equipped or in the inventory
// (keeping that copy's spelling), so there is never a second one; else
// equipped while fewer than MaxEquipped are; else the inventory.
func Place(equipped, stashed []Entry, file string) (Location, string) {
	norm := NormalizeName(file)
	same := func(e Entry) bool { return e.Norm == norm }
	if i := slices.IndexFunc(equipped, same); i >= 0 {
		return Equipped, equipped[i].File
	}
	if i := slices.IndexFunc(stashed, same); i >= 0 {
		return Inventory, stashed[i].File
	}
	if len(equipped) < MaxEquipped {
		return Equipped, file
	}
	return Inventory, file
}
