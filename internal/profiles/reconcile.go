package profiles

import (
	"strings"

	"github.com/lucky7xz/drako/internal/equip"
	"github.com/lucky7xz/drako/internal/paths"
)

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
