package ui

import (
	"fmt"
	"log"
	"os"
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

	// Keep the initial state to calculate the diff on apply
	initialVisible   []string
	initialInventory []string
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

	var profiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".profile.toml") {
			profiles = append(profiles, entry.Name())
		}
	}
	return profiles, nil
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

	// Build visible list including the special "Core" entry
	// Persisted equipped_order uses canonical names (e.g., "Core", "nw_pro")
	var visible []string // contains filenames for overlays
	if pf, err := config.ReadPivotProfile(configDir); err == nil && len(pf.EquippedOrder) > 0 {
		// Map canonical name -> filename
		nameToFile := make(map[string]string, len(visibleFiles))
		for _, f := range visibleFiles {
			name := strings.TrimSuffix(f, ".profile.toml")
			nameToFile[name] = f
		}
		// Track remaining overlays by name
		remaining := make(map[string]string, len(nameToFile))
		for n, f := range nameToFile {
			remaining[n] = f
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

	state := core.NewInventoryState(visible, inventory)

	return inventoryModel{
		State:            state,
		initialVisible:   append([]string{}, visible...),
		initialInventory: append([]string{}, inventory...),
	}
}

// ApplyInventoryChangesCmd reconciles the on-disk profiles to match the
// arranged visible list, then persists that order.
func ApplyInventoryChangesCmd(configDir string, m inventoryModel) tea.Cmd {
	return func() tea.Msg {
		currentVisible, _ := m.State.GetList(core.ListVisible)
		desired := make([]string, 0, len(*currentVisible))
		for _, v := range *currentVisible {
			desired = append(desired, strings.TrimSuffix(v, ".profile.toml"))
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

func Contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func (m Model) updateInventoryMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	inv := &m.inventory

	if inv.err != nil {
		// Any key dismisses an error
		m.mode = gridMode
		inv.err = nil
		return m, nil
	}

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

	// Lift and Place
	case IsConfirm(m.Config.Keys, msg):
		if inv.focusedList == 2 { // Apply button is focused
			return m, ApplyInventoryChangesCmd(m.configDir, m.inventory)
		}
		if inv.focusedList == 3 { // Rescue Mode button
			m.mode = gridMode
			rescueCfg := config.RescueConfig()
			rescueCfg.ApplyDefaults()
			m.applyConfig(rescueCfg)
			return m, nil
		}

		if inv.State.HeldItem == nil {
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
