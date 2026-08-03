package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) View() string {
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

	if m.mode == dropdownMode {
		return m.viewDropdownMode()
	}

	if m.mode == infoMode {
		return m.viewInfoMode()
	}

	var helpText string
	switch m.mode {
	case pathMode:
		helpText = "Path Mode | ←/→/ad: Select, ↓/s: Children, Enter: cd, e: Search, q/Esc: Back"
	case childMode:
		helpText = "Child Mode | ↑/↓/ws: Select, Enter: cd, e: Search, q/Esc: Back"
	case batchMode:
		helpText = m.batchHelpText()
	default:
		helpText = "Grid Mode | Enter: Select, e: Explain, Tab: Path, r: Start-Lock, i: Inventory"
	}

	layout := CalculateLayout(m.termWidth, m.termHeight, m.Config, m.styles.HeaderArt, helpText)

	header := ""
	if layout.ShowHeader {
		header = m.styles.renderHeaderArt(m.spinner.View())
	}
	counter := m.renderProfileCounter()
	if m.mode == batchMode {
		counter = m.renderBatchCounter()
	}

	footer := m.renderCombinedFooter(helpText)

	// Respect layout.ShowFooter
	if !layout.ShowFooter {
		footer = ""
	}

	hAlign := lipgloss.Center
	if layout.ShiftLeft {
		hAlign = lipgloss.Left
	}

	// The grid gets whatever vertical space the chrome leaves over. hAlign
	// governs these inner joins too, not just the outer Place() below:
	// otherwise the grid (narrower than the wrapped footer text) would stay
	// centered relative to its siblings even while the block as a whole
	// shifts left, undercutting the actual goal of moving the cells left.
	grid := m.renderGrid(gridRowBudget(m.termHeight, header, counter, footer))
	mainContent := lipgloss.JoinVertical(hAlign, header, counter, grid)

	finalContent := lipgloss.JoinVertical(
		hAlign,
		mainContent,
		footer,
	)

	return appStyle.Render(
		lipgloss.Place(m.termWidth, m.termHeight,
			hAlign, lipgloss.Center,
			finalContent,
		),
	)
}

// renderCombinedFooter creates the standard bottom block: Help | Status | Profile | Path.
// helpText is the raw (unstyled) active-mode help line; pass "" to skip it.
// It is word-wrapped (WrapText) before styling, one Render() call per
// wrapped line, so it degrades to multiple lines on narrow terminals
// instead of overflowing — wrapping before styling matters here, since a
// single Render() over the whole line carries one ANSI start/reset pair
// that a post-hoc wrap could split across lines. The NET/STATUS/THEME line
// is NOT wrapped: m.net.traffic/m.net.online already carry ANSI color
// codes and can contain internal spaces (e.g. "↓ 1.2kb/s ↑ 3.4kb/s"), so
// word-wrapping that joined line risks splitting an ANSI start/reset pair
// across lines and bleeding color into unrelated text — it's truncated
// instead.
func (m Model) renderCombinedFooter(helpText string) string {
	availWidth := m.termWidth - LayoutSideMargin

	var help string
	if helpText != "" {
		lines := WrapText(helpText, availWidth)
		styledLines := make([]string, len(lines))
		for i, line := range lines {
			styledLines[i] = m.styles.Help.Render(line)
		}
		help = strings.Join(styledLines, "\n")
	}

	netLabel := lipgloss.NewStyle().Render("NET: ")
	netText := netLabel + m.net.traffic
	statusText := fmt.Sprintf("STATUS: %s", m.net.online)
	themeText := "THEME: "
	themeName := m.styles.ThemeName.Render(m.Config.Theme)
	separator := m.styles.Help.Render(" | ")

	statusLine := netText + separator + statusText + separator + themeText + themeName
	networkStatusBar := lipgloss.NewStyle().PaddingTop(1).Render(truncateText(statusLine, availWidth))

	profileBar := m.renderProfileBar()
	pathBar := m.path.RenderPathBar(m.mode == pathMode, m.styles)
	childDirs := m.path.RenderChildDirs(m.mode, m.styles)

	items := []string{}
	if help != "" {
		items = append(items, help)
	}
	items = append(items, networkStatusBar, profileBar, pathBar, childDirs)

	return lipgloss.JoinVertical(lipgloss.Left, items...)
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
	maxRowNumWidth := len(fmt.Sprintf("%d", max(totalRows-1, 0)))

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
