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

	// A pending leader sequence consumes the next key, whatever it is.
	if m.leader.pending {
		return m.handleLeaderContinuation(msg)
	}
	if IsLeader(m.Config.Keys, msg) {
		return m.armLeader()
	}

	// Handle number-based navigation (1-9)
	if num, err := strconv.Atoi(key); err == nil && num >= 1 && num <= 9 {
		return m.quickNav(num - 1)
	}

	// Any non-numeric key cancels a navigation sequence in progress.
	if m.gridNav.timer != nil {
		m.gridNav.timer.Stop()
		m.gridNav.timer = nil
	}

	switch {
	case IsQuit(m.Config.Keys, msg):
		m.Quitting = true
		return m, tea.Quit

	// Glassroot Cases - handled in update.go
	// ====================
	case IsInventory(m.Config.Keys, msg):
		m.mode = inventoryMode
		m.inventory = InitInventoryModel(m.profile.configDir)
		return m, nil

	case IsPathGridMode(m.Config.Keys, msg):
		m.mode = pathMode
	// ====================

	case IsUp(m.Config.Keys, msg):
		m.gridNav.moveCursor(-1, 0)
	case IsDown(m.Config.Keys, msg):
		m.gridNav.moveCursor(1, 0)
	case IsLeft(m.Config.Keys, msg):
		m.gridNav.moveCursor(0, -1)
	case IsRight(m.Config.Keys, msg):
		m.gridNav.moveCursor(0, 1)
	case IsExplain(m.Config.Keys, msg):
		selectedChoice := m.gridNav.grid[m.gridNav.cursorRow][m.gridNav.cursorCol]
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
		selectedChoice := m.gridNav.grid[m.gridNav.cursorRow][m.gridNav.cursorCol]

		// Special handling for Exit Rescue Mode command
		if selectedChoice == "Exit Rescue Mode" {
			// Reload the bundle and land on whatever is actually equipped;
			// with nothing equipped this resolves back to the rescue grid.
			return m, func() tea.Msg { return reloadProfilesMsg{} }
		}

		if selectedChoice != "" {
			// Check if this command has dropdown items
			for _, cmd := range m.Config.Commands {
				if cmd.Name == selectedChoice {
					if len(cmd.Items) > 0 {
						// Open dropdown menu
						m.mode = dropdownMode
						m.dropdown.row = m.gridNav.cursorRow
						m.dropdown.col = m.gridNav.cursorCol
						m.dropdown.items = cmd.Items
						m.dropdown.selectedIdx = 0
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
	if m.gridNav.timer != nil {
		// Second press: choose a row within the already-selected column.
		m.gridNav.timer.Stop()
		m.gridNav.timer = nil
		lastRow := core.FindLastPopulatedRow(m.gridNav.grid, m.gridNav.cursorCol)
		m.gridNav.cursorRow = min(targetIndex, lastRow)
		return m, nil
	}

	// First press: choose a column, parking on its first populated row.
	targetCol := min(targetIndex, core.FindLastPopulatedCol(m.gridNav.grid))
	if targetCol < 0 {
		return m, nil
	}
	m.gridNav.cursorCol = targetCol
	m.gridNav.cursorRow = core.FindFirstPopulatedRow(m.gridNav.grid, targetCol)
	m.gridNav.timer = time.NewTimer(500 * time.Millisecond)
	return m, func() tea.Msg {
		<-m.gridNav.timer.C
		return navTimeoutMsg{}
	}
}

func (g *gridNav) moveCursor(rowDir, colDir int) {
	bestRow, bestCol := -1, -1
	minDist := math.MaxFloat64

	for r, row := range g.grid {
		for c, val := range row {
			if val == "" || (r == g.cursorRow && c == g.cursorCol) {
				continue
			}

			rowDiff := r - g.cursorRow
			colDiff := c - g.cursorCol
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
		g.cursorRow = bestRow
		g.cursorCol = bestCol
	}
}
