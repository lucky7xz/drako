package tui

import (
	"fmt"
	"log"
	"path/filepath"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/equip"
	"github.com/lucky7xz/drako/internal/lair"
	"github.com/lucky7xz/drako/internal/settings"
)

// reloadProfilesMsg signals the app to reload the configuration.
type reloadProfilesMsg struct{}

// inventoryErrorMsg signals that an error occurred during inventory operations.
type inventoryErrorMsg struct{ err error }

func (e inventoryErrorMsg) Error() string { return e.err.Error() }

// The four things focusedList can point at, stacked in the order ↑/↓ walk
// them. The first two are core's list IDs, since focusedList is passed straight
// to GetList; the buttons continue the same run of numbers.
const (
	focusApply  = equip.ListInventory + 1
	focusRescue = focusApply + 1
	focusLast   = focusRescue
)

// inventoryModel holds the state for the inventory management TUI.
type inventoryModel struct {
	State *equip.InventoryState

	cursor      int    // Position in the current list
	focusedList int    // one of equip.ListVisible/ListInventory, focusApply, focusRescue
	status      string // Feedback message for the user
	// statusOK marks status as a confirmation rather than a refusal. The line
	// used to carry only rejections, so it was unconditionally red; staging a
	// spec and saving an edit are the two things that succeed there.
	statusOK bool
	err      error // Any error that has occurred

	// pending is the armed delete confirmation, nil when nothing is armed.
	pending *pendingDelete

	// specs is the spec-function picker, nil when it is closed.
	specs *specsOverlay

	// seen is both lists as last read from disk, so refreshInventory can tell
	// a change on disk from a staged move.
	seen [2][]string

	// leaving is set by a first press to leave while moves are staged; the
	// same press again discards them. Any other key takes it back.
	leaving bool
}

// staged says what Apply would change, in a few words: "" when the lists are
// as they are on disk. Lifting and placing, and a spec, only stage; trash and
// edit act on the file at once and never show here.
func (inv inventoryModel) staged() string {
	if inv.State == nil {
		return ""
	}
	moves := 0
	for i, list := range [][]string{inv.State.Visible, inv.State.Inventory} {
		for _, f := range list {
			if !slices.Contains(inv.seen[i], f) {
				moves++
			}
		}
	}
	if inv.State.HeldItem != nil {
		moves++
	}
	switch {
	case moves == 1:
		return "1 move"
	case moves > 1:
		return fmt.Sprintf("%d moves", moves)
	case !slices.Equal(inv.State.Visible, inv.seen[0]):
		return "new order"
	}
	return ""
}

// slotForHeld is the one extra place the cursor has while a deck is held: the
// held deck is drawn at the cursor, between the others, and may go after the
// last of them.
func (inv inventoryModel) slotForHeld() int {
	if inv.State != nil && inv.State.HeldItem != nil {
		return 1
	}
	return 0
}

// keepStaged is asked by every way out of the inventory that drops staged
// moves. The first press stays and says what would be lost; the same press
// again goes.
func (inv *inventoryModel) keepStaged(armed bool, again string) bool {
	staged := inv.staged()
	if staged == "" || armed {
		return false
	}
	inv.leaving = true
	inv.status = staged + " not applied — " + again + " again to discard"
	return true
}

// pendingDelete is a delete waiting on confirmation: the user types the
// profile's name before the file at rel is moved to trash. Trashing is only
// reversible by hand, so the keystroke that arms it must not be the one that
// performs it.
type pendingDelete struct {
	rel   string // path relative to the config dir, for profiles.MoveToTrash
	name  string // display name the user has to type
	link  bool   // the file is a link: only the link goes, and the texts say so
	typed string
}

// InitInventoryModel creates the initial state for the inventory TUI.
func InitInventoryModel(configDir string) inventoryModel {
	if err := lair.MakeInventory(configDir); err != nil {
		log.Printf("could not create inventory directory: %v", err)
		return inventoryModel{err: err}
	}

	equipped, err := lair.List(configDir, equip.Equipped)
	if err != nil {
		log.Printf("could not read config directory: %v", err)
		return inventoryModel{err: err}
	}
	stashed, err := lair.List(configDir, equip.Inventory)
	if err != nil {
		log.Printf("could not read inventory directory: %v", err)
		return inventoryModel{err: err}
	}

	// Equipped decks in the grid's order; an unreadable pivot.toml means none saved.
	pf, _ := lair.ReadPivotProfile(configDir)
	visible := equip.Order(equip.Files(equipped), pf.EquippedOrder)
	inventory := equip.Files(stashed)

	state := equip.NewInventoryState(visible, inventory, equip.MaxEquipped)

	return inventoryModel{State: state, seen: [2][]string{slices.Clone(visible), slices.Clone(inventory)}}
}

// isLockedProfile reports whether name is the profile the pivot lock points
// at. Moving or deleting it would quietly change what drako launches next, so
// both refuse and send the user to the grid to break the lock themselves.
func (m Model) isLockedProfile(name string) bool {
	return m.profile.pivotName != "" &&
		equip.NormalizeName(m.profile.pivotName) == equip.NormalizeName(name)
}

// lockedRefusal is the message both guards show, naming the configured key.
func (m Model) lockedRefusal() string {
	return "Anchored profile — lift the anchor with " + m.Config.Keys.Lock + " in the grid"
}

// selectedFile finds where the highlighted deck really is: staged moves leave
// files in place, so it may differ from its list. rel is relative to configDir.
func (inv inventoryModel) selectedFile(configDir string) (rel, name string, ok bool) {
	if inv.focusedList != equip.ListVisible && inv.focusedList != equip.ListInventory {
		return "", "", false
	}
	listPtr, err := inv.State.GetList(inv.focusedList)
	if err != nil || inv.cursor < 0 || inv.cursor >= len(*listPtr) {
		return "", "", false
	}
	file := (*listPtr)[inv.cursor]
	name = strings.TrimSuffix(file, equip.ProfileSuffix)

	if lair.Exists(configDir, file) {
		return file, name, true
	}
	stashed := filepath.Join(lair.InventoryDir(configDir), file)
	if !lair.Exists(configDir, filepath.Join(lair.InventoryDirName, file)) {
		return "", "", false
	}
	rel, err = filepath.Rel(configDir, stashed)
	if err != nil {
		return "", "", false
	}
	return rel, name, true
}

// ApplyInventoryChangesCmd reconciles disk to the visible list and saves its
// order. The list is copied now; keys keep editing State while the Cmd runs.
func ApplyInventoryChangesCmd(configDir string, m inventoryModel) tea.Cmd {
	desired := make([]string, 0, len(m.State.Visible))
	for _, v := range m.State.Visible {
		desired = append(desired, strings.TrimSuffix(v, equip.ProfileSuffix))
	}
	return func() tea.Msg {
		// Equip/stash files to match the arrangement (core stays protected).
		if _, err := lair.Reconcile(configDir, desired); err != nil {
			return inventoryErrorMsg{err: fmt.Errorf("apply inventory changes failed: %w", err)}
		}

		// Persist the visible order into pivot.toml as equipped_order.
		if err := lair.WritePivotEquippedOrder(configDir, desired); err != nil {
			return inventoryErrorMsg{err: fmt.Errorf("could not save the profile order: pivot.toml: %w", err)}
		}

		return reloadProfilesMsg{}
	}
}

func (m Model) updateInventoryMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	inv := &m.inventory

	if inv.err != nil {
		// Any key dismisses an error
		m.mode = gridMode
		inv.err = nil
		return m, nil
	}

	// An armed delete swallows every key: the name is typed one character at a
	// time, so letters must reach the prompt instead of firing their normal
	// action.
	if inv.pending != nil {
		return m.updatePendingDelete(msg)
	}

	// The spec picker owns every key while it is open — the inventory's own
	// bindings act on a cursor the user can no longer see.
	if inv.specs != nil {
		return m.updateSpecsOverlay(msg)
	}

	// Each keystroke starts from a clean slate, so a rejected action's message
	// reads as feedback on the last key rather than lingering over later ones.
	inv.status, inv.statusOK = "", false
	armed := inv.leaving
	inv.leaving = false

	switch m.Config.Screen(settings.Inventory).Act(msg.String()) {
	case settings.Back:
		if inv.keepStaged(armed, "Esc") {
			return m, nil
		}
		m = m.presentNextBrokenProfile() // Return to next error or grid
		return m, nil
	case settings.Quit:
		m.quitting = true
		return m, tea.Quit

	// Navigation
	case settings.MoveV:
		if IsUp(m.Config.Keys, msg) {
			if inv.focusedList > 0 {
				inv.focusedList--
				inv.cursor = 0
			}
		} else if inv.State.HeldItem != nil && inv.focusedList == equip.ListInventory {
			// A held deck is drawn in a list, where it will land. Below the
			// lists there is nowhere to show it.
			inv.status = "Place the profile first"
		} else if inv.focusedList < focusLast {
			inv.focusedList++
			inv.cursor = 0
		}
	case settings.MoveH:
		if inv.focusedList < focusApply {
			if IsLeft(m.Config.Keys, msg) {
				inv.cursor = max(inv.cursor-1, 0)
			} else if listPtr, _ := inv.State.GetList(inv.focusedList); inv.cursor < len(*listPtr)-1+inv.slotForHeld() {
				inv.cursor++
			}
		}
	// Open the spec picker. Tab used to cycle focus, which is the walk up/down
	// already do — so it was free to take.
	case settings.Specs:
		if inv.State.HeldItem != nil {
			inv.status = "Place the held item before choosing a spec"
			return m, nil
		}
		inv.specs = &specsOverlay{plans: m.buildSpecPlans()}

	// Edit the selected profile file in the user's editor
	case settings.Edit:
		if m.GlassrootMode {
			// Unreachable (glassroot gates inventory itself), kept as
			// defense in depth: an editor is a shell.
			return m, nil
		}
		if inv.State.HeldItem != nil {
			inv.status = "Place the held item before editing"
			return m, nil
		}
		rel, _, ok := inv.selectedFile(m.profile.configDir)
		if !ok {
			return m, nil
		}
		return m, openInEditorCmd(filepath.Join(m.profile.configDir, rel))

	// Arm a delete; the trashing itself waits on the typed name.
	case settings.Trash:
		if m.GlassrootMode {
			// Unreachable (glassroot gates inventory itself), kept as
			// defense in depth: this destroys a file.
			return m, nil
		}
		if inv.State.HeldItem != nil {
			inv.status = "Place the held item before deleting"
			return m, nil
		}
		rel, name, ok := inv.selectedFile(m.profile.configDir)
		if !ok {
			return m, nil
		}
		if m.isLockedProfile(name) {
			inv.status = m.lockedRefusal()
			return m, nil
		}
		inv.pending = &pendingDelete{rel: rel, name: name, link: lair.LinkTarget(m.profile.configDir, rel) != ""}

	// Lift and Place
	case settings.Confirm:
		if inv.focusedList == focusApply {
			if inv.State.HeldItem != nil {
				inv.status = "Place the held item before applying"
				return m, nil
			}
			return m, ApplyInventoryChangesCmd(m.profile.configDir, m.inventory)
		}
		if inv.focusedList == focusRescue {
			if inv.keepStaged(armed, "Enter") {
				return m, nil
			}
			m.mode = gridMode
			m.enterRescue()
			return m, nil
		}

		if inv.State.HeldItem == nil {
			// Refusing the lift is what blocks the move: nothing can be placed
			// that was never picked up. Without this, stashing the locked
			// profile leaves pivot.toml pointing at an unequipped profile,
			// which drops the next launch into the rescue grid.
			if listPtr, err := inv.State.GetList(inv.focusedList); err == nil &&
				inv.cursor >= 0 && inv.cursor < len(*listPtr) &&
				m.isLockedProfile(strings.TrimSuffix((*listPtr)[inv.cursor], equip.ProfileSuffix)) {
				inv.status = m.lockedRefusal()
				return m, nil
			}
			// Pick up
			// The cursor stays: it now stands for the held deck, drawn where
			// it was lifted from, which is where Enter would put it back.
			if err := inv.State.PickUpItem(inv.focusedList, inv.cursor); err != nil {
				inv.status = err.Error()
			}
		} else {
			// Place
			if err := inv.State.PlaceItem(inv.focusedList, inv.cursor); err != nil {
				inv.status = err.Error()
				return m, nil
			}
		}
	}

	return m, nil
}

// updatePendingDelete drives the typed-name confirmation. Cancel is esc alone,
// not the table's Back, which also takes 'q' — while typing a name every letter has
// to stay a literal.
func (m Model) updatePendingDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	inv := &m.inventory

	if m.GlassrootMode {
		// Unreachable (glassroot gates inventory itself, and arming re-checks),
		// kept as defense in depth: this is the step that destroys the file, so
		// it must not be the one place that trusts the gate above it.
		inv.pending = nil
		return m, nil
	}

	pd := *inv.pending

	switch msg.Type {
	case tea.KeyEsc:
		inv.pending = nil
		inv.status = "Cancelled"
		return m, nil

	case tea.KeyCtrlC:
		m.quitting = true
		return m, tea.Quit

	case tea.KeyBackspace:
		if r := []rune(pd.typed); len(r) > 0 {
			pd.typed = string(r[:len(r)-1])
		}
		inv.pending = &pd
		return m, nil

	case tea.KeyEnter:
		inv.pending = nil
		if equip.NormalizeName(pd.typed) != equip.NormalizeName(pd.name) {
			inv.status = "Name doesn't match — nothing deleted"
			return m, nil
		}
		if err := lair.MoveToTrash(m.profile.configDir, pd.rel); err != nil {
			inv.status = "Could not delete: " + err.Error()
			return m, nil
		}
		inv.dropSelected()
		for i := range inv.seen { // drako's own trash is no news to refreshInventory
			inv.seen[i] = slices.DeleteFunc(inv.seen[i], func(f string) bool { return f == filepath.Base(pd.rel) })
		}
		inv.status, inv.statusOK = "Trashed "+pd.name+" → trash/", true
		if pd.link {
			inv.status = "Trashed the link " + pd.name + " → trash/ · the file stays"
		}
		return m, nil

	// A space arrives as KeySpace on unix and KeyRunes on Windows, but both
	// carry the rune, so Runes alone is the portable read.
	case tea.KeyRunes, tea.KeySpace:
		pd.typed += string(msg.Runes)
		inv.pending = &pd
		return m, nil
	}

	return m, nil
}

// refreshInventory rereads the decks on disk while the inventory is open. When
// they changed since the last read, the lists are replaced and any staged
// moves go with them: they were made against a stale picture.
func (m Model) refreshInventory() Model {
	fresh := InitInventoryModel(m.profile.configDir)
	if old := m.inventory.seen; fresh.err == nil && m.inventory.err == nil &&
		slices.Equal(fresh.seen[0], old[0]) && slices.Equal(fresh.seen[1], old[1]) {
		return m
	}
	prev := m.inventory
	m.inventory = fresh
	if fresh.err == nil { // a failed rebuild has no State; the view shows err
		m.inventory.focusedList = prev.focusedList
		m.inventory.cursor = prev.cursor
		if listPtr, err := fresh.State.GetList(prev.focusedList); err == nil && m.inventory.cursor >= len(*listPtr) {
			m.inventory.cursor = max(len(*listPtr)-1, 0)
		}
		m.inventory.status = "Profiles changed on disk — inventory reloaded"
	}
	return m
}

// dropSelected removes the highlighted entry from its staged list after the
// file behind it is gone, keeping the cursor in range. The list is edited in
// place rather than rebuilt from disk so an unapplied arrangement survives.
func (inv *inventoryModel) dropSelected() {
	listPtr, err := inv.State.GetList(inv.focusedList)
	if err != nil || inv.cursor < 0 || inv.cursor >= len(*listPtr) {
		return
	}
	*listPtr = append((*listPtr)[:inv.cursor], (*listPtr)[inv.cursor+1:]...)
	if inv.cursor >= len(*listPtr) && len(*listPtr) > 0 {
		inv.cursor = len(*listPtr) - 1
	}
}
