package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)



func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		checkNetworkStatus(),
		m.spinner.Tick,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		return m, nil

	case pathChangedMsg:
		m.updatePathComponents()
		m.listChildDirs()
		return m, nil

	case reloadProfilesMsg:
		bundle := loadConfig(nil)
		m.applyBundle(bundle)
		m.mode = gridMode
		return m, nil

	case inventoryErrorMsg:
		m.inventory.err = msg.err
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		key := msg.String()
		log.Printf("Key pressed: %q", key)
		if key == "r" {
			cmd := m.toggleProfileLock()
			return m, cmd
		}
		// Profile switching with configurable modifier + Number or ~ (Shift + `)
		if m.mode == gridMode || m.mode == childMode {
			prefix := m.config.NumbModifier + "+"
			if strings.HasPrefix(key, prefix) && len(key) > len(prefix) {
				numberChar := key[len(key)-1]
				if numberChar >= '1' && numberChar <= '9' {
					target := int(numberChar - '1') // '1' -> 0, '2' -> 1, etc.
					if target < len(m.profiles) {
						if updated, cmd, ok := m.switchToProfileIndex(target); ok {
							m = updated
							return m, cmd
						}
					}
					return m, nil
				}
			}
			if key == "`" {// might use ~ too with shift
				return m.handleProfileCycle()
			}
		}
		switch m.mode {
		case gridMode:
			return m.updateGridMode(msg)
		case pathMode:
			return m.updatePathMode(msg)
		case childMode:
			return m.updateChildMode(msg)
		case inventoryMode:
			return m.updateInventoryMode(msg)
		case dropdownMode:
			return m.updateDropdownMode(msg)
		case infoMode:
			return m.updateInfoMode(msg)
		}

	case networkStatusMsg:
		if msg.err != nil {
			m.traffic = themeNameStyle.Render("error")
		} else {
			if msg.online {
				m.onlineStatus = onlineStyle.Render("online")
			} else {
				m.onlineStatus = offlineStyle.Render("offline")
			}

			now := msg.t
			currentSent := msg.counters.BytesSent
			currentRecv := msg.counters.BytesRecv

			m.sentHistory = append(m.sentHistory, currentSent)
			m.recvHistory = append(m.recvHistory, currentRecv)
			m.timeHistory = append(m.timeHistory, now)

			cutoff := now.Add(-time.Duration(m.trafficAvgSeconds * float64(time.Second)))
			firstValidIndex := 0
			for i, t := range m.timeHistory {
				if !t.Before(cutoff) {
					break
				}
				firstValidIndex = i + 1
			}
			if firstValidIndex > 0 && len(m.timeHistory) > firstValidIndex {
				m.timeHistory = m.timeHistory[firstValidIndex:]
				m.sentHistory = m.sentHistory[firstValidIndex:]
				m.recvHistory = m.recvHistory[firstValidIndex:]
			}

			if len(m.timeHistory) > 1 {
				duration := m.timeHistory[len(m.timeHistory)-1].Sub(m.timeHistory[0]).Seconds()
				sentDelta := m.sentHistory[len(m.sentHistory)-1] - m.sentHistory[0]
				recvDelta := m.recvHistory[len(m.recvHistory)-1] - m.recvHistory[0]

				if duration > 0 {
					sentBps := float64(sentDelta) / duration
					recvBps := float64(recvDelta) / duration
					m.traffic = themeNameStyle.Render(fmt.Sprintf("↓ %s ↑ %s", formatTraffic(recvBps), formatTraffic(sentBps)))
				} else {
					m.traffic = themeNameStyle.Render("---")
				}
			} else {
				m.traffic = themeNameStyle.Render("calculating...")
			}
		}
		return m, networkTick()



	case navTimeoutMsg:
		if m.navigationTimer != nil {
			m.navigationTimer.Stop()
		}
		m.navigationTimer = nil
		return m, nil

	case profileStatusClearMsg:
		if msg.id != m.statusClearTimerID {
			return m, nil
		}
		m.statusClearTimerID = 0
		m.profileStatusMessage = ""
		return m, nil

	}

	return m, nil
}

func (m model) updateGridMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle number-based navigation (1-9)
	if num, err := strconv.Atoi(key); err == nil && num >= 1 && num <= 9 {
		targetIndex := num - 1 // Convert to 0-based index

		if m.navigationTimer == nil { // This is the first number press (column selection)
			lastCol := findLastPopulatedCol(m.grid)
			targetCol := min(targetIndex, lastCol)

			// Ensure the target column is valid before proceeding
			if targetCol < 0 {
				return m, nil
			}

			targetRow := findFirstPopulatedRow(m.grid, targetCol)

			m.cursorCol = targetCol
			m.cursorRow = targetRow

			m.navigationTimer = time.NewTimer(500 * time.Millisecond)

			return m, func() tea.Msg {
				<-m.navigationTimer.C
				return navTimeoutMsg{}
			}

		} else { // This is the second number press (row selection)
			m.navigationTimer.Stop()
			m.navigationTimer = nil

			lastRow := findLastPopulatedRow(m.grid, m.cursorCol)
			targetRow := min(targetIndex, lastRow)

			m.cursorRow = targetRow
			return m, nil
		}
	}

	// If a navigation sequence was in progress, any non-numeric key cancels it.
	if m.navigationTimer != nil {
		m.navigationTimer.Stop()
		m.navigationTimer = nil
	}

	switch key {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit
	case "i":
		m.mode = inventoryMode
		m.inventory = initInventoryModel(m.configDir)
		return m, nil
	case "up", "k", "w":
		m.moveCursor(-1, 0)
	case "down", "j", "s":
		m.moveCursor(1, 0)
	case "left", "h", "a":
		m.moveCursor(0, -1)
	case "right", "l", "d":
		m.moveCursor(0, 1)
	case "tab":
		m.mode = pathMode
	case "e":
		selectedChoice := m.grid[m.cursorRow][m.cursorCol]
		if strings.TrimSpace(selectedChoice) == "" {
			return m, nil
		}
		for _, cmd := range m.config.Commands {
			if cmd.Name == selectedChoice {
				m.previousMode = m.mode
				m.infoTitle = selectedChoice
				m.infoDescription = cmd.Description
				// Resolve execution mode and auto-close
				autoClose := true
				if cmd.AutoCloseExecution != nil { autoClose = *cmd.AutoCloseExecution }
				debug := false
				if cmd.DebugExecution != nil { debug = *cmd.DebugExecution }
				if debug { m.infoExecMode = "debug" } else { m.infoExecMode = "live" }
				m.infoAutoClose = autoClose
				m.infoCwd = m.currentPath
				if strings.TrimSpace(cmd.Command) == "" {
					m.infoCommand = "Error: no command. ( This might be a folder of commands!)"
				} else {
					m.infoCommand = strings.ReplaceAll(cmd.Command, "{dR4ko_path}", m.config.DR4koPath)
				}
				m.mode = infoMode
				return m, nil
			}
		}
		// Not found in config
		m.previousMode = m.mode
		m.infoTitle = selectedChoice
		m.infoDescription = ""
		m.infoExecMode = ""
		m.infoAutoClose = false
		m.infoCwd = m.currentPath
		m.infoCommand = "Error: command not found"
		m.mode = infoMode
		return m, nil
	case "enter", " ":
		selectedChoice := m.grid[m.cursorRow][m.cursorCol]
		if selectedChoice != "" {
			// Check if this command has dropdown items
			for _, cmd := range m.config.Commands {
				if cmd.Name == selectedChoice {
					if len(cmd.Items) > 0 {
						// Open dropdown menu
						m.mode = dropdownMode
						m.dropdownRow = m.cursorRow
						m.dropdownCol = m.cursorCol
						m.dropdownItems = cmd.Items
						m.dropdownSelectedIdx = 0
						return m, nil
					}
					break
				}
			}
			// Single command, execute normally
			m.selected = selectedChoice
			return m, tea.Quit
		}
	}
	return m, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func findLastPopulatedCol(grid [][]string) int {
	lastCol := -1
	if len(grid) == 0 {
		return lastCol
	}
	for r := 0; r < len(grid); r++ {
		for c := 0; c < len(grid[r]); c++ {
			if grid[r][c] != "" && c > lastCol {
				lastCol = c
			}
		}
	}
	return lastCol
}

func findLastPopulatedRow(grid [][]string, col int) int {
	lastRow := -1
	if len(grid) == 0 || col < 0 {
		return lastRow
	}
	for r := 0; r < len(grid); r++ {
		if col < len(grid[r]) {
			if grid[r][col] != "" {
				lastRow = r
			}
		}
	}
	return lastRow
}

func findFirstPopulatedRow(grid [][]string, col int) int {
	if len(grid) == 0 || col < 0 {
		return 0
	}
	for r := 0; r < len(grid); r++ {
		if col < len(grid[r]) {
			if grid[r][col] != "" {
				return r
			}
		}
	}
	return 0 // Fallback
}

func (m model) updatePathMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit
	case "left", "h", "a":
		if m.selectedPathIndex > 0 {
			m.selectedPathIndex--
			m.listChildDirs()
		}
	case "right", "l", "d":
		if m.selectedPathIndex < len(m.pathComponents)-1 {
			m.selectedPathIndex++
			m.listChildDirs()
		}
	case "down", "j", "s":
		if len(m.childDirs) > 0 {
			m.mode = childMode
			m.selectedChildIndex = 0
		}
	case "tab":
		m.mode = gridMode
	case "enter", " ":
		targetPath := m.buildPathFromComponents(m.selectedPathIndex)
		if err := os.Chdir(targetPath); err == nil {
			m.currentPath, _ = os.Getwd()
			m.mode = gridMode
			return m, func() tea.Msg { return pathChangedMsg{} }
		}
	}
	return m, nil
}

func (m model) updateChildMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit
	case "up", "k", "w":
		if m.selectedChildIndex > 0 {
			m.selectedChildIndex--
		} else {
			m.mode = pathMode
		}
	case "down", "j", "s":
		if m.selectedChildIndex < len(m.childDirs)-1 {
			m.selectedChildIndex++
		}
	case "tab":
		m.mode = gridMode
	case "enter", " ":
		parentPath := m.buildPathFromComponents(m.selectedPathIndex)
		targetPath := filepath.Join(parentPath, m.childDirs[m.selectedChildIndex])
		if err := os.Chdir(targetPath); err == nil {
			m.currentPath, _ = os.Getwd()
			m.mode = gridMode
			return m, func() tea.Msg { return pathChangedMsg{} }
		}
	}
	return m, nil
}

func (m model) updateDropdownMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle number-based navigation (1-9)
	if num, err := strconv.Atoi(key); err == nil && num >= 1 && num <= 9 {
		targetIndex := num - 1 // Convert to 0-based index
		if len(m.dropdownItems) > 0 {
			// If the target is out of bounds, select the last item.
			if targetIndex >= len(m.dropdownItems) {
				m.dropdownSelectedIdx = len(m.dropdownItems) - 1
			} else {
				m.dropdownSelectedIdx = targetIndex
			}
		}
		return m, nil
	}

	switch key {
	case "q", "esc":
		// Close dropdown and return to grid mode
		m.mode = gridMode
		m.dropdownItems = nil
		return m, nil
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "up", "k", "w":
		if m.dropdownSelectedIdx > 0 {
			m.dropdownSelectedIdx--
		}
	case "down", "j", "s":
		if m.dropdownSelectedIdx < len(m.dropdownItems)-1 {
			m.dropdownSelectedIdx++
		}
	case "e":
		if m.dropdownSelectedIdx >= 0 && m.dropdownSelectedIdx < len(m.dropdownItems) {
			item := m.dropdownItems[m.dropdownSelectedIdx]
			parent := ""
			if m.dropdownRow >= 0 && m.dropdownCol >= 0 && m.dropdownRow < len(m.grid) && m.dropdownCol < len(m.grid[0]) {
				parent = m.grid[m.dropdownRow][m.dropdownCol]
			}
			m.previousMode = m.mode
			if strings.TrimSpace(parent) == "" {
				m.infoTitle = item.Name
			} else {
				m.infoTitle = fmt.Sprintf("%s: %s", parent, item.Name)
			}
			m.infoDescription = item.Description
			// Resolve execution mode and auto-close for item
			autoClose := true
			if item.AutoCloseExecution != nil { autoClose = *item.AutoCloseExecution }
			debug := false
			if item.DebugExecution != nil { debug = *item.DebugExecution }
			if debug { m.infoExecMode = "debug" } else { m.infoExecMode = "live" }
			m.infoAutoClose = autoClose
			m.infoCwd = m.currentPath
			if strings.TrimSpace(item.Command) == "" {
				m.infoCommand = "Error: no command configured"
			} else {
				m.infoCommand = strings.ReplaceAll(item.Command, "{dR4ko_path}", m.config.DR4koPath)
			}
			m.mode = infoMode
			return m, nil
		}
		return m, nil
	case "enter", " ":
		// Execute the selected dropdown item
		if m.dropdownSelectedIdx >= 0 && m.dropdownSelectedIdx < len(m.dropdownItems) {
			selectedItem := m.dropdownItems[m.dropdownSelectedIdx]
			m.selected = selectedItem.Name
			// Store the command to execute
			// We need to create a temporary command entry for execution
			return m, tea.Quit
		}
	}
	return m, nil
}

// Copy text to clipboard via OSC52 sequence
func copyToClipboardCmd(s string) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		enc := base64.StdEncoding.EncodeToString([]byte(s))
		fmt.Printf("\033]52;c;%s\a", enc)
		return nil
	}
}

func (m model) updateInfoMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		prev := m.previousMode
		m.mode = prev
		return m, copyToClipboardCmd(m.infoCommand)
	default:
		m.mode = m.previousMode
		return m, nil
	}
}

func (m model) updateInventoryMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	inv := &m.inventory

	if inv.err != nil {
		// Any key dismisses an error
		m.mode = gridMode
		inv.err = nil
		return m, nil
	}

	switch key := msg.String(); key {
	case "q", "esc":
		m.mode = gridMode
		return m, nil
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	// Navigation
	case "up", "k", "w":
		if inv.focusedList > 0 {
			inv.focusedList--
			inv.cursor = 0
		}
	case "down", "j", "s":
		if inv.focusedList < 2 {
			inv.focusedList++
			inv.cursor = 0
		}
	case "left", "h", "a":
		if inv.focusedList < 2 && inv.cursor > 0 {
			inv.cursor--
		}
	case "right", "l", "d":
		if inv.focusedList < 2 {
			list := inv.visible
			if inv.focusedList == 1 {
				list = inv.inventory
			}
			if inv.cursor < len(list)-1 {
				inv.cursor++
			}
		}
	case "tab":
		inv.focusedList = (inv.focusedList + 1) % 3 // 0: visible, 1: inventory, 2: apply
		inv.cursor = 0

	// Lift and Place
	case " ", "enter":
		if inv.focusedList == 2 { // Apply button is focused
			return m, applyInventoryChangesCmd(m.configDir, m.inventory)
		}

		currentList := &inv.visible
		if inv.focusedList == 1 {
			currentList = &inv.inventory
		}

		if inv.heldItem == nil {
			// Pick up
			if len(*currentList) > 0 {
				item := (*currentList)[inv.cursor]
				inv.heldItem = &item
				*currentList = append((*currentList)[:inv.cursor], (*currentList)[inv.cursor+1:]...)
				// Adjust cursor if it's now out of bounds
				if inv.cursor >= len(*currentList) && len(*currentList) > 0 {
					inv.cursor = len(*currentList) - 1
				}
			}
		} else {
			// Place
			// Prevent placing Default into Inventory list (focusedList==1)
			if inv.focusedList == 1 && inv.heldItem != nil && *inv.heldItem == "Default" {
				inv.status = "Default cannot be moved to Inventory"
				return m, nil
			}
			// Ensure cursor is valid for placement
			if inv.cursor > len(*currentList) {
				inv.cursor = len(*currentList)
			}
			// Insert the held item at the cursor position
			*currentList = append((*currentList)[:inv.cursor], append([]string{*inv.heldItem}, (*currentList)[inv.cursor:]...)...)
			inv.heldItem = nil
		}
	}

	return m, nil
}

func (m model) switchToProfileIndex(target int) (model, tea.Cmd, bool) {
	if len(m.profiles) == 0 {
		return m, nil, false
	}
	total := len(m.profiles)
	if target < 0 || target >= total {
		target = ((target % total) + total) % total
	}

	selected := m.profiles[target]
	norm := strings.TrimSpace(strings.ToLower(selected.Name))

	// Skip missing non-default profiles
	if norm != "default" {
		if !fileExists(selected.Path) {
			log.Printf("skipping missing profile: %s", selected.Path)
			return m, nil, false
		}
	}

	updated := m
	updated.activeProfileIndex = target
	if norm == "default" {
		_ = os.Unsetenv("DRAKO_PROFILE")
		updated.config = m.baseConfig
	} else {
		_ = os.Setenv("DRAKO_PROFILE", selected.Name)
		updated.config = applyProfileOverlay(m.baseConfig, selected.Overlay)
	}
	updated.applyConfig(updated.config)

	return updated, nil, true
}

func (m model) handleProfileCycle() (tea.Model, tea.Cmd) {
	if len(m.profiles) <= 1 {
		return m, nil
	}

	current := m.activeProfileIndex
	total := len(m.profiles)
	for attempts := 0; attempts < total; attempts++ {
		next := (current + 1 + attempts) % total
		nextModel, cmd, ok := m.switchToProfileIndex(next)
		if ok {
			return nextModel, cmd
		}
	}
	return m, nil
}

func (m *model) findFirstNonEmptyRow(col int) int {
	if col < 0 || col >= len(m.grid[0]) {
		return 0
	}
	for r := 0; r < len(m.grid); r++ {
		if m.grid[r][col] != "" {
			return r
		}
	}
	return 0
}

func (m *model) findLastNonEmptyRow(col int) int {
	if col < 0 || col >= len(m.grid[0]) {
		return 0
	}
	for r := len(m.grid) - 1; r >= 0; r-- {
		if m.grid[r][col] != "" {
			return r
		}
	}
	return 0
}

func (m *model) moveCursor(rowDir, colDir int) {
	bestRow, bestCol := -1, -1
	minDist := math.MaxFloat64

	for r, row := range m.grid {
		for c, val := range row {
			if val == "" || (r == m.cursorRow && c == m.cursorCol) {
				continue
			}

			rowDiff := r - m.cursorRow
			colDiff := c - m.cursorCol
			isCorrectDirection := false
			if rowDir > 0 && rowDiff > 0 {
				isCorrectDirection = true
			}
			if rowDir < 0 && rowDiff < 0 {
				isCorrectDirection = true
			}
			if colDir > 0 && colDiff > 0 {
				isCorrectDirection = true
			}
			if colDir < 0 && colDiff < 0 {
				isCorrectDirection = true
			}

			if isCorrectDirection {
				dist := math.Sqrt(math.Pow(float64(rowDiff), 2) + math.Pow(float64(colDiff), 2))
				if dist < minDist {
					minDist = dist
					bestRow, bestCol = r, c
				}
			}
		}
	}

	if bestRow != -1 {
		m.cursorRow = bestRow
		m.cursorCol = bestCol
	}
}