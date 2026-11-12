package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/launch"
)

// dropdownCell is the name of the folder cell the open dropdown belongs to.
func (m Model) dropdownCell() string {
	r, c := m.dropdown.row, m.dropdown.col
	if r < 0 || c < 0 || r >= len(m.gridNav.grid) || c >= len(m.gridNav.grid[r]) {
		return ""
	}
	return m.gridNav.grid[r][c]
}

func (m Model) updateDropdownMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// The layout dialog owns every key once it is open, the leader included.
	if m.batch.choosing() {
		return m.updateBatchDialog(msg)
	}

	// Leader sequence: inside a dropdown the only continuation is 'b'
	// (start marking this folder's items); everything else is swallowed.
	if m.leader.pending {
		m.disarmLeader()
		if key == "b" {
			return m.enterDropdownBatch()
		}
		return m, nil
	}
	if IsLeader(m.Config.Keys, msg) {
		return m.armLeader()
	}

	// An active dropdown batch owns space/enter/esc; other keys fall through
	// to the normal dropdown behavior below (digit jumps, nav, explain).
	if m.batch.dropdown {
		if next, cmd, handled := m.updateDropdownMarking(msg); handled {
			return next, cmd
		}
	}

	// Handle number-based navigation (1-9)
	if num, err := strconv.Atoi(key); err == nil && num >= 1 && num <= 9 {
		targetIndex := num - 1 // Convert to 0-based index
		if len(m.dropdown.items) > 0 {
			// If the target is out of bounds, select the last item.
			if targetIndex >= len(m.dropdown.items) {
				m.dropdown.selectedIdx = len(m.dropdown.items) - 1
			} else {
				m.dropdown.selectedIdx = targetIndex
			}
		}
		return m, nil
	}

	switch {
	case IsCancel(m.Config.Keys, msg):
		// Close dropdown and return to grid mode
		m.mode = gridMode
		m.dropdown.items = nil
		return m, nil
	case Matches(m.Config.Keys, msg, "ctrl+c"):
		m.quitting = true
		return m, tea.Quit
	case IsUp(m.Config.Keys, msg):
		if m.dropdown.selectedIdx > 0 {
			m.dropdown.selectedIdx--
		}
	case IsDown(m.Config.Keys, msg):
		if m.dropdown.selectedIdx < len(m.dropdown.items)-1 {
			m.dropdown.selectedIdx++
		}
	case IsExplain(m.Config.Keys, msg):
		if m.dropdown.selectedIdx >= 0 && m.dropdown.selectedIdx < len(m.dropdown.items) {
			item := m.dropdown.items[m.dropdown.selectedIdx]
			parent := m.dropdownCell()
			m.previousMode = m.mode

			title := item.Name
			if strings.TrimSpace(parent) != "" {
				title = fmt.Sprintf("%s: %s", parent, item.Name)
			}

			cmdStr := item.Command
			if strings.TrimSpace(item.Command) == "" {
				cmdStr = "Error: no command configured"
			}

			m.activeDetail = newCommandDetail(title, m.runAddress(parent, m.dropdown.selectedIdx+1), cmdStr, item.Description, item.AutoCloseExecution, m.path.CurrentPath)
			m.mode = infoMode
			return m, nil
		}
		return m, nil
	case IsConfirm(m.Config.Keys, msg):
		// Execute the selected dropdown item
		if m.dropdown.selectedIdx >= 0 && m.dropdown.selectedIdx < len(m.dropdown.items) {
			m.selected = launch.Selection{Cell: m.dropdownCell(), Item: m.dropdown.selectedIdx + 1}
			return m, tea.Quit
		}
	}
	return m, nil
}
