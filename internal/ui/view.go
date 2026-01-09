package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/config"
)

func (m Model) View() string {
	if m.termWidth == 0 {
		return "Initializing..."
	}

	// If terminal is too small even at min_scale, show blocking overlay
	if tooSmall, reqW, reqH := IsBelowMinimum(m.termWidth, m.termHeight, m.Config); tooSmall {
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
		header = renderHeaderArt(m.spinner.View())
	}
	counter := m.renderProfileCounter()
	grid := m.renderGrid()
	mainContent := lipgloss.JoinVertical(lipgloss.Center, header, counter, grid)

	var helpText string
	switch m.mode {
	case pathMode:
		helpText = "Path Mode | ←/→/ad: Select, ↓/s: Children, Enter: cd, e: Search, q/Esc: Back"
	case childMode:
		helpText = "Child Mode | ↑/↓/ws: Select, Enter: cd, e: Search, q/Esc: Back"
	default:
		helpText = "Grid Mode | Enter: Select, e: Explain, Tab: Path, r: Start-Lock, i: Inventory"
	}
	help := helpStyle.Render(helpText)

	footer := m.renderCombinedFooter(help)

	// Respect layout.ShowFooter
	if !layout.ShowFooter {
		footer = ""
	}

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
	netText := netLabel + m.traffic
	statusText := fmt.Sprintf("STATUS: %s", m.onlineStatus)
	themeText := "THEME: "
	themeName := themeNameStyle.Render(m.Config.Theme)
	separator := helpStyle.Render(" | ")

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
	pathBar := m.path.RenderPathBar(m.mode == pathMode)
	childDirs := m.path.RenderChildDirs(m.mode)

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
	return footerStyle.Render("[ github.com/lucky7xz | {chronyx}.xyz ]")
}

// --- Minimum size helpers (computed from grid size and min_scale) ---

// CalculateRequiredSize computes the minimum terminal dimensions needed
// for the current grid at 100% scale
func CalculateRequiredSize(cfg config.Config) (minWidth, minHeight int) {
	// Grid area
	gridWidth := cfg.X * GridCellWidth
	gridHeight := cfg.Y * GridCellHeight

	minWidth = gridWidth + LayoutSideMargin
	// Header is now optional, so minimum height doesn't strictly require it
	// But let's keep it in the calculation for optimal experience,
	// or reduce it?
	// If we want to support small screens, we should say min height is grid + footer.
	// Let's be permissive.
	minHeight = gridHeight + LayoutStatusHeight + LayoutVertPadding
	return minWidth, minHeight
}

// RequiredSizeAtScale estimates the space needed at a given scale factor
func RequiredSizeAtScale(cfg config.Config, scale float64) (int, int) {
	w, h := CalculateRequiredSize(cfg)
	return int(float64(w) * scale), int(float64(h) * scale)
}

// IsBelowMinimum returns whether the terminal is too small even at min_scale,
// and returns the required width/height at min_scale for display.
func IsBelowMinimum(termWidth, termHeight int, cfg config.Config) (bool, int, int) {
	// Default minimum scale is 60% (triggers a bit sooner)
	scale := 0.60
	reqW, reqH := RequiredSizeAtScale(cfg, scale)
	if termWidth < reqW || termHeight < reqH {
		return true, reqW, reqH
	}
	return false, reqW, reqH
}
