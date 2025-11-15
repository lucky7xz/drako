package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Init() tea.Cmd {
	configDir, _ := getConfigDir()
	return tea.Batch(
		tea.EnterAltScreen,
		checkNetworkStatus(),
		m.spinner.Tick,
		watchConfigCmd(configDir),
		lockCheckTick(),
	)
}

// lockCheckTick creates a command that checks for auto-lock every 30 seconds
func lockCheckTick() tea.Cmd {
	return tea.Tick(30*time.Second, func(time.Time) tea.Msg {
		return lockCheckMsg{}
	})
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
		if len(bundle.Broken) > 0 {
			m.pendingProfileErrors = append(m.pendingProfileErrors, bundle.Broken...)
			m.profileErrorQueueActive = true
			m = m.presentNextBrokenProfile()
			return m, nil
		}
		m.mode = gridMode
		return m, nil

	case configChangedMsg:
		// Config file changed on disk, reload everything
		log.Printf("Config file change detected: %s", msg.path)
		bundle := loadConfig(nil)
		m.applyBundle(bundle)
		if len(bundle.Broken) > 0 {
			m.pendingProfileErrors = append(m.pendingProfileErrors, bundle.Broken...)
			m.profileErrorQueueActive = true
			m = m.presentNextBrokenProfile()
		}
		// Restart the watcher for the next change
		configDir, _ := getConfigDir()
		return m, watchConfigCmd(configDir)

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

		// Update last activity time for any key press (except in locked mode)
		if m.mode != lockedMode {
			m.lastActivityTime = time.Now()
		}

		// Handle locked mode separately
		if m.mode == lockedMode {
			return m.updateLockedMode(msg)
		}

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
			if key == "`" { // might use ~ too with shift
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
					m.traffic = themeNameStyle.Render(fmt.Sprintf("↓ %s ↑ %s", formatTraffic(recvBps), formatTraffic(sentBps)))
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
				if cmd.AutoCloseExecution != nil {
					autoClose = *cmd.AutoCloseExecution
				}
				debug := false
				if cmd.DebugExecution != nil {
					debug = *cmd.DebugExecution
				}
				if debug {
					m.infoExecMode = "debug"
				} else {
					m.infoExecMode = "live"
				}
				m.infoAutoClose = autoClose
				m.infoCwd = m.currentPath
				if strings.TrimSpace(cmd.Command) == "" {
					m.infoCommand = "Error: no command. ( This might be a folder of commands!)"
				} else {
					m.infoCommand = expandCommandTokens(cmd.Command, m.config)
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
			if item.AutoCloseExecution != nil {
				autoClose = *item.AutoCloseExecution
			}
			debug := false
			if item.DebugExecution != nil {
				debug = *item.DebugExecution
			}
			if debug {
				m.infoExecMode = "debug"
			} else {
				m.infoExecMode = "live"
			}
			m.infoAutoClose = autoClose
			m.infoCwd = m.currentPath
			if strings.TrimSpace(item.Command) == "" {
				m.infoCommand = "Error: no command configured"
			} else {
				m.infoCommand = expandCommandTokens(item.Command, m.config)
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

// copyToClipboardCmd copies text to clipboard using the best available method
func copyToClipboardCmd(s string) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(s) == "" {
			return nil
		}

		// Try multiple clipboard methods in order of preference
		tryClipboardMethods(s)
		return nil
	}
}

func (m model) updateInfoMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if m.profileErrorQueueActive {
		var cmds []tea.Cmd
		if key == "y" {
			cmds = append(cmds, copyToClipboardCmd(m.infoCommand))
		}
		if len(m.pendingProfileErrors) > 0 {
			m = m.presentNextBrokenProfile()
			return m, tea.Batch(cmds...)
		}
		m.profileErrorQueueActive = false
		m.mode = m.previousMode
		return m, tea.Batch(cmds...)
	}
	switch key {
	case "y":
		prev := m.previousMode
		m.mode = prev
		return m, copyToClipboardCmd(m.infoCommand)
	default:
		m.mode = m.previousMode
		return m, nil
	}
}

func (m model) updateLockedMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Allow Ctrl+C to quit even when locked
	if key == "ctrl+c" {
		m.quitting = true
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

// tryClipboardMethods attempts to copy text to clipboard using various methods
// based on the current environment, in order of preference.
func tryClipboardMethods(s string) {
	// First try platform-specific clipboard tools
	switch runtime.GOOS {
	case "linux":
		cmd, args := getLinuxClipboardCommand()
		if cmd != "" {
			if tryCommand(s, cmd, args...) {
				return
			}
		}
	case "darwin":
		if tryCommand(s, "pbcopy") {
			return
		}
	case "windows":
		// Try PowerShell first, then clip.exe
		if tryPowerShellClipboard(s) {
			return
		}
		if tryCommand(s, "clip.exe") {
			return
		}
	}

	// Then try OSC52 (direct terminal clipboard)
	if tryOSC52(s) {
		return
	}

	// Finally try tmux/screen wrappers
	if tryTmux(s) {
		return
	}
	if tryScreen(s) {
		return
	}
}

// isWayland checks if we are running under Wayland
func isWayland() bool {
	return os.Getenv("WAYLAND_DISPLAY") != "" || strings.Contains(os.Getenv("XDG_SESSION_TYPE"), "wayland")
}

// getLinuxClipboardCommand returns the appropriate clipboard command for Linux systems
func getLinuxClipboardCommand() (string, []string) {
	// Check for Wayland first
	if isWayland() {
		if _, err := exec.LookPath("wl-copy"); err == nil {
			return "wl-copy", []string{}
		}
	}

	// Check for X11 clipboard utilities
	if _, err := exec.LookPath("xclip"); err == nil {
		return "xclip", []string{"-selection", "clipboard"}
	}

	if _, err := exec.LookPath("xsel"); err == nil {
		return "xsel", []string{"--clipboard", "--input"}
	}

	// No clipboard utility found
	return "", []string{}
}

// tryOSC52 attempts to copy text using the OSC52 escape sequence
func tryOSC52(s string) bool {
	// Encode the string in base64
	enc := base64.StdEncoding.EncodeToString([]byte(s))

	// Check if we're running in tmux
	if os.Getenv("TMUX") != "" {
		// Tmux requires special wrapping
		seq := fmt.Sprintf("\x1bPtmux;\x1b\x1b]52;c;%s\x07\x1b\\", enc)
		_, err := os.Stderr.WriteString(seq)
		if err != nil {
			log.Printf("OSC52 copy failed: %v", err)
			return false
		}
		log.Printf("Attempted to copy to clipboard using OSC52 (tmux) - this requires terminal support")
		return true
	}

	// Check if we're running in screen
	if os.Getenv("STY") != "" {
		// Screen requires special wrapping
		seq := fmt.Sprintf("\x1bP\x1b]52;c;%s\x07\x1b\\", enc)
		_, err := os.Stderr.WriteString(seq)
		if err != nil {
			log.Printf("OSC52 copy failed: %v", err)
			return false
		}
		log.Printf("Attempted to copy to clipboard using OSC52 (screen) - this requires terminal support")
		return true
	}

	// Standard OSC52 sequence
	seq := fmt.Sprintf("\x1b]52;c;%s\x07", enc)
	_, err := os.Stderr.WriteString(seq)
	if err != nil {
		log.Printf("OSC52 copy failed: %v", err)
		return false
	}

	log.Printf("Attempted to copy to clipboard using OSC52 - this requires terminal support")
	return true
}

// tryTmux attempts to copy text when running inside tmux
func tryTmux(s string) bool {
	if os.Getenv("TMUX") == "" {
		return false
	}

	// Try both tmux escape sequences
	enc := base64.StdEncoding.EncodeToString([]byte(s))
	fmt.Printf("\033Ptmux;\033\033]52;c;%s\a\033\\", enc)
	return true
}

// tryScreen attempts to copy text when running inside screen
func tryScreen(s string) bool {
	if os.Getenv("STY") == "" {
		return false
	}

	// Screen requires a special escape sequence
	enc := base64.StdEncoding.EncodeToString([]byte(s))
	fmt.Printf("\033P\033]52;c;%s\a\033\\", enc)
	return true
}

// tryCommand attempts to copy text using an external command
func tryCommand(s string, name string, args ...string) bool {
	// Check if command exists
	if _, err := exec.LookPath(name); err != nil {
		log.Printf("Clipboard command not found: %s", name)
		return false
	}

	// Execute command with text as input
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(s)

	// Run command and return success status
	err := cmd.Run()
	if err != nil {
		log.Printf("Clipboard command failed: %s %v, error: %v", name, args, err)
		return false
	}

	log.Printf("Successfully copied to clipboard using: %s %v", name, args)
	return true
}

// tryPowerShellClipboard attempts to copy text using PowerShell on Windows
func tryPowerShellClipboard(s string) bool {
	if runtime.GOOS != "windows" {
		return false
	}

	// Try PowerShell with Set-Clipboard
	cmd := exec.Command("powershell.exe", "-NoLogo", "-NoProfile", "-Command", fmt.Sprintf("Set-Clipboard -Value \"%s\"", s))
	err := cmd.Run()
	return err == nil
}
