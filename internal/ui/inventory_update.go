package ui

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
	"github.com/lucky7xz/drako/internal/paths"
	"github.com/lucky7xz/drako/internal/profiles"
)

// reloadProfilesMsg signals the app to reload the configuration.
type reloadProfilesMsg struct{}

// inventoryErrorMsg signals that an error occurred during inventory operations.
type inventoryErrorMsg struct{ err error }

func (e inventoryErrorMsg) Error() string { return e.err.Error() }

// inventoryModel holds the state for the inventory management TUI.
type inventoryModel struct {
	State *core.InventoryState

	cursor      int    // Position in the current list
	focusedList int    // 0 for visible, 1 for inventory, 2 for apply, 3 for rescue
	status      string // Feedback message for the user
	err         error  // Any error that has occurred

	// pending is the armed delete confirmation, nil when nothing is armed.
	pending *pendingDelete
}

// pendingDelete is a delete waiting on confirmation: the user types the
// profile's name before the file at rel is moved to trash. Trashing is only
// reversible by hand, so the keystroke that arms it must not be the one that
// performs it.
type pendingDelete struct {
	rel   string // path relative to the config dir, for profiles.MoveToTrash
	name  string // display name the user has to type
	typed string
}

// NewList creates a new list of profiles by scanning a directory for .profile.toml files.
func NewList(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), profiles.ProfileSuffix) {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}

// InitInventoryModel creates the initial state for the inventory TUI.
func InitInventoryModel(configDir string) inventoryModel {
	inventoryDir := paths.InventoryDir(configDir)

	if err := os.MkdirAll(inventoryDir, 0755); err != nil {
		log.Printf("could not create inventory directory: %v", err)
		return inventoryModel{err: err}
	}

	visibleFiles, err := NewList(configDir)
	if err != nil {
		log.Printf("could not read config directory: %v", err)
		return inventoryModel{err: err}
	}

	inventory, err := NewList(inventoryDir)
	if err != nil {
		log.Printf("could not read inventory directory: %v", err)
		return inventoryModel{err: err}
	}

	// Sort inventory list alphabetically
	sort.Strings(inventory)

	// Order the visible list by the persisted equipped_order, which uses
	// canonical names (e.g., "core", "nw_pro"); leftovers follow alphabetically.
	var visible []string // contains filenames for overlays
	if pf, err := config.ReadPivotProfile(configDir); err == nil && len(pf.EquippedOrder) > 0 {
		// Canonical name -> filename; entries are consumed as the saved
		// order claims them, leftovers get appended below.
		remaining := make(map[string]string, len(visibleFiles))
		for _, f := range visibleFiles {
			remaining[strings.TrimSuffix(f, profiles.ProfileSuffix)] = f
		}

		for _, n := range pf.EquippedOrder {
			if f, ok := remaining[n]; ok {
				visible = append(visible, f)
				delete(remaining, n)
			}
		}
		// Append any leftovers alphabetically by name
		var restNames []string
		for n := range remaining {
			restNames = append(restNames, n)
		}
		sort.Strings(restNames)
		for _, n := range restNames {
			visible = append(visible, remaining[n])
		}
	} else {
		// No saved order: files alphabetically
		sort.Strings(visibleFiles)
		visible = append(visible, visibleFiles...)
	}

	state := core.NewInventoryState(visible, inventory, profiles.MaxEquipped)

	return inventoryModel{State: state}
}

// selectedFilePath resolves the highlighted list item to its current on-disk
// location. Staged (unapplied) moves don't matter: disk is unchanged until
// Apply Changes runs. ok is false when a button row or no item is selected.
func (inv inventoryModel) selectedFilePath(configDir string) (string, bool) {
	if inv.focusedList != core.ListVisible && inv.focusedList != core.ListInventory {
		return "", false
	}
	listPtr, err := inv.State.GetList(inv.focusedList)
	if err != nil || inv.cursor < 0 || inv.cursor >= len(*listPtr) {
		return "", false
	}
	name := (*listPtr)[inv.cursor]
	if inv.focusedList == core.ListVisible {
		return filepath.Join(configDir, name), true
	}
	return filepath.Join(paths.InventoryDir(configDir), name), true
}

// isLockedProfile reports whether name is the profile the pivot lock points
// at. Moving or deleting it would quietly change what drako launches next, so
// both refuse and send the user to the grid to break the lock themselves.
func (m Model) isLockedProfile(name string) bool {
	return m.profile.pivotName != "" &&
		profiles.NormalizeName(m.profile.pivotName) == profiles.NormalizeName(name)
}

// lockedRefusal is the message both guards show, naming the configured key.
func (m Model) lockedRefusal() string {
	return "Locked profile — unlock it with " + m.Config.Keys.Lock + " in the grid"
}

// trashTarget resolves the highlighted item to where it actually sits on disk,
// which can differ from the list it is currently shown in: lift-and-place is
// staged until Apply runs, so an item dragged out of the inventory is still an
// inventory file. Returns the path relative to configDir, as MoveToTrash wants
// it, plus the display name for the confirmation prompt.
func (inv inventoryModel) trashTarget(configDir string) (rel, name string, ok bool) {
	if inv.focusedList != core.ListVisible && inv.focusedList != core.ListInventory {
		return "", "", false
	}
	listPtr, err := inv.State.GetList(inv.focusedList)
	if err != nil || inv.cursor < 0 || inv.cursor >= len(*listPtr) {
		return "", "", false
	}
	file := (*listPtr)[inv.cursor]
	name = strings.TrimSuffix(file, profiles.ProfileSuffix)

	if _, err := os.Stat(filepath.Join(configDir, file)); err == nil {
		return file, name, true
	}
	stashed := filepath.Join(paths.InventoryDir(configDir), file)
	if _, err := os.Stat(stashed); err != nil {
		return "", "", false
	}
	rel, err = filepath.Rel(configDir, stashed)
	if err != nil {
		return "", "", false
	}
	return rel, name, true
}

// ApplyInventoryChangesCmd reconciles the on-disk profiles to match the
// arranged visible list, then persists that order.
func ApplyInventoryChangesCmd(configDir string, m inventoryModel) tea.Cmd {
	return func() tea.Msg {
		currentVisible, _ := m.State.GetList(core.ListVisible)
		desired := make([]string, 0, len(*currentVisible))
		for _, v := range *currentVisible {
			desired = append(desired, strings.TrimSuffix(v, profiles.ProfileSuffix))
		}

		// Equip/stash files to match the arrangement (core stays protected).
		if _, err := profiles.Reconcile(configDir, desired); err != nil {
			return inventoryErrorMsg{err: fmt.Errorf("apply inventory changes failed: %w", err)}
		}

		// Persist the visible order into pivot.toml as equipped_order.
		if err := config.WritePivotEquippedOrder(configDir, desired); err != nil {
			log.Printf("could not write equipped order: %v", err)
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

	// Each keystroke starts from a clean slate, so a rejected action's message
	// reads as feedback on the last key rather than lingering over later ones.
	inv.status = ""

	switch {
	case IsCancel(m.Config.Keys, msg):
		m = m.presentNextBrokenProfile() // Return to next error or grid
		return m, nil
	case Matches(m.Config.Keys, msg, "ctrl+c"):
		m.Quitting = true
		return m, tea.Quit

	// Navigation
	case IsUp(m.Config.Keys, msg):
		if inv.focusedList > 0 {
			inv.focusedList--
			inv.cursor = 0
		}
	case IsDown(m.Config.Keys, msg):
		if inv.focusedList < 3 {
			inv.focusedList++
			inv.cursor = 0
		}
	case IsLeft(m.Config.Keys, msg):
		if inv.focusedList < 2 && inv.cursor > 0 {
			inv.cursor--
		}
	case IsRight(m.Config.Keys, msg):
		if inv.focusedList < 2 {
			listPtr, _ := inv.State.GetList(inv.focusedList)
			list := *listPtr
			if inv.cursor < len(list)-1 {
				inv.cursor++
			}
		}
	case IsPathGridMode(m.Config.Keys, msg): // Reuse tab for focus cycle
		inv.focusedList = (inv.focusedList + 1) % 4 // 0: visible, 1: inventory, 2: apply, 3: rescue
		inv.cursor = 0

	// Edit the selected profile file in the user's editor
	case IsEditFile(m.Config.Keys, msg):
		if m.GlassrootMode {
			// Unreachable (glassroot gates inventory itself), kept as
			// defense in depth: an editor is a shell.
			return m, nil
		}
		if inv.State.HeldItem != nil {
			inv.status = "Place the held item before editing"
			return m, nil
		}
		path, ok := inv.selectedFilePath(m.profile.configDir)
		if !ok {
			return m, nil
		}
		if _, err := os.Stat(path); err != nil {
			inv.status = "Cannot edit: " + err.Error()
			return m, nil
		}
		return m, openInEditorCmd(path)

	// Arm a delete; the trashing itself waits on the typed name.
	case IsDelete(m.Config.Keys, msg):
		if m.GlassrootMode {
			// Unreachable (glassroot gates inventory itself), kept as
			// defense in depth: this destroys a file.
			return m, nil
		}
		if inv.State.HeldItem != nil {
			inv.status = "Place the held item before deleting"
			return m, nil
		}
		rel, name, ok := inv.trashTarget(m.profile.configDir)
		if !ok {
			return m, nil
		}
		if m.isLockedProfile(name) {
			inv.status = m.lockedRefusal()
			return m, nil
		}
		inv.pending = &pendingDelete{rel: rel, name: name}

	// Lift and Place
	case IsConfirm(m.Config.Keys, msg):
		if inv.focusedList == 2 { // Apply button is focused
			return m, ApplyInventoryChangesCmd(m.profile.configDir, m.inventory)
		}
		if inv.focusedList == 3 { // Rescue Mode button
			m.mode = gridMode
			rescueCfg := config.RescueConfig()
			rescueCfg.ApplyDefaults()
			m.applyConfig(rescueCfg)
			return m, nil
		}

		if inv.State.HeldItem == nil {
			// Refusing the lift is what blocks the move: nothing can be placed
			// that was never picked up. Without this, stashing the locked
			// profile leaves pivot.toml pointing at an unequipped profile,
			// which drops the next launch into the rescue grid.
			if listPtr, err := inv.State.GetList(inv.focusedList); err == nil &&
				inv.cursor >= 0 && inv.cursor < len(*listPtr) &&
				m.isLockedProfile(strings.TrimSuffix((*listPtr)[inv.cursor], profiles.ProfileSuffix)) {
				inv.status = m.lockedRefusal()
				return m, nil
			}
			// Pick up
			if err := inv.State.PickUpItem(inv.focusedList, inv.cursor); err != nil {
				inv.status = err.Error()
			} else {
				// Adjust cursor if it's now out of bounds
				listPtr, _ := inv.State.GetList(inv.focusedList)
				if inv.cursor >= len(*listPtr) && len(*listPtr) > 0 {
					inv.cursor = len(*listPtr) - 1
				}
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
// not IsCancel, which also matches 'q' — while typing a name every letter has
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
		m.Quitting = true
		return m, tea.Quit

	case tea.KeyBackspace:
		if r := []rune(pd.typed); len(r) > 0 {
			pd.typed = string(r[:len(r)-1])
		}
		inv.pending = &pd
		return m, nil

	case tea.KeyEnter:
		inv.pending = nil
		if profiles.NormalizeName(pd.typed) != profiles.NormalizeName(pd.name) {
			inv.status = "Name doesn't match — nothing deleted"
			return m, nil
		}
		if err := profiles.MoveToTrash(m.profile.configDir, pd.rel); err != nil {
			inv.status = "Could not delete: " + err.Error()
			return m, nil
		}
		inv.dropSelected()
		inv.status = "Trashed " + pd.name + " → trash/"
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
