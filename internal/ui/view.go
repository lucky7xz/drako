package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/lucky7xz/drako/internal/core"
	"github.com/muesli/termenv"
)

// View is the current screen, with the key sheet floating over it while open.
// The frame is wider than the terminal by appStyle's margin, so the sheet is
// centered on that width — in line with the screen's own content.
func (m Model) View() string {
	frame := m.view()
	if m.sheet == nil || m.termWidth == 0 {
		return frame
	}
	if tooSmall, _, _ := m.belowMinimum(); tooSmall {
		return frame
	}
	return overlay(frame, m.renderKeySheet(), m.termWidth+appStyle.GetHorizontalMargins(), m.termHeight)
}

func (m Model) view() string {
	if m.termWidth == 0 {
		return "Initializing..."
	}

	// If even the minimum grid window cannot fit, show the blocking overlay
	if tooSmall, reqW, reqH := m.belowMinimum(); tooSmall {
		return m.renderSizeOverlay(reqW, reqH)
	}

	if m.mode == lockedMode {
		return m.viewLockedMode()
	}

	if m.mode == inventoryMode {
		return m.viewInventoryMode()
	}

	// The layout dialog is reachable from the grid and from inside a folder,
	// and looks the same either way — so it is checked before the mode.
	if m.batch.choosing() {
		return m.viewLayoutDialog()
	}

	if m.mode == dropdownMode {
		return m.viewDropdownMode()
	}

	if m.mode == infoMode {
		return m.viewInfoMode()
	}

	layout := CalculateLayout(m.termWidth, m.termHeight, m.Config, m.styles.HeaderArt)

	header := ""
	if layout.ShowHeader {
		header = m.styles.renderHeaderArt(m.spinner.View())
	}
	counter := m.renderProfileCounter()
	if m.mode == batchMode {
		counter = m.renderBatchCounter()
	}

	footer := m.renderCombinedFooter()

	// Respect layout.ShowFooter
	if !layout.ShowFooter {
		footer = ""
	}

	// The grid gets whatever vertical space the chrome leaves over.
	grid := m.renderGrid(gridRowBudget(m.termHeight, header, counter, footer))
	mainContent := lipgloss.JoinVertical(lipgloss.Center, header, counter, grid)

	finalContent := lipgloss.JoinVertical(
		lipgloss.Center,
		mainContent,
		footer,
	)

	return appStyle.Render(
		lipgloss.Place(m.termWidth, m.termHeight,
			lipgloss.Center, lipgloss.Center,
			finalContent,
		),
	)
}

// renderCombinedFooter creates the standard bottom block: Status | Profile | Path.
// Every bar trims to the width rather than wrapping: m.net.traffic and
// m.net.online already carry ANSI color codes and can contain internal
// spaces (e.g. "↓ 1.2kb/s ↑ 3.4kb/s"), so word-wrapping a joined line risks
// splitting an ANSI start/reset pair across lines and bleeding color into
// unrelated text.
func (m Model) renderCombinedFooter() string {
	availWidth := m.termWidth - LayoutSideMargin

	networkStatusBar := lipgloss.NewStyle().PaddingTop(1).Render(m.renderNetworkBar(availWidth))

	// JoinVertical rectangularizes to the widest sibling and Place will not pad
	// content wider than the box, so one unbounded line skews the whole frame.
	profileBar := lipgloss.NewStyle().PaddingTop(1).Render(m.renderProfileBar(availWidth))
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
		themeName += m.styles.ThemeName.Render(" (" + core.ColorProfileName(profile) + ")")
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
		minCols*min(cellFootprint(m.gridNav.grid), MinCellFootprint)
	if totalCols > minCols {
		reqW += 2 // the appended ▸ marker
	}
	return m.termWidth < reqW || m.termHeight < reqH, reqW, reqH
}
