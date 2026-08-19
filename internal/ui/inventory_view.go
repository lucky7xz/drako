package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
	"github.com/lucky7xz/drako/internal/profiles"
)

// inventoryHelpText is this view's footer help line — extracted to a
// package-level const so CalculateLayout and the footer-building code below
// measure/render the exact same string.
const inventoryHelpText = "↑/↓/tab: Switch Grid | ←/→: Move | space/enter: Lift/Place | e: Edit | q/esc: Back"

func (m Model) viewInventoryMode() string {
	// If there's an error, just show that.
	if m.inventory.err != nil {
		errorText := lipgloss.JoinVertical(lipgloss.Center,
			m.styles.ErrorTitle.Render("Error"),
			m.styles.ErrorText.Render(m.inventory.err.Error()),
			m.styles.Help.Render("\nPress any key to return to the grid."),
		)
		return appStyle.Render(
			lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center, errorText),
		)
	}

	// Calculate layout to determine visibility of header/footer. This
	// view's header is the "Inventory Management" title, not the ASCII-art
	// logo, so the width gate measures that string's actual length.
	layout := CalculateLayout(m.termWidth, m.termHeight, m.Config, "Inventory Management", inventoryHelpText)

	title := m.styles.InventoryTitle.Render("Inventory Management")

	visiblePtr, _ := m.inventory.State.GetList(core.ListVisible)
	visible := *visiblePtr
	inventoryPtr, _ := m.inventory.State.GetList(core.ListInventory)
	inventory := *inventoryPtr

	// The count is of the staged list, so the budget updates while arranging.
	equippedLabel := fmt.Sprintf("Equipped Items (%d/%d)", len(visible), profiles.MaxEquipped)
	if len(visible) > profiles.MaxEquipped {
		equippedLabel += " ⚠"
	}
	equippedHeader := m.styles.ListHeader.Render(equippedLabel)
	inventoryHeader := m.styles.ListHeader.Render("Inventory Items")

	applyButton := m.styles.Button.Render("[ Apply Changes ]")
	if m.inventory.focusedList == 2 {
		applyButton = m.styles.SelectedButton.Render("[ Apply Changes ]")
	}

	rescueButton := m.styles.RescueButton.Render("[ Rescue Mode ]")
	if m.inventory.focusedList == 3 {
		rescueButton = m.styles.SelectedRescueButton.Render("[ Rescue Mode ]")
	}

	heldItemStatus := " " // Reserve space
	if m.inventory.State.HeldItem != nil {
		heldItemStatus = m.styles.Help.Render("Holding: ") + m.styles.SelectedItem.Render(*m.inventory.State.HeldItem)
	}

	// A rejected action keeps holding the item, so this sits on its own line
	// under Holding: rather than crowding it. The line is reserved either way
	// so the lists below don't jump when a message appears; truncated rather
	// than wrapped to keep it exactly one row.
	statusLine := " " // Reserve space
	if m.inventory.status != "" {
		statusLine = m.styles.StatusNegative.Render(truncateText(m.inventory.status, m.termWidth-LayoutSideMargin))
	}

	var footer string
	if layout.ShowFooter {
		// Render Help, wrapped before styling (one Render() per wrapped
		// line) so narrow terminals get multiple lines instead of overflow.
		availWidth := m.termWidth - LayoutSideMargin
		wrapped := WrapText(inventoryHelpText, availWidth)
		helpLines := make([]string, len(wrapped))
		for i, line := range wrapped {
			helpLines[i] = m.styles.Help.Render(line)
		}
		help := lipgloss.JoinVertical(lipgloss.Left, helpLines...)
		version := m.styles.Help.Render(config.AppName + " | " + config.Version())
		footer = m.styles.Footer.Render(lipgloss.JoinVertical(lipgloss.Center, help, version))
	}

	// Everything except the two grids is fixed-height chrome — measure it
	// first (gridRowBudget sums lipgloss.Height of each piece, exactly like
	// the main grid's own budget calc) so the grids get whatever vertical
	// room is actually left. Section headers, buttons, and the held-item and
	// status lines are unconditional; title/footer follow the existing
	// layout.ShowHeader/ShowFooter gates.
	var chrome []string
	if layout.ShowHeader {
		chrome = append(chrome, title)
	}
	chrome = append(chrome, equippedHeader, inventoryHeader, applyButton, rescueButton, heldItemStatus, statusLine)
	if layout.ShowFooter {
		chrome = append(chrome, footer)
	}
	fixedBudget := gridRowBudget(m.termHeight, chrome...)

	// Split the remaining room between the two grids: whichever list is
	// focused gets priority (most of the space), the other is capped to a
	// small fixed row count. This is what stops a large Equipped Items
	// list (e.g. a big custom grid with many equipped slots) from
	// stranding a selected-but-invisible item at narrow widths, where both
	// lists' renderInventoryGrid wrap math drops to one item per row.
	rowHeight := lipgloss.Height(m.styles.Cell.Render("x"))
	secondaryBudget := minSecondaryListRows * rowHeight

	equippedBudget, inventoryBudget := secondaryBudget, secondaryBudget
	switch m.inventory.focusedList {
	case core.ListVisible:
		equippedBudget = fixedBudget - secondaryBudget
	case core.ListInventory:
		inventoryBudget = fixedBudget - secondaryBudget
	}

	equippedGrid := m.renderInventoryGrid(visible, core.ListVisible, equippedBudget)
	inventoryGrid := m.renderInventoryGrid(inventory, core.ListInventory, inventoryBudget)

	var sections []string
	if layout.ShowHeader {
		sections = append(sections, title)
	}
	sections = append(sections, equippedHeader, equippedGrid, inventoryHeader, inventoryGrid, applyButton, rescueButton, heldItemStatus, statusLine)
	if layout.ShowFooter {
		sections = append(sections, footer)
	}
	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	hAlign := lipgloss.Center
	if layout.ShiftLeft {
		hAlign = lipgloss.Left
	}

	return appStyle.Render(
		lipgloss.Place(m.termWidth, m.termHeight, hAlign, lipgloss.Center, content),
	)
}

// unlimitedRowBudget is a large sentinel that makes visibleCount always
// return the full row count, so no scrolling logic engages — used by tests
// to verify renderInventoryGrid's unwindowed behavior.
const unlimitedRowBudget = 1 << 30

// minSecondaryListRows is how many rows the non-priority list (Equipped or
// Inventory, whichever isn't focused) is guaranteed in viewInventoryMode's
// budget split, even when the focused list claims the rest.
const minSecondaryListRows = 2

// renderInventoryGrid wraps profiles into rows and windows those rows to
// rowBudget lines, reusing the same window()/visibleCount() primitives and
// ▴/▾ marker convention the main grid uses for its own scrolling. The
// window centers on the cursor's row when this list is focused; otherwise
// it starts from the top (window(0, ...) naturally yields offset 0), while
// still respecting rowBudget so an unfocused list can't overflow the page.
func (m Model) renderInventoryGrid(profiles []string, listID, rowBudget int) string {
	var cells []string
	isFocused := m.inventory.focusedList == listID

	// Add a placeholder cell for dropping if the list is empty
	if len(profiles) == 0 {
		style := m.styles.Cell
		if isFocused {
			style = m.styles.SelectedCell
		}
		cells = append(cells, style.Render(" (empty) "))
	} else {
		for i, p := range profiles {
			style := m.styles.Cell
			if isFocused && i == m.inventory.cursor {
				style = m.styles.SelectedCell
			}
			cells = append(cells, style.Render(p))
		}
	}

	if len(cells) == 0 {
		return ""
	}

	// Wrap cells into rows of however many fit per line.
	cellWidth := lipgloss.Width(cells[0])
	maxCellsPerLine := m.termWidth / cellWidth
	if maxCellsPerLine < 1 {
		maxCellsPerLine = 1
	}
	var rows []string
	for i := 0; i < len(cells); i += maxCellsPerLine {
		end := min(i+maxCellsPerLine, len(cells))
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Left, cells[i:end]...))
	}

	cursorRow := 0
	if isFocused {
		cursorRow = m.inventory.cursor / maxCellsPerLine
	}
	rowHeight := lipgloss.Height(m.styles.Cell.Render("x"))
	// reserve 1: the marker line while scrolling
	rowWin := window(cursorRow, len(rows), visibleCount(rowBudget, rowHeight, len(rows), 1))

	visibleRows := rows[rowWin.start:rowWin.end]

	if rowWin.scrolling() {
		down, up := " ", " "
		if rowWin.hiddenAfter > 0 {
			down = "▾"
		}
		if rowWin.hiddenBefore > 0 {
			up = "▴"
		}
		visibleRows = append(visibleRows, m.styles.SelectedCursor.Render(down+" "+up))
	}

	return lipgloss.JoinVertical(lipgloss.Left, visibleRows...)
}
