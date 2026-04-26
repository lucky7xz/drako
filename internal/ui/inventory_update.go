package ui

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
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
	inventoryDir := filepath.Join(configDir, "inventory")

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

// ApplyInventoryChangesCmd calculates the necessary file moves and executes them.
func ApplyInventoryChangesCmd(configDir string, m inventoryModel) tea.Cmd {
	return func() tea.Msg {

		// Use core logic to calculate moves
		moves, err := m.State.CalculateMoves(configDir, m.initialVisible, m.initialInventory)
		if err != nil {
			return inventoryErrorMsg{err: fmt.Errorf("calc moves failed: %w", err)}
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
		currentVisible, _ := m.State.GetList(core.ListVisible)
		order := make([]string, 0, len(*currentVisible))
		for _, v := range *currentVisible {
			order = append(order, strings.TrimSuffix(v, ".profile.toml"))
		}
		if err := config.WritePivotEquippedOrder(configDir, order); err != nil {
			log.Printf("could not write equipped order: %v", err)
		}

		// Always reload to reflect order and membership changes
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

func (m Model) resolveInventoryMouseClick(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}
	log.Printf("Mouse click in inventory: X=%d, Y=%d", msg.X, msg.Y)

	if m.inventory.err != nil {
		m.mode = gridMode
		m.inventory.err = nil
		return m, nil
	}

	layout := CalculateLayout(m.termWidth, m.termHeight, m.Config)

	// --- 1. Measure Components ---
	var (
		headerContent  string
		equippedTitle  string
		equippedGrid   string
		inventoryTitle string
		inventoryGrid  string
		applyButton    string
		rescueButton   string
		heldStatus     string
		footerContent  string
	)

	if layout.ShowHeader {
		headerContent = inventoryTitleStyle.Render("Inventory Management") + "\n\n"
	}

	visiblePtr, _ := m.inventory.State.GetList(core.ListVisible)
	visible := *visiblePtr
	inventoryPtr, _ := m.inventory.State.GetList(core.ListInventory)
	inventory := *inventoryPtr

	equippedTitle = listHeaderStyle.Render("Equipped Items") + "\n"
	equippedGrid = m.renderInventoryGrid(visible, 0)
	inventoryTitle = listHeaderStyle.Render("Inventory Items") + "\n"
	inventoryGrid = m.renderInventoryGrid(inventory, 1)

	applyButtonStr := "[ Apply Changes ]"
	if m.inventory.focusedList == 2 {
		applyButton = selectedButtonStyle.Render(applyButtonStr)
	} else {
		applyButton = buttonStyle.Render(applyButtonStr)
	}

	rescueButtonStr := "[ Rescue Mode ]"
	if m.inventory.focusedList == 3 {
		rescueButton = selectedRescueButtonStyle.Render(rescueButtonStr)
	} else {
		rescueButton = rescueButtonStyle.Render(rescueButtonStr)
	}

	heldItemStatus := " "
	if m.inventory.State.HeldItem != nil {
		heldItemStatus = helpStyle.Render("Holding: ") + selectedItemStyle.Render(*m.inventory.State.HeldItem)
	}
	heldStatus = heldItemStatus

	if layout.ShowFooter {
		help := helpStyle.Render("↑/↓/tab: Switch Grid | ←/→: Move | space/enter: Lift/Place | q/esc: Back")
		version := helpStyle.Render(config.AppName + " | " + config.Version)
		footerContent = footerStyle.Render(lipgloss.JoinVertical(lipgloss.Center, help, version))
	}

	// --- 2. Calculate Vertical Layout ---
	hH := lipgloss.Height(headerContent)
	etH := lipgloss.Height(equippedTitle)
	egH := lipgloss.Height(equippedGrid)
	itH := lipgloss.Height(inventoryTitle)
	igH := lipgloss.Height(inventoryGrid)
	abH := lipgloss.Height(applyButton)
	rbH := lipgloss.Height(rescueButton)
	hsH := lipgloss.Height(heldStatus)
	fH := lipgloss.Height(footerContent)

	totalHeight := hH + etH + egH + 2 + itH + igH + 2 + abH + 2 + rbH + 2 + hsH
	if layout.ShowFooter {
		totalHeight += fH
	}

	yOffset := (m.termHeight - totalHeight) / 2

	// --- 3. Hit Detection ---

	currentY := yOffset + hH

	// Check Equipped Grid
	currentY += etH
	if msg.Y >= currentY && msg.Y < currentY+egH {
		// Calculate hit in equipped grid
		return m.resolveInventoryGridClick(msg, visible, 0, currentY)
	}
	currentY += egH + 2 // +2 for \n\n

	// Check Inventory Grid
	currentY += itH
	if msg.Y >= currentY && msg.Y < currentY+igH {
		// Calculate hit in inventory grid
		return m.resolveInventoryGridClick(msg, inventory, 1, currentY)
	}
	currentY += igH + 2 // +2 for \n\n

	// Check Apply Button
	if msg.Y >= currentY && msg.Y < currentY+abH {
		bW := lipgloss.Width(applyButton)
		bXStart := (m.termWidth - bW) / 2
		if msg.X >= bXStart && msg.X < bXStart+bW {
			m.inventory.focusedList = 2
			return m.updateInventoryMode(tea.KeyMsg{Type: tea.KeyEnter})
		}
	}
	currentY += abH + 2

	// Check Rescue Button
	if msg.Y >= currentY && msg.Y < currentY+rbH {
		bW := lipgloss.Width(rescueButton)
		bXStart := (m.termWidth - bW) / 2
		if msg.X >= bXStart && msg.X < bXStart+bW {
			m.inventory.focusedList = 3
			return m.updateInventoryMode(tea.KeyMsg{Type: tea.KeyEnter})
		}
	}

	return m, nil
}

func (m Model) resolveInventoryGridClick(msg tea.MouseMsg, profiles []string, listID int, gridStartY int) (tea.Model, tea.Cmd) {
	if len(profiles) == 0 {
		// Handle empty list placeholder click
		m.inventory.focusedList = listID
		m.inventory.cursor = 0
		return m.updateInventoryMode(tea.KeyMsg{Type: tea.KeySpace})
	}

	// Calculate cell width (all cells in a list have same width because of styling)
	sampleStyle := cellStyle
	if listID == m.inventory.focusedList && m.inventory.cursor == 0 {
		sampleStyle = selectedCellStyle
	}
	cellW := lipgloss.Width(sampleStyle.Render(profiles[0]))
	maxCellsPerLine := m.termWidth / cellW
	if maxCellsPerLine <= 0 {
		maxCellsPerLine = 1
	}

	// Calculate total width of a row to center it
	cellsInFirstRow := len(profiles)
	if cellsInFirstRow > maxCellsPerLine {
		cellsInFirstRow = maxCellsPerLine
	}
	rowW := cellsInFirstRow * cellW
	xOffset := (m.termWidth - rowW) / 2

	localY := msg.Y - gridStartY
	rowIdx := localY // Each cell is 1 line high
	colIdx := (msg.X - xOffset) / cellW

	clickedIdx := rowIdx*maxCellsPerLine + colIdx
	if clickedIdx >= 0 && clickedIdx < len(profiles) && msg.X >= xOffset && msg.X < xOffset+rowW {
		m.inventory.focusedList = listID
		m.inventory.cursor = clickedIdx
		// Simulate a space press to pick up or place
		return m.updateInventoryMode(tea.KeyMsg{Type: tea.KeySpace})
	}

	return m, nil
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
