package ui

import (
	"fmt"
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
		targetIndex := num - 1 // Convert to 0-based index

		if m.navigationTimer == nil { // This is the first number press (column selection)
			lastCol := core.FindLastPopulatedCol(m.grid)
			targetCol := min(targetIndex, lastCol)

			// Ensure the target column is valid before proceeding
			if targetCol < 0 {
				return m, nil
			}

			targetRow := core.FindFirstPopulatedRow(m.grid, targetCol)

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

			lastRow := core.FindLastPopulatedRow(m.grid, m.cursorCol)
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
