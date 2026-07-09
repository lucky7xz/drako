package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

	layout := CalculateLayout(m.termWidth, m.termHeight, m.Config)

	header := ""
	if layout.ShowHeader {
		header = m.styles.renderHeaderArt(m.spinner.View())
	}
	counter := m.renderProfileCounter()

	var helpText string
	switch m.mode {
	case pathMode:
		helpText = "Path Mode | ←/→/ad: Select, ↓/s: Children, Enter: cd, e: Search, q/Esc: Back"
	case childMode:
		helpText = "Child Mode | ↑/↓/ws: Select, Enter: cd, e: Search, q/Esc: Back"
	default:
		helpText = "Grid Mode | Enter: Select, e: Explain, Tab: Path, r: Start-Lock, i: Inventory"
	}
	help := m.styles.Help.Render(helpText)

	footer := m.renderCombinedFooter(help)

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

// renderCombinedFooter creates the standard bottom block: Help | Status | Profile | Path
// Pass empty help string to skip help (e.g. if help is rendered differently)
func (m Model) renderCombinedFooter(helpRendered string) string {
	netLabel := lipgloss.NewStyle().Render("NET: ")
	netText := netLabel + m.net.traffic
	statusText := fmt.Sprintf("STATUS: %s", m.net.online)
	themeText := "THEME: "
	themeName := m.styles.ThemeName.Render(m.Config.Theme)
	separator := m.styles.Help.Render(" | ")

	networkStatusBar := lipgloss.NewStyle().PaddingTop(1).Render(
		lipgloss.JoinHorizontal(lipgloss.Left,
			netText,
			separator,
			statusText,
			separator,
			themeText,
			themeName,
		),
	)
	profileBar := m.renderProfileBar()
	pathBar := m.path.RenderPathBar(m.mode == pathMode, m.styles)
	childDirs := m.path.RenderChildDirs(m.mode, m.styles)

	items := []string{}
	if helpRendered != "" {
		items = append(items, helpRendered)
	}
	items = append(items, networkStatusBar, profileBar, pathBar, childDirs)

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

// truncateText clips a string to a max visual width
func truncateText(s string, maxLength int) string {
	if lipgloss.Width(s) <= maxLength {
		return s
	}

	var truncated strings.Builder
	var currentWidth int
	for _, r := range s {
		runeWidth := lipgloss.Width(string(r))
		if currentWidth+runeWidth+3 > maxLength {
			break
		}
		truncated.WriteRune(r)
		currentWidth += runeWidth
	}

	return truncated.String() + "..."
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
	reqW := appStyle.GetHorizontalMargins() + maxRowNumWidth + 1 + minCols*cellFootprint(m.gridNav.grid)
	if totalCols > minCols {
		reqW += 2 // the appended ▸ marker
	}
	return m.termWidth < reqW || m.termHeight < reqH, reqW, reqH
}
