package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/core"
)

func (m Model) viewInventoryMode() string {
	// If there's an error, just show that.
	if m.inventory.err != nil {
		errorText := lipgloss.JoinVertical(lipgloss.Center,
			errorTitleStyle.Render("Error"),
			errorTextStyle.Render(m.inventory.err.Error()),
			helpStyle.Render("\nPress any key to return to the grid."),
		)
		return appStyle.Render(
			lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center, errorText),
		)
	}

	var s strings.Builder
	// Title
	s.WriteString(inventoryTitleStyle.Render("Inventory Management") + "\n\n")

	visiblePtr, _ := m.inventory.State.GetList(core.ListVisible)
	visible := *visiblePtr
	inventoryPtr, _ := m.inventory.State.GetList(core.ListInventory)
	inventory := *inventoryPtr

	// Draw visible list
	s.WriteString(listHeaderStyle.Render("Equipped Items") + "\n")
	s.WriteString(m.renderInventoryGrid(visible, 0))
	s.WriteString("\n\n")

	// Draw inventory list
	s.WriteString(listHeaderStyle.Render("Inventory Items") + "\n")
	s.WriteString(m.renderInventoryGrid(inventory, 1))
	s.WriteString("\n\n")

	// Render Apply Button
	applyButton := buttonStyle.Render("[ Apply Changes ]")
	if m.inventory.focusedList == 2 {
		applyButton = selectedButtonStyle.Render("[ Apply Changes ]")
	}
	s.WriteString(applyButton)

	// Render Rescue Mode Button
	s.WriteString("\n\n")
	rescueButton := rescueButtonStyle.Render("[ Rescue Mode ]")
	if m.inventory.focusedList == 3 {
		rescueButton = selectedRescueButtonStyle.Render("[ Rescue Mode ]")
	}
	s.WriteString(rescueButton)

	// Render Held Item Status
	heldItemStatus := " " // Reserve space
	if m.inventory.State.HeldItem != nil {
		heldItemStatus = helpStyle.Render("Holding: ") + selectedItemStyle.Render(*m.inventory.State.HeldItem)
	}
	s.WriteString("\n\n" + heldItemStatus)

	// Render Help
	help := helpStyle.Render("\n\n↑/↓/jk: Switch Grid | ←/→/hl: Move | space/enter: Lift/Place | tab: Focus Apply | q/esc: Back")
	s.WriteString(help)

	return appStyle.Render(
		lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center, s.String()),
	)
}

func (m Model) renderInventoryGrid(profiles []string, listID int) string {
	var cells []string
	isFocused := m.inventory.focusedList == listID

	// Add a placeholder cell for dropping if the list is empty
	if len(profiles) == 0 {
		style := cellStyle
		if isFocused {
			style = selectedCellStyle
		}
		cells = append(cells, style.Render(" (empty) "))
	} else {
		for i, p := range profiles {
			style := cellStyle
			if isFocused && i == m.inventory.cursor {
				style = selectedCellStyle
			}
			cells = append(cells, style.Render(p))
		}
	}

	// Wrap cells into multiple lines if there are too many to fit on one line
	if len(cells) == 0 {
		return ""
	}

	// Calculate how many cells can fit on one line
	cellWidth := lipgloss.Width(cells[0])
	maxCellsPerLine := m.termWidth / cellWidth

	// If we can fit all cells on one line, do so
	if len(cells) <= maxCellsPerLine || maxCellsPerLine <= 0 {
		return lipgloss.JoinHorizontal(lipgloss.Left, cells...)
	}

	// Otherwise, wrap into multiple lines
	var lines []string
	for i := 0; i < len(cells); i += maxCellsPerLine {
		end := i + maxCellsPerLine
		if end > len(cells) {
			end = len(cells)
		}
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, cells[i:end]...))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
