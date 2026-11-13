package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/lucky7xz/drako/internal/host"
	"github.com/muesli/termenv"
)

// View is the screen the current mode sits on, with its popups floated over
// it. Explain has no screen of its own: it floats over the one it was opened
// from.
func (m Model) View() string {
	if m.termWidth == 0 {
		return "Initializing..."
	}

	// If even the minimum grid window cannot fit, show the blocking overlay
	if tooSmall, reqW, reqH := m.belowMinimum(); tooSmall {
		return m.renderSizeOverlay(reqW, reqH)
	}

	// The screen underneath is drawn as its own mode, but its footer hint is
	// this one's: under explain, q is not what the grid makes of it.
	under := m
	under.mode = m.underneath()
	frame := under.screen(m.keyHint())
	for _, popup := range m.popups(under.mode) {
		frame = overlay(frame, popup, m.termWidth, m.termHeight)
	}
	return frame
}

// underneath is the mode whose screen shows beneath the current one. Explain
// floats over the screen it was opened from; a profile note returns to the
// grid when it is dismissed, so it floats over the grid — a folder it
// interrupted belongs to the config the reload replaced.
func (m Model) underneath() navMode {
	switch {
	case m.mode != infoMode:
		return m.mode
	case m.profile.errorQueueActive:
		return gridMode
	}
	return m.previousMode
}

// screen is the mode's own screen, with nothing floating over it, its footer
// carrying hint. Every mode but the lock screen and the inventory sits on the
// grid.
func (m Model) screen(hint string) string {
	switch m.mode {
	case lockedMode:
		return m.viewLockedMode()
	case inventoryMode:
		return m.viewInventoryMode(hint)
	}
	return m.viewGrid(hint)
}

// popups lists what floats over the screen of mode under, bottom to top: the
// folder's dropdown, or the batch layout dialog that takes its place; the spec
// picker; explain; the key sheet. The lock screen hides them all. A dropdown
// with no items is a folder that already closed — previousMode can be stale.
func (m Model) popups(under navMode) []string {
	if m.mode == lockedMode {
		return nil
	}
	var ps []string
	switch {
	case m.batch.choosing():
		ps = append(ps, m.renderLayoutPopup())
	case under == dropdownMode && len(m.dropdown.items) > 0:
		ps = append(ps, m.renderDropdownPopup())
	}
	if under == inventoryMode && m.inventory.specs != nil {
		ps = append(ps, m.renderSpecsPopup())
	}
	if m.mode == infoMode {
		ps = append(ps, m.renderInfoPopup())
	}
	if m.sheet != nil {
		ps = append(ps, m.renderKeySheet())
	}
	return ps
}

// frame centres content on the screen. appStyle's margin sits inside the
// terminal, so the content gets the terminal less that margin and the frame
// is exactly the terminal's size: the renderer never has to crop it.
func (m Model) frame(content string) string {
	return appStyle.Render(lipgloss.Place(
		m.termWidth-appStyle.GetHorizontalMargins(), m.termHeight-appStyle.GetVerticalMargins(),
		lipgloss.Center, lipgloss.Center,
		content,
	))
}

// viewGrid is the grid screen: header, deck counter, grid and footer.
func (m Model) viewGrid(hint string) string {
	layout := CalculateLayout(m.termWidth, m.termHeight, m.Config, m.styles.HeaderArt)

	header := ""
	if layout.ShowHeader {
		header = m.styles.renderHeaderArt(m.spinner.View())
	}
	counter := m.renderProfileCounter()
	if m.mode == batchMode {
		counter = m.renderBatchCounter()
	}

	footer := m.renderCombinedFooter(hint)

	// Respect layout.ShowFooter
	if !layout.ShowFooter {
		footer = ""
	}

	// The grid gets whatever vertical space the chrome leaves over.
	grid := m.renderGrid(gridRowBudget(m.termHeight, header, counter, footer))
	mainContent := lipgloss.JoinVertical(lipgloss.Center, header, counter, grid)

	return m.frame(lipgloss.JoinVertical(lipgloss.Center, mainContent, footer))
}

// renderCombinedFooter creates the standard bottom block: Status | Profile |
// Path, the profile bar ending in hint. Every bar trims to the width rather
// than wrapping: m.net.traffic and m.net.online already carry ANSI color codes
// and can contain internal spaces (e.g. "↓ 1.2kb/s ↑ 3.4kb/s"), so
// word-wrapping a joined line risks splitting an ANSI start/reset pair across
// lines and bleeding color into unrelated text.
func (m Model) renderCombinedFooter(hint string) string {
	availWidth := m.termWidth - LayoutSideMargin

	networkStatusBar := lipgloss.NewStyle().PaddingTop(1).Render(m.renderNetworkBar(availWidth))

	// JoinVertical rectangularizes to the widest sibling and Place will not pad
	// content wider than the box, so one unbounded line skews the whole frame.
	profileBar := lipgloss.NewStyle().PaddingTop(1).Render(m.renderProfileBar(availWidth, hint))
	pathBar := m.path.RenderPathBar(m.mode == pathMode, m.styles, availWidth)
	childDirs := m.path.RenderChildDirs(m.mode, m.styles, availWidth)

	items := []string{networkStatusBar, profileBar}
	if statusMsg := m.renderStatusMessage(availWidth); statusMsg != "" {
		items = append(items, statusMsg)
	}
	items = append(items, pathBar, childDirs)

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

// renderNetworkBar shows NET | STATUS | THEME. When it doesn't fit
// availWidth, THEME drops first, then STATUS — NET (live throughput)
// stays as long as anything does.
func (m Model) renderNetworkBar(availWidth int) string {
	netText := lipgloss.NewStyle().Render("NET: ") + m.net.traffic
	statusText := fmt.Sprintf("STATUS: %s", m.net.online)
	themeName := m.styles.ThemeName.Render(m.Config.Theme)
	if profile := lipgloss.ColorProfile(); profile != termenv.TrueColor {
		themeName += m.styles.ThemeName.Render(" (" + host.ColorProfileName(profile) + ")")
	}
	themeText := "THEME: " + themeName
	sep := m.styles.Help.Render(" | ")

	full := netText + sep + statusText + sep + themeText
	if lipgloss.Width(full) <= availWidth {
		return full
	}

	mid := netText + sep + statusText
	if lipgloss.Width(mid) <= availWidth {
		return mid
	}

	return truncateText(netText, availWidth)
}

// truncateText clips a string to a max visual width. ansi.Truncate measures
// grapheme clusters (emoji + variation selectors), which per-rune width
// arithmetic gets wrong.
func truncateText(s string, maxLength int) string {
	if lipgloss.Width(s) <= maxLength {
		return s
	}
	return ansi.Truncate(s, maxLength, "...")
}

// truncateLeft clips s to maxLength from the left, keeping its end: for a
// path, the part that says where.
func truncateLeft(s string, maxLength int) string {
	w := lipgloss.Width(s)
	if w <= maxLength {
		return s
	}
	return ansi.TruncateLeft(s, w-maxLength+1, "…")
}

func (m Model) renderFooter() string {
	return m.styles.Footer.Render("[ github.com/lucky7xz | {chronyx}.xyz ]")
}

// rowLabelWidth is the digit width of the grid's 1-based row labels.
func rowLabelWidth(totalRows int) int {
	return len(fmt.Sprintf("%d", totalRows))
}

// belowMinimum reports whether the terminal cannot fit even the minimum
// grid window (2x2, capped by the grid's own size), plus the required
// dimensions for the overlay. It uses the same measured quantities as
// renderGrid, so the overlay fires exactly when the window would otherwise
// shrink below the floor — a 1-cell peephole is never rendered.
func (m Model) belowMinimum() (bool, int, int) {
	totalRows := len(m.gridNav.grid)
	totalCols := 0
	if totalRows > 0 {
		totalCols = len(m.gridNav.grid[0])
	}
	minRows := min(2, totalRows)
	minCols := min(2, totalCols)

	rowHeight := lipgloss.Height(m.styles.Cell.Render("x"))
	maxRowNumWidth := rowLabelWidth(totalRows)

	// Chrome floor: hidden header/counter/footer still cost one line each,
	// plus the column-header line; scroll markers only when the axis scrolls.
	reqH := appStyle.GetVerticalMargins() + 3 + 1 + minRows*rowHeight
	if totalRows > minRows {
		reqH++ // the ▾ ▴ marker line
	}
	// Cells compress down to MinCellFootprint before the overlay fires, so
	// long text cannot raise the demand past a constant; naturally narrower
	// cells lower it.
	reqW := appStyle.GetHorizontalMargins() + maxRowNumWidth + 1 +
		minCols*min(m.cellFootprint(), MinCellFootprint)
	if totalCols > minCols {
		reqW += 2 // the appended ▸ marker
	}
	return m.termWidth < reqW || m.termHeight < reqH, reqW, reqH
}
