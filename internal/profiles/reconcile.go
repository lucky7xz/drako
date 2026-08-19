package profiles

import (
	"fmt"
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
// The result may not exceed MaxEquipped; see ReconcileOverCap.
func Reconcile(configDir string, desired []string) (Result, error) {
	return reconcile(configDir, desired, true)
}

// ReconcileOverCap is Reconcile without the MaxEquipped ceiling, for callers
// that have taken explicit consent from the user — today only the spec CLI's
// confirmed path. Prefer Reconcile everywhere else.
func ReconcileOverCap(configDir string, desired []string) (Result, error) {
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
	moves := planReconcile(fileNames(equipped), fileNames(inventory), desired)
	return finalCount(len(equipped), moves), nil
}

func listBothLocations(configDir string) (equipped, inventory []Entry, err error) {
	equipped, err = List(configDir)
	if err != nil {
		return nil, nil, err
	}
	inventory, err = List(paths.InventoryDir(configDir))
	if err != nil {
		return nil, nil, err
	}
	return equipped, inventory, nil
}

func reconcile(configDir string, desired []string, enforceCap bool) (Result, error) {
	equipped, inventory, err := listBothLocations(configDir)
	if err != nil {
		return Result{}, err
	}

	moves := planReconcile(fileNames(equipped), fileNames(inventory), desired)
	if enforceCap {
		if err := checkCap(len(equipped), moves); err != nil {
			return Result{}, err
		}
	}

	var res Result
	for _, m := range moves {
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

// finalCount reports the equipped total a plan would leave behind. Counting
// the plan's outcome rather than len(desired) keeps names that match no file
// in either location from inflating the total.
func finalCount(equipped int, moves []move) int {
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

// checkCap rejects a plan that would push the equipped count past MaxEquipped.
// The rule is monotonic rather than an invariant: someone already over the cap
// (profiles copied in by hand, a config predating the cap, or a confirmed
// over-cap spec) can still reorder and stash — only growth is refused, so the
// overflow drains but never returns.
func checkCap(equipped int, moves []move) error {
	if final := finalCount(equipped, moves); final > MaxEquipped && final > equipped {
		return fmt.Errorf("at most %d profiles equipped (got %d); stash some first", MaxEquipped, final)
	}
	return nil
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
