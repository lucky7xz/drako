package ui

import (
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
)

func (m Model) Init() tea.Cmd {
	configDir, _ := config.GetConfigDir()
	return tea.Batch(
		tea.EnterAltScreen,
		checkNetworkStatusCmd(),
		m.spinner.Tick,
		WatchConfigCmd(configDir),
		lockCheckTick(),
	)
}

func checkNetworkStatusCmd() tea.Cmd {
	return func() tea.Msg {
		status := core.CheckNetworkStatus()
		return networkStatusMsg{online: status.Online, counters: status.Counters, t: status.Time, err: status.Err}
	}
}

func networkTick() tea.Cmd {
	return tea.Tick(2500*time.Millisecond, func(t time.Time) tea.Msg {
		return checkNetworkStatusCmd()()
	})
}

// lockCheckTick creates a command that checks for auto-lock every 30 seconds
func lockCheckTick() tea.Cmd {
	return tea.Tick(30*time.Second, func(time.Time) tea.Msg {
		return lockCheckMsg{}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		return m, nil

	case pathChangedMsg:
		m.path.UpdatePathComponents()
		m.path.ListChildDirs()
		return m, nil

	case reloadProfilesMsg:
		bundle := config.LoadConfig(nil)
		m.applyBundle(bundle)
		if len(bundle.Broken) > 0 {
			m.pendingProfileErrors = append(m.pendingProfileErrors, bundle.Broken...)
			m.profileErrorQueueActive = true
			m = m.presentNextBrokenProfile()
			return m, nil
		}
		m.mode = gridMode
		return m, nil

	case ConfigChangedMsg:
		// Config file changed on disk, reload everything
		log.Printf("Config file change detected: %s", msg.Path)
		bundle := config.LoadConfig(nil)
		m.applyBundle(bundle)
		if len(bundle.Broken) > 0 {
			m.pendingProfileErrors = append(m.pendingProfileErrors, bundle.Broken...)
			m.profileErrorQueueActive = true
			m = m.presentNextBrokenProfile()
		}
		// Restart the watcher for the next change
		configDir, _ := config.GetConfigDir()
		return m, WatchConfigCmd(configDir)

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

		// Global Emergency Exit: Ctrl+C should always quit (except in Locked Mode, handled below)
		if key == "ctrl+c" {
			m.Quitting = true
			return m, tea.Quit
		}

		// Update last activity time for any key press (except in locked mode)
		if m.mode != lockedMode {
			m.lastActivityTime = time.Now()
		}

		// Handle locked mode separately
		if m.mode == lockedMode {
			return m.updateLockedMode(msg)
		}

		// If searching in Path/Child mode, bypass global key handlers (lock, tab, etc.)
		// so that typing keys like 'r' or 'i' goes to the search filter instead.
		if (m.mode == pathMode || m.mode == childMode) && m.path.Searching {
			if m.mode == pathMode {
				mode, cmd := m.path.UpdatePathMode(msg, m.Config)
				m.mode = mode
				return m, cmd
			}
			mode, cmd := m.path.UpdateChildMode(msg, m.Config)
			m.mode = mode
			return m, cmd
		}

		if IsLock(m.Config.Keys, msg) {
			cmd := m.toggleProfileLock()
			return m, cmd
		}
		// Profile switching with configurable modifier + Number or ~ (Shift + `)
		if m.mode == gridMode || m.mode == childMode {
			if ok, target := IsProfileSwitch(m.Config.Keys, msg, m.Config.NumbModifier); ok {
				if target < len(m.profiles) {
					if updated, cmd, ok := m.switchToProfileIndex(target); ok {
						m = updated
						return m, cmd
					}
				}
				return m, nil
			}
			if IsProfilePrev(m.Config.Keys, msg) {
				return m.handleProfileCycle(-1)
			}
			if IsProfileNext(m.Config.Keys, msg) {
				return m.handleProfileCycle(1)
			}
		}
		switch m.mode {
		case gridMode:
			return m.updateGridMode(msg)
		case pathMode:
			mode, cmd := m.path.UpdatePathMode(msg, m.Config)
			m.mode = mode
			return m, cmd
		case childMode:
			mode, cmd := m.path.UpdateChildMode(msg, m.Config)
			m.mode = mode
			return m, cmd
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

			isActive := false
			if len(m.timeHistory) > 1 {
				duration := m.timeHistory[len(m.timeHistory)-1].Sub(m.timeHistory[0]).Seconds()
				sentDelta := m.sentHistory[len(m.sentHistory)-1] - m.sentHistory[0]
				recvDelta := m.recvHistory[len(m.recvHistory)-1] - m.recvHistory[0]

				if duration > 0 {
					sentBps := float64(sentDelta) / duration
					recvBps := float64(recvDelta) / duration
					m.traffic = themeNameStyle.Render(fmt.Sprintf("↓ %s ↑ %s", core.FormatTraffic(recvBps), core.FormatTraffic(sentBps)))
					if sentBps > 2*1024 || recvBps > 2*1024 {
						isActive = true
					}
				} else {
					m.traffic = themeNameStyle.Render("---")
				}
			} else {
				m.traffic = themeNameStyle.Render("calculating...")
			}

			if msg.online {
				if isActive {
					m.onlineStatus = onlineStyle.Render("online (active)")
				} else {
					m.onlineStatus = onlineStyle.Render("online (idle)")
				}
			} else {
				m.onlineStatus = offlineStyle.Render("offline")
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

	case lockCheckMsg:
		// Check if we should auto-lock
		if m.mode != lockedMode && m.lockTimeoutMins > 0 {
			elapsed := time.Since(m.lastActivityTime)
			if elapsed >= time.Duration(m.lockTimeoutMins)*time.Minute {
				log.Printf("Auto-locking after %v of inactivity", elapsed)
				m = m.enterLockedMode()
			}
		}
		return m, lockCheckTick()

	}

	return m, nil
}

func (m Model) updateGridMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	switch {
	case IsQuit(m.Config.Keys, msg):
		m.Quitting = true
		return m, tea.Quit
	case IsInventory(m.Config.Keys, msg):
		m.mode = inventoryMode
		m.inventory = InitInventoryModel(m.configDir)
		return m, nil
	case IsUp(m.Config.Keys, msg):
		m.moveCursor(-1, 0)
	case IsDown(m.Config.Keys, msg):
		m.moveCursor(1, 0)
	case IsLeft(m.Config.Keys, msg):
		m.moveCursor(0, -1)
	case IsRight(m.Config.Keys, msg):
		m.moveCursor(0, 1)
	case IsPathGridMode(m.Config.Keys, msg):
		m.mode = pathMode
	case IsExplain(m.Config.Keys, msg):
		selectedChoice := m.grid[m.cursorRow][m.cursorCol]
		if strings.TrimSpace(selectedChoice) == "" {
			return m, nil
		}
		for _, cmd := range m.Config.Commands {
			if cmd.Name == selectedChoice {
				m.previousMode = m.mode

				// Resolve execution mode and auto-close
				autoClose := true
				if cmd.AutoCloseExecution != nil {
					autoClose = *cmd.AutoCloseExecution
				}
				debug := false
				if cmd.DebugExecution != nil {
					debug = *cmd.DebugExecution
				}
				execMode := "live"
				if debug {
					execMode = "debug"
				}

				cmdStr := ""
				if strings.TrimSpace(cmd.Command) == "" {
					cmdStr = "Error: no command. ( This might be a folder of commands!)"
				} else {
					cmdStr = cmd.Command
				}

				m.activeDetail = &DetailState{
					Title:       selectedChoice,
					KeyLabel:    "Command",
					Value:       cmdStr,
					Description: cmd.Description,
					Meta: []DetailMeta{
						{Label: "Exec", Value: execMode},
						{Label: "Auto-close", Value: fmt.Sprintf("%v", autoClose)},
						{Label: "CWD", Value: m.path.CurrentPath},
					},
				}
				m.mode = infoMode
				return m, nil
			}
		}
		// Not found in config
		m.previousMode = m.mode
		m.activeDetail = &DetailState{
			Title:       selectedChoice,
			KeyLabel:    "Command",
			Value:       "Error: command not found",
			Description: "",
			Meta: []DetailMeta{
				{Label: "CWD", Value: m.path.CurrentPath},
			},
		}
		m.mode = infoMode
		return m, nil
	case IsConfirm(m.Config.Keys, msg):
		selectedChoice := m.grid[m.cursorRow][m.cursorCol]

		// Special handling for Exit Rescue Mode command
		if selectedChoice == "Exit Rescue Mode" {
			// Reset to Core profile (index 0)
			if updated, cmd, ok := m.switchToProfileIndex(0); ok {
				m = updated
				return m, cmd
			}
			return m, nil
		}

		if selectedChoice != "" {
			// Check if this command has dropdown items
			for _, cmd := range m.Config.Commands {
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
			m.Selected = selectedChoice
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

func (m Model) updateDropdownMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	switch {
	case IsCancel(m.Config.Keys, msg):
		// Close dropdown and return to grid mode
		m.mode = gridMode
		m.dropdownItems = nil
		return m, nil
	case Matches(m.Config.Keys, msg, "ctrl+c"):
		m.Quitting = true
		return m, tea.Quit
	case IsUp(m.Config.Keys, msg):
		if m.dropdownSelectedIdx > 0 {
			m.dropdownSelectedIdx--
		}
	case IsDown(m.Config.Keys, msg):
		if m.dropdownSelectedIdx < len(m.dropdownItems)-1 {
			m.dropdownSelectedIdx++
		}
	case IsExplain(m.Config.Keys, msg):
		if m.dropdownSelectedIdx >= 0 && m.dropdownSelectedIdx < len(m.dropdownItems) {
			item := m.dropdownItems[m.dropdownSelectedIdx]
			parent := ""
			if m.dropdownRow >= 0 && m.dropdownCol >= 0 && m.dropdownRow < len(m.grid) && m.dropdownCol < len(m.grid[0]) {
				parent = m.grid[m.dropdownRow][m.dropdownCol]
			}
			m.previousMode = m.mode

			title := item.Name
			if strings.TrimSpace(parent) != "" {
				title = fmt.Sprintf("%s: %s", parent, item.Name)
			}

			// Resolve execution mode and auto-close for item
			autoClose := true
			if item.AutoCloseExecution != nil {
				autoClose = *item.AutoCloseExecution
			}
			debug := false
			if item.DebugExecution != nil {
				debug = *item.DebugExecution
			}
			execMode := "live"
			if debug {
				execMode = "debug"
			}

			cmdStr := ""
			if strings.TrimSpace(item.Command) == "" {
				cmdStr = "Error: no command configured"
			} else {
				cmdStr = item.Command
			}

			m.activeDetail = &DetailState{
				Title:       title,
				KeyLabel:    "Command",
				Value:       cmdStr,
				Description: item.Description,
				Meta: []DetailMeta{
					{Label: "Exec", Value: execMode},
					{Label: "Auto-close", Value: fmt.Sprintf("%v", autoClose)},
					{Label: "CWD", Value: m.path.CurrentPath},
				},
			}
			m.mode = infoMode
			return m, nil
		}
		return m, nil
	case IsConfirm(m.Config.Keys, msg):
		// Execute the selected dropdown item
		if m.dropdownSelectedIdx >= 0 && m.dropdownSelectedIdx < len(m.dropdownItems) {
			selectedItem := m.dropdownItems[m.dropdownSelectedIdx]
			m.Selected = selectedItem.Name
			// Store the command to execute
			// We need to create a temporary command entry for execution
			return m, tea.Quit
		}
	}
	return m, nil
}

// copyToClipboardCmd copies text to clipboard using the best available method
func copyToClipboardCmd(s string) tea.Cmd {
	return func() tea.Msg {
		CopyToClipboard(s)
		return nil
	}
}

func (m Model) updateInfoMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if m.profileErrorQueueActive {
		var cmds []tea.Cmd
		if key == "y" {
			if m.activeDetail != nil {
				cmds = append(cmds, copyToClipboardCmd(m.activeDetail.Value))
			}
		}
		// Always delegate to presentNextBrokenProfile to handle queue exhaustion and Rescue Trigger
		m = m.presentNextBrokenProfile()
		return m, tea.Batch(cmds...)
	}
	switch key {
	case "y":
		if m.activeDetail != nil {
			prev := m.previousMode
			m.mode = prev
			cmd := copyToClipboardCmd(m.activeDetail.Value)
			m.activeDetail = nil // Clear detail state
			return m, cmd
		}
		m.mode = m.previousMode
		m.activeDetail = nil
		return m, nil
	default:
		m.mode = m.previousMode
		m.activeDetail = nil // Clear detail state
		return m, nil
	}
}

func (m Model) updateLockedMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Allow Ctrl+C to quit even when locked
	if key == "ctrl+c" {
		m.Quitting = true
		return m, tea.Quit
	}

	dir := pumpDirectionForKey(key)
	if dir == 0 {
		return m, nil
	}

	// Require alternating directions to "pump" the slider
	if m.lockLastDirection == dir {
		if m.lockProgress > 0 {
			m.lockProgress--
		}
		return m, nil
	}

	m.lockLastDirection = dir
	if m.lockProgress < m.lockPumpGoal {
		m.lockProgress++
	}

	if m.lockProgress >= m.lockPumpGoal {
		log.Printf("Unlocking via pump sequence after %d steps", m.lockProgress)
		m = m.exitLockedMode()
		return m, nil
	}

	return m, nil
}

func pumpDirectionForKey(key string) int {
	switch key {
	case "left", "h", "a":
		return -1
	case "right", "l", "d":
		return 1
	default:
		return 0
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

	switch {
	case IsCancel(m.Config.Keys, msg):
		m.mode = gridMode
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
			list := inv.visible
			if inv.focusedList == 1 {
				list = inv.inventory
			}
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

func (m Model) switchToProfileIndex(target int) (Model, tea.Cmd, bool) {
	if len(m.profiles) == 0 {
		return m, nil, false
	}
	total := len(m.profiles)
	if target < 0 || target >= total {
		target = ((target % total) + total) % total
	}

	selected := m.profiles[target]

	// Check for existence
	if _, err := os.Stat(selected.Path); err != nil {
		log.Printf("skipping missing profile: %s", selected.Path)
		return m, nil, false
	}

	updated := m
	updated.activeProfileIndex = target
	_ = os.Setenv("DRAKO_PROFILE", selected.Name)
	updated.Config = config.ApplyProfileOverlay(m.baseConfig, selected.Profile)
	updated.applyConfig(updated.Config)

	return updated, nil, true
}

func (m Model) handleProfileCycle(direction int) (tea.Model, tea.Cmd) {
	if len(m.profiles) <= 1 {
		return m, nil
	}

	current := m.activeProfileIndex
	total := len(m.profiles)
	// Only try up to 'total' times.
	// Start from 1 to avoid re-selecting the current profile immediately.
	for i := 1; i <= total; i++ {
		next := (current + i*direction) % total
		if next < 0 {
			next += total
		}
		nextModel, cmd, ok := m.switchToProfileIndex(next)
		if ok {
			return nextModel, cmd
		}
	}
	return m, nil
}

func (m *Model) moveCursor(rowDir, colDir int) {
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
