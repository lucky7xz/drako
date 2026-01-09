package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// renderSizeOverlay shows a centered panel with current and required dimensions
func (m Model) renderSizeOverlay(reqW, reqH int) string {
	title := titleStyle.Render("Terminal too small")
	minScalePct := 60
	info := helpStyle.Render(
		fmt.Sprintf("Current: %dx%d  |  Required (at %d%%): %dx%d",
			m.termWidth, m.termHeight, minScalePct, reqW, reqH),
	)
	hint := helpStyle.Render("Hint: maximize the window or lower grid size (x,y)")

	box := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		info,
		hint,
	)

	overlay := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF5F5F")).
		Align(lipgloss.Center).
		Render(box)

	return lipgloss.Place(
		m.termWidth, m.termHeight,
		lipgloss.Center, lipgloss.Center,
		overlay,
	)
}

func (m Model) viewDropdownMode() string {
	// Render the base grid view
	layout := CalculateLayout(m.termWidth, m.termHeight, m.Config)
	header := ""
	if layout.ShowHeader {
		header = renderHeaderArt(m.spinner.View())
	}
	grid := m.renderGrid()
	mainContent := lipgloss.JoinVertical(lipgloss.Center, header, grid)

	helpText := "Dropdown Mode | ↑/↓/ws: Select, Enter: Execute, Esc/q: Cancel"
	help := helpStyle.Render(helpText)

	// Adjust footer rendering for layout?
	// The original code rendered StatusBar, ProfileBar, PathBar etc explicitly here.
	// Since we are refactoring, let's keep it close to original for now, using helpers if possible.

	// Re-implementing the original comprehensive footer block for now until we unify `renderWithLayout`

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
	pathBar := m.path.RenderPathBar(false)
	childDirs := m.path.RenderChildDirs(m.mode)

	var footer string
	if layout.ShowFooter {
		footer = lipgloss.JoinVertical(
			lipgloss.Left,
			help,
			networkStatusBar,
			profileBar,
			pathBar,
			childDirs,
		)
	}

	finalContent := lipgloss.JoinVertical(
		lipgloss.Center,
		mainContent,
		footer,
	)

	// Render dropdown popup
	dropdownPopup := m.renderDropdownPopup()

	// Place the dropdown in the center of the screen
	popupOverlay := lipgloss.Place(m.termWidth, m.termHeight,
		lipgloss.Center, lipgloss.Center,
		dropdownPopup,
	)

	return appStyle.Render(
		lipgloss.Place(m.termWidth, m.termHeight,
			lipgloss.Center, lipgloss.Center,
			finalContent+"\n"+popupOverlay,
		),
	)
}

func (m Model) renderDropdownPopup() string {
	// Ensure every segment renders with the popup background to avoid black gaps
	bg := dropdownPopupStyle.GetBackground()
	bgFill := lipgloss.NewStyle().Background(bg)
	cursorSel := selectedCursorStyle.Background(bg)
	textNorm := itemStyle.Background(bg)
	textSel := selectedItemStyle.Background(bg)
	gap := lipgloss.NewStyle().Background(bg)

	// Build lines and compute max width
	var raw []string
	maxW := 0
	for i, item := range m.dropdownItems {
		var line string
		if i == m.dropdownSelectedIdx {
			line = cursorSel.Render("► ") + textSel.Render(item.Name)
		} else {
			line = gap.Render("  ") + textNorm.Render(item.Name)
		}
		raw = append(raw, line)
		if w := lipgloss.Width(line); w > maxW {
			maxW = w
		}
	}
	if maxW == 0 {
		maxW = 1
	}

	// Right-pad each line with background-colored spaces to equal width
	var lines []string
	for _, line := range raw {
		pad := maxW - lipgloss.Width(line)
		if pad < 0 {
			pad = 0
		}
		padded := line + bgFill.Render(strings.Repeat(" ", pad))
		lines = append(lines, padded)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return dropdownPopupStyle.Render(content)
}

func (m Model) viewLockedMode() string {
	// Calculate time since last activity
	elapsed := time.Since(m.lastActivityTime)
	elapsedMins := int(elapsed.Minutes())

	if elapsedMins < 0 {
		elapsedMins = 0
	}

	goal := m.lockPumpGoal
	if goal <= 0 {
		goal = defaultLockPumpGoal
	}

	barWidth := 24
	progress := m.lockProgress
	if progress < 0 {
		progress = 0
	}
	if progress > goal {
		progress = goal
	}

	filled := progress * barWidth / goal
	if filled > barWidth {
		filled = barWidth
	}

	bar := "[" + strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled) + "]"

	lockIcon := "🔒"
	title := titleStyle.Render("Session Locked")
	timeInfo := helpStyle.Render(fmt.Sprintf("Idle for %d minute(s)", elapsedMins))
	instructions := helpStyle.Render("Pump ← → (A/D or H/L) to fill the slider and unlock")
	progressLabel := helpStyle.Render(fmt.Sprintf("%d / %d pumps", m.lockProgress, goal))
	quitHint := helpStyle.Render("Press Ctrl+C to quit")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		"",
		lockIcon,
		"",
		title,
		"",
		timeInfo,
		"",
		instructions,
		"",
		progressLabel,
		bar,
		"",
		quitHint,
	)

	// Add a border box around the lock screen
	box := lipgloss.NewStyle().
		Padding(2, 4).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FFA500")).
		Align(lipgloss.Center).
		Render(content)

	footer := lipgloss.NewStyle().
		Width(m.termWidth).
		Align(lipgloss.Center).
		Render(m.renderFooter())

	body := lipgloss.JoinVertical(
		lipgloss.Center,
		box,
		footer,
	)

	return appStyle.Render(
		lipgloss.Place(m.termWidth, m.termHeight,
			lipgloss.Center, lipgloss.Center,
			body,
		),
	)
}

func (m Model) viewInfoMode() string {
	layout := CalculateLayout(m.termWidth, m.termHeight, m.Config)
	header := ""
	if layout.ShowHeader {
		header = renderHeaderArt(m.spinner.View())
	}

	// Build info lines with same background rules to avoid black gaps
	bg := dropdownPopupStyle.GetBackground()
	bgFill := lipgloss.NewStyle().Background(bg)
	titleStyleLocal := titleStyle.Background(bg)
	labelStyle := helpStyle.Background(bg)
	valueStyle := itemStyle.Background(bg)

	// Wrap width for info popup content
	wrapWidth := m.termWidth - 10
	if wrapWidth > 80 {
		wrapWidth = 80
	}
	if wrapWidth < 20 {
		wrapWidth = 20
	}

	var raw []string

	// Safety check if activeDetail is nil (should not happen in infoMode ideally)
	if m.activeDetail == nil {
		return appStyle.Render(lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center, "Error: No detail state"))
	}

	if strings.TrimSpace(m.activeDetail.Title) != "" {
		raw = append(raw, titleStyleLocal.Render(m.activeDetail.Title))
	}

	if strings.TrimSpace(m.activeDetail.Value) != "" {
		raw = append(raw, "")
		label := "Value:"
		if m.activeDetail.KeyLabel != "" {
			label = m.activeDetail.KeyLabel + ":"
		}
		raw = append(raw, labelStyle.Render(label))
		for _, ln := range WrapText(m.activeDetail.Value, wrapWidth) {
			raw = append(raw, valueStyle.Render(ln))
		}
	}

	if strings.TrimSpace(m.activeDetail.Description) != "" {
		raw = append(raw, "")
		raw = append(raw, labelStyle.Render("Description:"))
		for _, ln := range WrapText(m.activeDetail.Description, wrapWidth) {
			raw = append(raw, valueStyle.Render(ln))
		}
	}

	if len(m.activeDetail.Meta) > 0 {
		raw = append(raw, "")
		for _, meta := range m.activeDetail.Meta {
			raw = append(raw, labelStyle.Render(meta.Label+": ")+valueStyle.Render(meta.Value))
		}
	}

	raw = append(raw, "")
	raw = append(raw, helpStyle.Render("Press y to copy command/details to clipboard • any key to close"))

	// Compute max width and pad
	maxW := 0
	for _, line := range raw {
		if w := lipgloss.Width(line); w > maxW {
			maxW = w
		}
	}
	if maxW == 0 {
		maxW = 1
	}

	var lines []string
	for _, line := range raw {
		pad := maxW - lipgloss.Width(line)
		if pad < 0 {
			pad = 0
		}
		lines = append(lines, line+bgFill.Render(strings.Repeat(" ", pad)))
	}

	popup := dropdownPopupStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	content := lipgloss.JoinVertical(lipgloss.Center, header, popup)
	return appStyle.Render(lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center, content))
}
