package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// renderSizeOverlay shows a centered panel with current and required dimensions
func (m Model) renderSizeOverlay(reqW, reqH int) string {
	title := m.styles.Title.Render("Terminal too small")
	info := m.styles.Help.Render(
		fmt.Sprintf("Current: %dx%d  |  Required: %dx%d",
			m.termWidth, m.termHeight, reqW, reqH),
	)
	hint := m.styles.Help.Render("Hint: maximize the window or lower grid size (x,y)")

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

// padLinesToWidth right-pads every line to the widest line's width with
// background-colored spaces, so popup contents form a solid block.
func padLinesToWidth(raw []string, bgFill lipgloss.Style) []string {
	maxW := 0
	for _, line := range raw {
		if w := lipgloss.Width(line); w > maxW {
			maxW = w
		}
	}
	if maxW == 0 {
		maxW = 1
	}
	lines := make([]string, len(raw))
	for i, line := range raw {
		pad := maxW - lipgloss.Width(line)
		if pad < 0 {
			pad = 0
		}
		lines[i] = line + bgFill.Render(strings.Repeat(" ", pad))
	}
	return lines
}

func (m Model) viewDropdownMode() string {
	// Render the base grid view
	layout := CalculateLayout(m.termWidth, m.termHeight, m.Config)
	header := ""
	if layout.ShowHeader {
		header = m.styles.renderHeaderArt(m.spinner.View())
	}
	helpText := "Dropdown Mode | ↑/↓/ws: Select, Enter: Execute, Esc/q: Cancel"
	help := m.styles.Help.Render(helpText)

	var footer string
	if layout.ShowFooter {
		footer = m.renderCombinedFooter(help)
	}

	grid := m.renderGrid(gridRowBudget(m.termHeight, header, footer))
	mainContent := lipgloss.JoinVertical(lipgloss.Center, header, grid)

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
	bg := m.styles.DropdownPopup.GetBackground()
	bgFill := lipgloss.NewStyle().Background(bg)
	cursorSel := m.styles.SelectedCursor.Background(bg)
	textNorm := m.styles.Item.Background(bg)
	textSel := m.styles.SelectedItem.Background(bg)
	gap := lipgloss.NewStyle().Background(bg)

	// Build lines, then right-pad them into a solid block.
	var raw []string
	for i, item := range m.dropdown.items {
		if i == m.dropdown.selectedIdx {
			raw = append(raw, cursorSel.Render("► ")+textSel.Render(item.Name))
		} else {
			raw = append(raw, gap.Render("  ")+textNorm.Render(item.Name))
		}
	}
	lines := padLinesToWidth(raw, bgFill)

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return m.styles.DropdownPopup.Render(content)
}

func (m Model) viewLockedMode() string {
	// Calculate time since last activity
	elapsed := time.Since(m.lock.lastActivity)
	elapsedMins := int(elapsed.Minutes())

	if elapsedMins < 0 {
		elapsedMins = 0
	}

	goal := m.lock.pumpGoal
	if goal <= 0 {
		goal = defaultLockPumpGoal
	}

	barWidth := 24
	progress := m.lock.progress
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
	title := m.styles.Title.Render("Session Locked")
	timeInfo := m.styles.Help.Render(fmt.Sprintf("Idle for %d minute(s)", elapsedMins))
	instructions := m.styles.Help.Render("Pump ← → (A/D or H/L) to fill the slider and unlock")
	progressLabel := m.styles.Help.Render(fmt.Sprintf("%d / %d pumps", m.lock.progress, goal))
	quitHint := m.styles.Help.Render("Press Ctrl+C to quit")

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
		header = m.styles.renderHeaderArt(m.spinner.View())
	}

	// Build info lines with same background rules to avoid black gaps
	bg := m.styles.DropdownPopup.GetBackground()
	bgFill := lipgloss.NewStyle().Background(bg)
	titleStyleLocal := m.styles.Title.Background(bg)
	labelStyle := m.styles.Help.Background(bg)
	valueStyle := m.styles.Item.Background(bg)

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
	raw = append(raw, m.styles.Help.Render("Press y to copy command/details to clipboard • any key to close"))

	lines := padLinesToWidth(raw, bgFill)

	popup := m.styles.DropdownPopup.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	content := lipgloss.JoinVertical(lipgloss.Center, header, popup)
	return appStyle.Render(lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center, content))
}
