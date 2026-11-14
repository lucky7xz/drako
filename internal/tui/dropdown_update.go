package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/settings"
)

func (m Model) updateDropdownMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// The layout dialog owns every key once it is open, the leader included.
	if m.batch.choosing() {
		return m.updateBatchDialog(msg)
	}

	// The key table knows the folder in three states: after the leader (the
	// only continuation is b, start marking; everything else is swallowed),
	// marking for a batch (space, enter and esc belong to the batch), and
	// plain.
	act := m.act(msg)
	if m.leader.pending {
		m.disarmLeader()
		if act == settings.Batch {
			return m.enterDropdownBatch()
		}
		return m, nil
	}

	switch act {
	case settings.Leader:
		return m.armLeader()

	case settings.Jump:
		// Past the last item, select the last.
		if len(m.dropdown.items) > 0 {
			m.dropdown.selectedIdx = min(digit(msg), len(m.dropdown.items)-1)
		}
		return m, nil

	case settings.Back:
		if m.batch.dropdown {
			// End marking, keep the dropdown open; the next esc/q closes it.
			m.batch = batchState{}
			return m, nil
		}
		// Close dropdown and return to grid mode
		m.mode = gridMode
		m.dropdown.items = nil
		return m, nil

	case settings.Mark:
		return m.markDropdownItem()
	case settings.Launch:
		return m.beginLaunch()

	case settings.MoveV:
		if IsUp(m.Config.Keys, msg) {
			m.dropdown.selectedIdx = max(m.dropdown.selectedIdx-1, 0)
		} else if m.dropdown.selectedIdx < len(m.dropdown.items)-1 {
			m.dropdown.selectedIdx++
		}
	case settings.Explain:
		if m.dropdown.selectedIdx >= 0 && m.dropdown.selectedIdx < len(m.dropdown.items) {
			item := m.dropdown.items[m.dropdown.selectedIdx]
			parent, _ := m.dropdownCell()
			m.previousMode = m.mode

			title := item.Name
			if parent != nil && strings.TrimSpace(parent.Name) != "" {
				title = fmt.Sprintf("%s: %s", parent.Name, item.Name)
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
	case settings.Confirm:
		// Execute the selected dropdown item
		if m.dropdown.selectedIdx >= 0 && m.dropdown.selectedIdx < len(m.dropdown.items) {
			_, sel := m.dropdownCell()
			sel.Item = m.dropdown.selectedIdx + 1
			m.selected = sel
			return m, tea.Quit
		}
	}
	return m, nil
}
