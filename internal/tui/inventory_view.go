package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/equip"
	"github.com/lucky7xz/drako/internal/version"
)

// viewInventoryMode is the inventory screen, its footer carrying hint.
func (m Model) viewInventoryMode(hint string) string {
	// If there's an error, just show that.
	if m.inventory.err != nil {
		errorText := lipgloss.JoinVertical(lipgloss.Center,
			m.styles.ErrorTitle.Render("Error"),
			m.styles.ErrorText.Render(m.inventory.err.Error()),
			m.styles.Help.Render("\nPress any key to return to the grid."),
		)
		return m.frame(errorText)
	}

	// The title asks for a roomy terminal, like the grid's header art. This
	// view's header is the "Inventory Management" title, not the ASCII-art
	// logo, so the width gate measures that string's actual length.
	layout := CalculateLayout(m.termWidth, m.termHeight, m.Config, "Inventory Management")

	title := m.styles.InventoryTitle.Render("Inventory Management")

	visiblePtr, _ := m.inventory.State.GetList(equip.ListVisible)
	visible := *visiblePtr
	inventoryPtr, _ := m.inventory.State.GetList(equip.ListInventory)
	inventory := *inventoryPtr

	// The count is of the staged list, so the budget updates while arranging.
	equippedLabel := fmt.Sprintf("Equipped Items (%d/%d)", len(visible), equip.MaxEquipped)
	if len(visible) > equip.MaxEquipped {
		equippedLabel += " ⚠"
	}
	equippedHeader := m.styles.ListHeader.Render(equippedLabel)
	inventoryHeader := m.styles.ListHeader.Render("Inventory Items")

	// The two buttons, each under margin blank lines. Lifting, placing and
	// specs only stage; while something is staged the Apply button says so.
	// Trash and edit act at once and say so themselves.
	buttons := func(margin int) (apply, rescue string) {
		apply = m.styles.Button.MarginTop(margin).Render("[ Apply Changes ]")
		if m.inventory.focusedList == focusApply {
			apply = m.styles.SelectedButton.MarginTop(margin).PaddingLeft(1).Render(buttonMark + "[ Apply Changes ]")
		}
		if staged := m.inventory.staged(); staged != "" {
			note := truncateText("  "+staged+" waiting — nothing changes until Apply", max(1, m.termWidth-LayoutSideMargin-lipgloss.Width(apply)))
			apply = lipgloss.JoinHorizontal(lipgloss.Bottom, apply, m.styles.Help.Render(note))
		}
		rescue = m.styles.RescueButton.MarginTop(margin).Render("[ Rescue Mode ]")
		if m.inventory.focusedList == focusRescue {
			rescue = m.styles.SelectedRescueButton.MarginTop(margin).PaddingLeft(1).Render(buttonMark + "[ Rescue Mode ]")
		}
		return apply, rescue
	}
	applyButton, rescueButton := buttons(1)

	// One line under the buttons carries every piece of inventory feedback — a
	// rejected action, and the delete confirmation prompt. It is reserved
	// either way so the lists below don't jump when a message appears, and
	// truncated rather than wrapped to keep it exactly one row.
	//
	// The prompt can't travel through status: that is cleared on every
	// keystroke, and the prompt has to survive the whole name being typed.
	statusLine := " " // Reserve space
	if pd := m.inventory.pending; pd != nil {
		prompt := fmt.Sprintf("Type '%s' to trash it now: %s_", pd.name, pd.typed)
		statusLine = m.styles.StatusNegative.Render(truncateText(prompt, m.termWidth-LayoutSideMargin))
	} else if m.inventory.status != "" {
		style := m.styles.StatusNegative
		if m.inventory.statusOK {
			style = m.styles.StatusPositive
		}
		statusLine = style.Render(truncateText(m.inventory.status, m.termWidth-LayoutSideMargin))
	} else if about := m.aboutSelected(); about != "" {
		statusLine = m.styles.Help.Render(truncateText(about, m.termWidth-LayoutSideMargin))
	}

	// The footer line carries the key hint, so it is always there. Trimmed
	// like the HOST bar: the version shortens, the hint stays.
	tail := " | " + hint
	room := max(1, m.termWidth-appStyle.GetHorizontalMargins()-lipgloss.Width(tail))
	label := truncateText(version.AppName+" | "+version.String(), room)
	footer := m.styles.Footer.PaddingTop(0).Render(m.styles.Help.Render(label + tail))

	// Everything except the two grids is fixed-height chrome — measure it
	// (gridRowBudget sums lipgloss.Height of each piece, exactly like the
	// main grid's own budget calc) so the grids get whatever vertical room is
	// actually left. Section headers, buttons, the status line and the footer
	// line are unconditional; the title follows layout.ShowHeader.
	budget := func() int {
		var chrome []string
		if layout.ShowHeader {
			chrome = append(chrome, title)
		}
		chrome = append(chrome, equippedHeader, inventoryHeader, applyButton, rescueButton, statusLine, footer)
		return gridRowBudget(m.termHeight, chrome...)
	}
	fixedBudget := budget()

	// The screen's blank lines take only room the lists can spare. The ones
	// over the buttons go when a list would otherwise not get a row with its
	// scroll marker. The three over the footer take only room the lists do
	// not need at all: with every deck on screen they are there, as before.
	rowHeight := lipgloss.Height(m.styles.Cell.Render("x"))
	equippedWhole := lipgloss.Height(m.renderInventoryGrid(visible, equip.ListVisible, unlimitedRowBudget))
	inventoryWhole := lipgloss.Height(m.renderInventoryGrid(inventory, equip.ListInventory, unlimitedRowBudget))
	if fixedBudget < min(equippedWhole, rowHeight+1)+min(inventoryWhole, rowHeight+1) {
		applyButton, rescueButton = buttons(0)
		fixedBudget = budget()
	}
	footerPad := min(max(fixedBudget-equippedWhole-inventoryWhole, 0), m.styles.Footer.GetPaddingTop())
	footer = m.styles.Footer.PaddingTop(footerPad).Render(m.styles.Help.Render(label + tail))
	fixedBudget -= footerPad

	// Split the remaining room between the two grids: whichever list is
	// focused gets priority (most of the space), the other is capped to a
	// small fixed row count, or to its own height when it is shorter. This
	// is what stops a large Equipped Items list (e.g. a big custom grid with
	// many equipped slots) from stranding a selected-but-invisible item at
	// narrow widths, where a row holds one item.
	// The focused list keeps at least a row and its scroll marker.
	secondaryBudget := minSecondaryListRows * rowHeight
	equippedBudget, inventoryBudget := min(secondaryBudget, equippedWhole), min(secondaryBudget, inventoryWhole)
	switch m.inventory.focusedList {
	case equip.ListVisible:
		inventoryBudget = min(inventoryBudget, fixedBudget-min(equippedWhole, rowHeight+1))
		equippedBudget = fixedBudget - inventoryBudget
	case equip.ListInventory:
		equippedBudget = min(equippedBudget, fixedBudget-min(inventoryWhole, rowHeight+1))
		inventoryBudget = fixedBudget - equippedBudget
	}

	equippedGrid := m.renderInventoryGrid(visible, equip.ListVisible, equippedBudget)
	inventoryGrid := m.renderInventoryGrid(inventory, equip.ListInventory, inventoryBudget)

	var sections []string
	if layout.ShowHeader {
		sections = append(sections, title)
	}
	sections = append(sections, equippedHeader, equippedGrid, inventoryHeader, inventoryGrid, applyButton, rescueButton, statusLine, footer)
	// The screen is as wide as the terminal, so its left edge stays where it
	// is whatever the lists hold. The spec picker floats over this screen
	// (View's popups), not inside it.
	avail := m.termWidth - appStyle.GetHorizontalMargins()
	return m.frame(lipgloss.NewStyle().Width(avail).Render(lipgloss.JoinVertical(lipgloss.Left, sections...)))
}

// buttonMark stands before the button the cursor is on, in the place of two
// columns of its padding: a sign that does not depend on colour.
const buttonMark = "► "

// activeMark stands before the deck the grid is showing.
const activeMark = "▸ "

// isActiveProfile reports whether name is the deck the grid is showing.
func (m Model) isActiveProfile(name string) bool {
	active := m.ActiveProfileName()
	return active != "" && equip.NormalizeName(active) == equip.NormalizeName(name)
}

// aboutSelected says what is special about the deck under the cursor: the one
// in use, the pinned one. It reads off the status line while nothing else has
// something to say there.
func (m Model) aboutSelected() string {
	inv := m.inventory
	if inv.focusedList != equip.ListVisible && inv.focusedList != equip.ListInventory {
		return ""
	}
	if inv.State.HeldItem != nil {
		return "" // the cursor stands for the held deck, not for one in the list
	}
	listPtr, err := inv.State.GetList(inv.focusedList)
	if err != nil || inv.cursor < 0 || inv.cursor >= len(*listPtr) {
		return ""
	}
	name := strings.TrimSuffix((*listPtr)[inv.cursor], equip.ProfileSuffix)
	var about []string
	if m.isActiveProfile(name) {
		about = append(about, "active deck")
	}
	if m.isLockedProfile(name) {
		about = append(about, "pinned — unpin with "+m.Config.Keys.Lock+" in the grid")
	}
	return strings.Join(about, " · ")
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
func (m Model) renderInventoryGrid(items []string, listID, rowBudget int) string {
	var cells []string
	isFocused := m.inventory.focusedList == listID

	// A held deck is drawn at the cursor, where Enter will put it: between
	// the decks either side of it, or after the last.
	held := -1
	if isFocused && m.inventory.State.HeldItem != nil {
		held = min(max(m.inventory.cursor, 0), len(items))
	}
	name := func(p string) string {
		if m.isActiveProfile(strings.TrimSuffix(p, equip.ProfileSuffix)) {
			return activeMark + p
		}
		return p
	}
	heldCell := func() string {
		return m.styles.SelectedCell.Render("« " + name(*m.inventory.State.HeldItem) + " »")
	}

	for i, p := range items {
		if i == held {
			cells = append(cells, heldCell())
		}
		selected := isFocused && held < 0 && i == m.inventory.cursor
		locked := m.isLockedProfile(strings.TrimSuffix(p, equip.ProfileSuffix))

		style := m.styles.Cell
		switch {
		case selected && locked:
			style = m.styles.LockedSelectedCell
		case selected:
			style = m.styles.SelectedCell
		case locked:
			style = m.styles.LockedCell
		}
		cells = append(cells, style.Render(name(p)))
	}
	switch {
	case held == len(items):
		cells = append(cells, heldCell())
	case len(items) == 0:
		// A placeholder to stand on while the list is empty.
		style := m.styles.Cell
		if isFocused {
			style = m.styles.SelectedCell
		}
		cells = append(cells, style.Render(" (empty) "))
	}

	if len(cells) == 0 {
		return ""
	}

	// Wrap cells into rows of however many fit inside appStyle's margin, by
	// each cell's own width: deck names differ in length.
	room := m.termWidth - appStyle.GetHorizontalMargins()
	cursorCell := -1
	if isFocused {
		cursorCell = min(max(m.inventory.cursor, 0), len(cells)-1)
	}
	var rows []string
	cursorRow, start, used := 0, 0, 0
	for i, c := range cells {
		w := lipgloss.Width(c)
		if i > start && used+w > room {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Left, cells[start:i]...))
			start, used = i, 0
		}
		used += w
		if i == cursorCell {
			cursorRow = len(rows)
		}
	}
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Left, cells[start:]...))

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
