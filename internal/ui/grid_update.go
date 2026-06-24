package ui

import (
	"math"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/core"
)

func (m Model) updateGridMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle number-based navigation (1-9)
	if num, err := strconv.Atoi(key); err == nil && num >= 1 && num <= 9 {
		return m.quickNav(num - 1)
	}

	// Any non-numeric key cancels a navigation sequence in progress.
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

				cmdStr := cmd.Command
				if strings.TrimSpace(cmd.Command) == "" {
					cmdStr = "Error: no command. ( This might be a folder of commands!)"
				}

				m.activeDetail = newCommandDetail(selectedChoice, cmdStr, cmd.Description, cmd.AutoCloseExecution, cmd.DebugExecution, m.path.CurrentPath)
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
			if updated, ok := m.switchToProfileIndex(0); ok {
				m = updated
				return m, nil
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

// quickNav handles one 1-9 keypress (as a 0-based index) of the two-step
// column-then-row jump. The first press selects a column, parks on its first
// populated row, and arms a 500ms timer; a second press while that timer is
// live selects a row within the chosen column. The timer expiry
// (navTimeoutMsg) ends the sequence elsewhere.
func (m Model) quickNav(targetIndex int) (tea.Model, tea.Cmd) {
	if m.navigationTimer != nil {
		// Second press: choose a row within the already-selected column.
		m.navigationTimer.Stop()
		m.navigationTimer = nil
		lastRow := core.FindLastPopulatedRow(m.grid, m.cursorCol)
		m.cursorRow = min(targetIndex, lastRow)
		return m, nil
	}

	// First press: choose a column, parking on its first populated row.
	targetCol := min(targetIndex, core.FindLastPopulatedCol(m.grid))
	if targetCol < 0 {
		return m, nil
	}
	m.cursorCol = targetCol
	m.cursorRow = core.FindFirstPopulatedRow(m.grid, targetCol)
	m.navigationTimer = time.NewTimer(500 * time.Millisecond)
	return m, func() tea.Msg {
		<-m.navigationTimer.C
		return navTimeoutMsg{}
	}
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
