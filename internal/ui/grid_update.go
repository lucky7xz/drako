package ui

import (
	"fmt"
	"math"
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
