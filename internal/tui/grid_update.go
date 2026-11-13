package tui

import (
	"math"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/settings"
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
	m.gridNav.cancel()

	switch {
	case IsQuit(m.Config.Keys, msg):
		m.quitting = true
		return m, tea.Quit

	// Glassroot Cases - handled in update.go
	// ====================
	case IsInventory(m.Config.Keys, msg):
		m.mode = inventoryMode
		m.inventory = InitInventoryModel(m.profile.configDir)
		return m, nil

	case IsPathGridMode(m.Config.Keys, msg):
		m.mode = pathMode
		m.path.clearSearch() // a filter left from the last visit would hide directories
	// ====================

	case m.gridNav.step(m.Config.Keys, msg):
		// a direction key: the cursor has moved
	case IsExplain(m.Config.Keys, msg):
		selectedChoice := m.gridNav.grid[m.gridNav.cursorRow][m.gridNav.cursorCol]
		if strings.TrimSpace(selectedChoice) == "" {
			return m, nil
		}
		if cmd, _ := m.cursorCell(); cmd != nil {
			m.previousMode = m.mode

			cmdStr := cmd.Command
			if strings.TrimSpace(cmd.Command) == "" {
				cmdStr = "A folder. Enter opens it."
			}

			m.activeDetail = newCommandDetail(cmd.Name, m.runAddress(cmd, 0), cmdStr, cmd.Description, cmd.AutoCloseExecution, m.path.CurrentPath)
			m.mode = infoMode
			return m, nil
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

		if cmd, sel := m.cursorCell(); cmd != nil {
			if len(cmd.Items) > 0 {
				// Open dropdown menu
				m.mode = dropdownMode
				m.dropdown.row = m.gridNav.cursorRow
				m.dropdown.col = m.gridNav.cursorCol
				m.dropdown.items = cmd.Items
				m.dropdown.selectedIdx = 0
				return m, nil
			}
			// Single command, execute normally
			m.selected = sel
			return m, tea.Quit
		}
	}
	return m, nil
}

// quickNav handles one 1-9 keypress (as a 0-based index) of the two-step
// column-then-row jump. The first press selects a column, parks on its first
// populated row, and arms a 500ms timer; a second press while that timer is
// live selects a row within the chosen column and briefly flashes it as a
// "locked in" confirmation. The timer expiry (navTimeoutMsg) and the flash
// expiry (rowFlashTimeoutMsg) end their respective states elsewhere.
func (m Model) quickNav(targetIndex int) (tea.Model, tea.Cmd) {
	if m.gridNav.navPending {
		// Second press: choose a row within the already-selected column.
		m.gridNav.navPending = false
		lastRow := findLastPopulatedRow(m.gridNav.grid, m.gridNav.cursorCol)
		m.gridNav.cursorRow = max(0, min(targetIndex, lastRow)) // an empty column's last row is -1

		m.gridNav.flashActive = true
		m.gridNav.flashSeq++
		seq := m.gridNav.flashSeq
		return m, tea.Tick(280*time.Millisecond, func(time.Time) tea.Msg {
			return rowFlashTimeoutMsg{seq: seq}
		})
	}

	// First press: choose a column, parking on its first populated row. A
	// fresh sequence starting mid-flash shouldn't leave the old row's flash
	// showing alongside the new column highlight.
	m.gridNav.flashActive = false

	targetCol := min(targetIndex, findLastPopulatedCol(m.gridNav.grid))
	if targetCol < 0 {
		return m, nil
	}
	m.gridNav.cursorCol = targetCol
	m.gridNav.cursorRow = findFirstPopulatedRow(m.gridNav.grid, targetCol)
	m.gridNav.navPending = true
	m.gridNav.navSeq++
	seq := m.gridNav.navSeq
	return m, tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return navTimeoutMsg{seq: seq}
	})
}

// step moves the cursor when msg is a direction key, and reports whether it
// was one. Grid mode and batch mode both steer the cursor this way.
func (g *gridNav) step(keys settings.InputConfig, msg tea.KeyMsg) bool {
	switch {
	case IsUp(keys, msg):
		g.moveCursor(-1, 0)
	case IsDown(keys, msg):
		g.moveCursor(1, 0)
	case IsLeft(keys, msg):
		g.moveCursor(0, -1)
	case IsRight(keys, msg):
		g.moveCursor(0, 1)
	default:
		return false
	}
	return true
}

// moveCursor jumps to the nearest filled cell in the given direction (one of
// rowDir, colDir is -1 or 1). The first cell found at the smallest distance
// wins; with none that way, the cursor stays.
func (g *gridNav) moveCursor(rowDir, colDir int) {
	bestRow, bestCol := -1, -1
	minDist := math.MaxInt

	for r, row := range g.grid {
		for c, val := range row {
			if val == "" || (r == g.cursorRow && c == g.cursorCol) {
				continue
			}

			rowDiff := r - g.cursorRow
			colDiff := c - g.cursorCol
			// That way: the cell lies on the side the key points to.
			thatWay := rowDir*rowDiff > 0 || colDir*colDiff > 0
			// Squared distance orders cells exactly as the distance does.
			if dist := rowDiff*rowDiff + colDiff*colDiff; thatWay && dist < minDist {
				minDist = dist
				bestRow, bestCol = r, c
			}
		}
	}

	if bestRow != -1 {
		g.cursorRow = bestRow
		g.cursorCol = bestCol
	}
}
