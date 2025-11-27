package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
)

// reloadProfilesMsg signals the app to reload the configuration.
type reloadProfilesMsg struct{}

// inventoryErrorMsg signals that an error occurred during inventory operations.
type inventoryErrorMsg struct{ err error }

func (e inventoryErrorMsg) Error() string { return e.err.Error() }

// inventoryModel holds the state for the inventory management TUI.
type inventoryModel struct {
	visible     []string // Profiles in the main config dir
	inventory   []string // Profiles in the inventory subdir
	cursor      int      // Position in the current list
	focusedList int      // 0 for visible, 1 for inventory, 2 for apply
	heldItem    *string  // The profile being moved
	status      string   // Feedback message for the user
	err         error    // Any error that has occurred

	// Keep the initial state to calculate the diff on apply
	initialVisible   []string
	initialInventory []string
}

// newList creates a new list of profiles by scanning a directory for .profile.toml files.
func newList(dir string) ([]string, error) {
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

// initInventoryModel creates the initial state for the inventory TUI.
func initInventoryModel(configDir string) inventoryModel {
	inventoryDir := filepath.Join(configDir, "inventory")

	if err := os.MkdirAll(inventoryDir, 0755); err != nil {
		log.Printf("could not create inventory directory: %v", err)
		return inventoryModel{err: err}
	}

	visibleFiles, err := newList(configDir)
	if err != nil {
		log.Printf("could not read config directory: %v", err)
		return inventoryModel{err: err}
	}

	inventory, err := newList(inventoryDir)
	if err != nil {
		log.Printf("could not read inventory directory: %v", err)
		return inventoryModel{err: err}
	}

	// Sort inventory list alphabetically
	sort.Strings(inventory)

	// Build visible list including the special "Default" entry
	// Persisted equipped_order uses canonical names (e.g., "Default", "nw_pro")
	var visible []string // contains filenames for overlays and the literal "Default"
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
		addedDefault := false
		for _, n := range pf.EquippedOrder {
			if n == "Default" {
				visible = append(visible, "Default")
				addedDefault = true
				continue
			}
			if f, ok := remaining[n]; ok {
				visible = append(visible, f)
				delete(remaining, n)
			}
		}
		// Append any leftovers alphabetically by name; if Default wasn't listed, append it at the end
		var restNames []string
		for n := range remaining {
			restNames = append(restNames, n)
		}
		sort.Strings(restNames)
		for _, n := range restNames {
			visible = append(visible, remaining[n])
		}
		if !addedDefault {
			visible = append(visible, "Default")
		}
	} else {
		// No saved order: Default first, then files alphabetically
		sort.Strings(visibleFiles)
		visible = append([]string{"Default"}, visibleFiles...)
	}

	return inventoryModel{
		visible:          visible,
		inventory:        inventory,
		initialVisible:   append([]string{}, visible...),
		initialInventory: append([]string{}, inventory...),
	}
}

// applyInventoryChangesCmd calculates the necessary file moves and executes them.
func applyInventoryChangesCmd(configDir string, m inventoryModel) tea.Cmd {
	return func() tea.Msg {
		inventoryDir := filepath.Join(configDir, "inventory")
		moves := map[string]string{} // from -> to

		// Find files to move from visible to inventory (skip Default)
		for _, file := range m.initialVisible {
			if file == "Default" {
				continue
			}
			if !contains(m.visible, file) {
				moves[filepath.Join(configDir, file)] = filepath.Join(inventoryDir, file)
			}
		}

		// Find files to move from inventory to visible
		for _, file := range m.initialInventory {
			if !contains(m.inventory, file) {
				moves[filepath.Join(inventoryDir, file)] = filepath.Join(configDir, file)
			}
		}

		// Pre-flight check for conflicts
		for _, dest := range moves {
			if _, err := os.Stat(dest); err == nil {
				return inventoryErrorMsg{err: fmt.Errorf("conflict: %s already exists", filepath.Base(dest))}
			}
		}

		// Execute moves
		for src, dest := range moves {
			if err := os.Rename(src, dest); err != nil {
				return inventoryErrorMsg{err: fmt.Errorf("failed to move %s: %w", filepath.Base(src), err)}
			}
		}

		// Persist the current visible order into pivot.toml as equipped_order (canonical names)
		order := make([]string, 0, len(m.visible))
		for _, v := range m.visible {
			if v == "Default" {
				order = append(order, "Default")
				continue
			}
			order = append(order, strings.TrimSuffix(v, ".profile.toml"))
		}
		if err := config.WritePivotEquippedOrder(configDir, order); err != nil {
			log.Printf("could not write equipped order: %v", err)
		}

		// Always reload to reflect order and membership changes
		return reloadProfilesMsg{}
	}
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
