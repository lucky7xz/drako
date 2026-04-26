package ui

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) updateGridMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle number-based navigation (1-9)
	if num, err := strconv.Atoi(key); err == nil && num >= 1 && num <= 9 {
		targetIndex := num - 1 // Convert to 0-based index

		if m.navigationTimer == nil { // This is the first number press (column selection)
			lastCol := len(m.grid[0]) - 1
			targetCol := min(targetIndex, lastCol)

			// Ensure the target column is valid before proceeding
			if targetCol < 0 {
				return m, nil
			}

			m.cursorCol = targetCol
			// Snap Row to first populated in this column
			m.cursorRow = 0
			for r := 0; r < len(m.grid); r++ {
				if strings.TrimSpace(m.grid[r][targetCol]) != "" {
					m.cursorRow = r
					break
				}
			}

			timeoutMs := m.Config.GridSelectionTimeoutMs
			if timeoutMs <= 0 {
				timeoutMs = 500
			}
			m.navigationTimer = time.NewTimer(time.Duration(timeoutMs) * time.Millisecond)

			return m, func() tea.Msg {
				<-m.navigationTimer.C
				return navTimeoutMsg{}
			}

		} else { // This is the second number press (row selection)
			m.navigationTimer.Stop()
			m.navigationTimer = nil

			lastRow := len(m.grid) - 1
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

	// Glassroot Cases - handled in update.go
	// ====================
	case IsInventory(m.Config.Keys, msg):
		m.mode = inventoryMode
		m.inventory = InitInventoryModel(m.configDir)
		return m, nil

	case IsPathGridMode(m.Config.Keys, msg):
		m.mode = pathMode
	// ====================

	case IsUp(m.Config.Keys, msg):
		m.moveCursor(-1, 0)
	case IsDown(m.Config.Keys, msg):
		m.moveCursor(1, 0)
	case IsLeft(m.Config.Keys, msg):
		m.moveCursor(0, -1)
	case IsRight(m.Config.Keys, msg):
		m.moveCursor(0, 1)
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
func (m Model) resolveMouseClick(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}

	if m.mode == dropdownMode {
		return m.resolveDropdownMouseClick(msg)
	}

	// Handle path mode and child mode mouse clicks
	if m.mode == pathMode || m.mode == pickerMode {
		return m.resolvePathMouseClick(msg)
	}

	if m.mode == inventoryMode {
		return m.resolveInventoryMouseClick(msg)
	}

	if m.mode == infoMode {
		m.mode = m.previousMode
		m.activeDetail = nil
		return m, nil
	}

	// In grid mode, check if click is on path bar area (footer) - enter path mode
	if m.mode == gridMode {
		// Check if click is in the footer area where path bar is shown
		pathBarY, inPathArea := m.getPathBarPosition()
		if inPathArea && msg.Y >= pathBarY {
			// Enter path mode
			m.mode = pathMode
			// Try to detect which path component was clicked
			components := m.path.PathComponents
			if len(components) > 0 {
				if msg.X < 15 {
					m.path.SelectedPathIndex = 0
				} else {
					accumX := 0
					found := false
					for i, comp := range components {
						compWidth := lipgloss.Width(comp) + 1
						if msg.X >= accumX && msg.X < accumX+compWidth {
							m.path.SelectedPathIndex = i
							found = true
							break
						}
						accumX += compWidth
					}
					if !found {
						m.path.SelectedPathIndex = len(components) - 1
					}
				}
				m.path.ListChildDirs()
			}
			return m, nil
		}
	}

	layout := CalculateLayout(m.termWidth, m.termHeight, m.Config)

	// --- 1. Measure Exact Component Dimensions ---

	var (
		headerContent  string
		counterContent string
		buttonsContent string
		gridContent    string
		footerContent  string
	)

	if layout.ShowHeader {
		headerContent = renderDefaultHeaderArt(m.spinner.View())
	}
	counterContent = m.renderProfileCounter()
	buttonsContent = m.renderProfileButtons()
	gridContent = m.renderGrid() // This includes grid header and all rows

	if layout.ShowFooter {
		helpText := "Grid Mode | Enter: Select, e: Explain, Tab: Path, r: Start-Lock, i: Inventory"
		help := helpStyle.Render(helpText)
		footerContent = m.renderCombinedFooter(help)
	}

	hHeight, hWidth := lipgloss.Height(headerContent), lipgloss.Width(headerContent)
	cHeight, cWidth := lipgloss.Height(counterContent), lipgloss.Width(counterContent)
	bHeight, bWidth := lipgloss.Height(buttonsContent), lipgloss.Width(buttonsContent)
	gHeight, gWidth := lipgloss.Height(gridContent), lipgloss.Width(gridContent)
	fHeight, fWidth := lipgloss.Height(footerContent), lipgloss.Width(footerContent)

	// --- 2. Calculate Centering Offsets ---

	totalHeight := hHeight + cHeight + bHeight + gHeight + fHeight
	widest := max(hWidth, cWidth, bWidth, gWidth, fWidth)

	yOffset := (m.termHeight - totalHeight) / 2
	xOffset := (m.termWidth - widest) / 2

	// --- 3. Hit Detection ---

	// Profile Buttons Detection
	buttonsY := yOffset + hHeight + cHeight
	if msg.Y == buttonsY {
		currentX := xOffset + (widest-bWidth)/2
		for i := 0; i < len(m.profiles); i++ {
			// Re-render single button to get its exact width
			style := profileButtonStyle
			if i == m.activeProfileIndex {
				style = activeProfileButtonStyle
			}
			profile := m.profiles[i]
			label := fmt.Sprintf("%d", i+1)
			if strings.TrimSpace(profile.Profile.Icon) != "" {
				label = profile.Profile.Icon
			}
			btnWidth := lipgloss.Width(style.Render(fmt.Sprintf(" %s ", label)))

			if msg.X >= currentX && msg.X < currentX+btnWidth {
				if updated, cmd, ok := m.switchToProfileIndex(i); ok {
					return updated, cmd
				}
				break
			}
			currentX += btnWidth
		}
	}

	// Grid Detection
	gridYStart := yOffset + hHeight + cHeight + bHeight + 1 // +1 skips grid header
	gridYEnd := gridYStart + (len(m.grid) * GridCellHeight)

	if msg.Y >= gridYStart && msg.Y < gridYEnd {
		blockXStart := xOffset + (widest-gWidth)/2

		maxRowNumWidth := len(fmt.Sprintf("%d", len(m.grid)))
		rowPrefixWidth := maxRowNumWidth + 1

		gridXStart := blockXStart + rowPrefixWidth

		// Recalculate dynamic cellWidth matching renderGrid
		maxCellContentWidth := 0
		for _, row := range m.grid {
			for _, cell := range row {
				w := lipgloss.Width(cell)
				if w > maxCellContentWidth {
					maxCellContentWidth = w
				}
			}
		}
		if maxCellContentWidth > GridMaxTextWidth {
			maxCellContentWidth = GridMaxTextWidth
		}
		actualCellWidth := maxCellContentWidth + 4

		if msg.X >= gridXStart && msg.X < blockXStart+gWidth {
			clickedCol := (msg.X - gridXStart) / actualCellWidth
			clickedRow := (msg.Y - gridYStart) / GridCellHeight

			if clickedRow < len(m.grid) && clickedCol < len(m.grid[0]) {
				// Double-click execution: if clicking already-selected cell, execute it
				if clickedCol == m.cursorCol && clickedRow == m.cursorRow {
					// Same cell - execute the command
					return m.updateGridMode(tea.KeyMsg{Type: tea.KeyEnter})
				}
				// Different cell - just select it
				m.cursorCol = clickedCol
				m.cursorRow = clickedRow
			}
		}
	}

	return m, nil
}

func (m Model) resolveDropdownMouseClick(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}

	var raw []string
	for _, item := range m.dropdownItems {
		raw = append(raw, item.Name)
	}
	content := lipgloss.JoinVertical(lipgloss.Left, raw...)
	popupHeight := lipgloss.Height(content)
	popupWidth := lipgloss.Width(content)

	popupYStart := (m.termHeight - popupHeight) / 2
	popupYEnd := popupYStart + popupHeight
	popupXStart := (m.termWidth - popupWidth) / 2
	popupXEnd := popupXStart + popupWidth

	if msg.Y < popupYStart || msg.Y >= popupYEnd || msg.X < popupXStart || msg.X >= popupXEnd {
		m.mode = gridMode
		m.dropdownItems = nil
		return m, nil
	}

	localY := msg.Y - popupYStart
	cellHeight := 1

	if localY >= 0 && localY < len(m.dropdownItems)*cellHeight {
		clickedIdx := localY / cellHeight
		if clickedIdx >= 0 && clickedIdx < len(m.dropdownItems) {
			item := m.dropdownItems[clickedIdx]
			m.Selected = item.Name
			m.dropdownItems = nil
			return m, tea.Quit
		}
	}

	return m, nil
}

// resolvePathMouseClick handles mouse clicks in path mode and child mode
func (m Model) resolvePathMouseClick(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	layout := CalculateLayout(m.termWidth, m.termHeight, m.Config)

	// Calculate positions
	var (
		headerContent  string
		counterContent string
		buttonsContent string
		gridContent    string
		footerContent  string
		pathBarContent string
	)

	if layout.ShowHeader {
		headerContent = renderDefaultHeaderArt(m.spinner.View())
	}
	counterContent = m.renderProfileCounter()
	buttonsContent = m.renderProfileButtons()
	gridContent = m.renderGrid()

	// Path mode elements
	pathBarContent = m.path.RenderPathBar(true)
	childDirsContent := m.path.RenderChildDirs(m.mode)

	if layout.ShowFooter {
		helpText := "Path Mode | ←/→/ad: Select, ↑/s: Children, Enter: cd, e: Search, q/Esc: Back"
		help := helpStyle.Render(helpText)
		footerContent = m.renderCombinedFooter(help)
	}

	// Calculate heights
	hHeight := lipgloss.Height(headerContent)
	cHeight := lipgloss.Height(counterContent)
	bHeight := lipgloss.Height(buttonsContent)
	gHeight := lipgloss.Height(gridContent)
	pathBarH := lipgloss.Height(pathBarContent)   // 2: statusBarStyle.PaddingTop(1) + content
	childDirsH := lipgloss.Height(childDirsContent)
	fHeight := lipgloss.Height(footerContent)

	// Calculate centering offset
	totalHeight := hHeight + cHeight + bHeight + gHeight + fHeight
	widest := lipgloss.Width(footerContent)
	if w := lipgloss.Width(gridContent); w > widest {
		widest = w
	}
	yOffset := (m.termHeight - totalHeight) / 2

	// Calculate y offset for footer section (accounting for centering)
	footerY := yOffset + hHeight + cHeight + bHeight + gHeight

	// Compute positions by working backward from the footer end.
	// Footer order: help | net(PaddingTop+content) | profile | pathBar(PaddingTop+content) | childDirs
	// This avoids relying on hardcoded line offsets that break when styles change.
	childDirsY := footerY + fHeight - childDirsH
	pathBarBlockStart := childDirsY - pathBarH

	// Click above the entire path/picker UI (in grid area or header) returns to grid mode.
	if msg.Y < pathBarBlockStart {
		m.mode = gridMode
		return m, nil
	}

	// Click on path bar - enter path mode OR navigate to clicked path component.
	if msg.Y >= pathBarBlockStart && msg.Y < childDirsY && msg.X >= 0 {
		components := m.path.PathComponents
		if len(components) > 0 {
			var clickedIndex int

			// Click on the first component (home or root)
			if msg.X < 15 {
				clickedIndex = 0
			} else {
				// Try to detect which component was clicked
				accumX := 0
				found := false
				for i, comp := range components {
					compWidth := lipgloss.Width(comp) + 1 // +1 for separator
					if msg.X >= accumX && msg.X < accumX+compWidth {
						clickedIndex = i
						found = true
						break
					}
					accumX += compWidth
				}
				if !found {
					clickedIndex = len(components) - 1
				}
			}

			// Check for double-click on same position to exit path mode
			now := time.Now()
			clickDelta := now.Sub(m.lastClickTime)
			isDoubleClick := clickDelta < 500*time.Millisecond &&
				msg.X == m.lastClickPos.x && msg.Y == m.lastClickPos.y

			// Update last click info
			m.lastClickTime = now
			m.lastClickPos.x = msg.X
			m.lastClickPos.y = msg.Y

			// If already in path mode
			if m.mode == pathMode {
				if isDoubleClick {
					// Double-click - choose this path and exit to grid
					targetPath := m.path.BuildPathFromComponents(clickedIndex)
					if err := os.Chdir(targetPath); err == nil {
						m.path.CurrentPath, _ = os.Getwd()
						m.mode = gridMode
						return m, func() tea.Msg { return pathChangedMsg{} }
					}
					// Even if chdir fails, still exit to grid
					m.mode = gridMode
					return m, nil
				}

				// Single click - just select the path component (don't navigate)
				m.path.SelectedPathIndex = clickedIndex
				m.path.ListChildDirs()
			} else {
				// Not in path mode yet - enter path mode and select component
				m.mode = pathMode
				m.path.SelectedPathIndex = clickedIndex
				m.path.ListChildDirs()
			}
		}
		return m, nil
	}

	// Click on child directories (in picker mode or path mode)
	if msg.Y >= childDirsY && msg.Y < childDirsY+childDirsH && len(m.path.ChildDirs) > 0 {
		localY := msg.Y - childDirsY
		clickedIndex := localY

		if clickedIndex >= 0 && clickedIndex < len(m.path.ChildDirs) {
			// Check for double-click to enter directory
			now := time.Now()
			clickDelta := now.Sub(m.lastClickTime)
			isDoubleClick := clickDelta < 500*time.Millisecond &&
				msg.X == m.lastClickPos.x && msg.Y == m.lastClickPos.y

			// Update last click info BEFORE checking double-click
			m.lastClickTime = now
			m.lastClickPos.x = msg.X
			m.lastClickPos.y = msg.Y

			if isDoubleClick {
				// Double-click - enter the directory and return to grid
				parentPath := m.path.BuildPathFromComponents(m.path.SelectedPathIndex)
				targetPath := filepath.Join(parentPath, m.path.ChildDirs[clickedIndex])

				if err := os.Chdir(targetPath); err == nil {
					m.path.CurrentPath, _ = os.Getwd()
					m.path.UpdatePathComponents()
					m.path.ListChildDirs()
					m.path.SelectedChildIndex = 0
					m.mode = gridMode
					return m, func() tea.Msg { return pathChangedMsg{} }
				}
			}

			// Single click - just select the directory (like keyboard navigation)
			m.path.SelectedChildIndex = clickedIndex

			// If in pathMode, enter pickerMode to show selection
			if m.mode == pathMode {
				m.mode = pickerMode
			}
			return m, nil
		}
	}

	// Click on help/footer area - go back to grid
	if msg.Y >= footerY+fHeight-2 {
		// Click on footer area returns to grid
	}

	return m, nil
}

// ChangeDir is a helper to change directory and update path model
func (pm *PathModel) ChangeDir(targetPath string) error {
	if err := os.Chdir(targetPath); err != nil {
		return err
	}
	pm.CurrentPath, _ = os.Getwd()
	return nil
}

// getPathBarPosition returns the Y position of the path bar in the footer
// Returns (y position, true) if the path bar is visible
func (m Model) getPathBarPosition() (int, bool) {
	layout := CalculateLayout(m.termWidth, m.termHeight, m.Config)
	if !layout.ShowFooter {
		return 0, false
	}

	// Calculate positions accounting for centering
	var (
		headerContent  string
		counterContent string
		buttonsContent string
		gridContent    string
		footerContent  string
	)

	if layout.ShowHeader {
		headerContent = renderDefaultHeaderArt(m.spinner.View())
	}
	counterContent = m.renderProfileCounter()
	buttonsContent = m.renderProfileButtons()
	gridContent = m.renderGrid()

	helpText := "Grid Mode | Enter: Select, e: Explain, Tab: Path, r: Start-Lock, i: Inventory"
	help := helpStyle.Render(helpText)
	footerContent = m.renderCombinedFooter(help)

	// Calculate heights and centering
	hHeight := lipgloss.Height(headerContent)
	cHeight := lipgloss.Height(counterContent)
	bHeight := lipgloss.Height(buttonsContent)
	gHeight := lipgloss.Height(gridContent)
	fHeight := lipgloss.Height(footerContent)

	totalHeight := hHeight + cHeight + bHeight + gHeight + fHeight
	yOffset := (m.termHeight - totalHeight) / 2

	footerStartY := yOffset + hHeight + cHeight + bHeight + gHeight

	// Compute pathBar position backward from footer end (same approach as resolvePathMouseClick).
	childDirsContent := m.path.RenderChildDirs(m.mode)
	childDirsH := lipgloss.Height(childDirsContent)
	pathBarContent := m.path.RenderPathBar(m.mode == pathMode)
	pathBarH := lipgloss.Height(pathBarContent)
	pathBarY := footerStartY + fHeight - childDirsH - pathBarH

	return pathBarY, true
}
