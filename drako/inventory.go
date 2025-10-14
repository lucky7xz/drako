package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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

	visible, err := newList(configDir)
	if err != nil {
		log.Printf("could not read config directory: %v", err)
		return inventoryModel{err: err}
	}

	inventory, err := newList(inventoryDir)
	if err != nil {
		log.Printf("could not read inventory directory: %v", err)
		return inventoryModel{err: err}
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

		// Find files to move from visible to inventory
		for _, file := range m.initialVisible {
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

		if len(moves) > 0 {
			return reloadProfilesMsg{}
		}

		// If no moves, just exit without reloading
		return tea.KeyMsg{Type: tea.KeyEsc}
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