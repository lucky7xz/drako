package ui

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strings"
	"time"

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

	return appStyle.Render(
		lipgloss.Place(m.termWidth, m.termHeight,
			lipgloss.Center, lipgloss.Center,
			finalContent,
		),
	)
}

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

func (m Model) renderGrid() string {
	maxContentWidth := 0
	for _, row := range m.grid {
		for _, cell := range row {
			contentWidth := lipgloss.Width(cell)
			if contentWidth > maxContentWidth {
				maxContentWidth = contentWidth
			}
		}
	}

	if maxContentWidth > GridMaxTextWidth {
		maxContentWidth = GridMaxTextWidth
	}

	// Total width must account for content, padding (1+1), and border (1+1).
	totalCellWidth := maxContentWidth + 4

	// --- Build Header ---
	var headerParts []string
	if len(m.grid) > 0 {
		for c := 0; c < len(m.grid[0]); c++ {
			headerLabel := fmt.Sprintf("[%s]", columnToLetter(c))
			styledLabel := titleStyle.Render(headerLabel)

			// Let lipgloss handle the centering of the styled text.
			headerContentWidth := totalCellWidth - 2 // for ┌ and ┐
			headerCellStyle := lipgloss.NewStyle().
				Width(headerContentWidth).
				Align(lipgloss.Center)

			headerContent := headerCellStyle.Render(styledLabel)
			headerWithLines := strings.ReplaceAll(headerContent, " ", "─")

			headerPart := fmt.Sprintf("┌%s┐", headerWithLines)
			headerParts = append(headerParts, headerPart)
		}
	}
	fullHeader := lipgloss.JoinHorizontal(lipgloss.Left, headerParts...)

	// --- Build Grid ---
	var renderedRows []string
	for r, row := range m.grid {
		var renderedCells []string
		for c, cell := range row {
			var style lipgloss.Style
			if m.mode == gridMode && r == m.cursorRow && c == m.cursorCol {
				style = selectedCellStyle
			} else {
				style = cellStyle
			}

			truncatedContent := truncateText(cell, maxContentWidth)

			// The cell style itself has padding, so we just need to render the content.
			paddedContent := lipgloss.NewStyle().
				Width(maxContentWidth).
				Align(lipgloss.Left).
				Render(truncatedContent)

			renderedCell := style.Render(paddedContent)
			renderedCells = append(renderedCells, renderedCell)
		}
		renderedRows = append(renderedRows, lipgloss.JoinHorizontal(lipgloss.Top, renderedCells...))
	}

	// --- Add Row Indicators and Final Assembly ---
	var finalRows []string
	// Calculate the padding needed for the largest row number.
	maxRowNumWidth := len(fmt.Sprintf("%d", len(renderedRows)-1))
	rowPrefix := strings.Repeat(" ", maxRowNumWidth+1) // Padding for continuation lines: "[0] ❭ "
	for i, row := range renderedRows {
		rowNum := fmt.Sprintf("%*d❭", maxRowNumWidth, i)
		// Split the row into lines and add proper prefix to each line
		lines := strings.Split(row, "\n")
		for j, line := range lines {
			if j == 0 {
				lines[j] = rowNum + line
			} else {
				lines[j] = rowPrefix + line
			}
		}
		finalRows = append(finalRows, strings.Join(lines, "\n"))
	}

	// Create padding for the header to align it with the grid body.
	headerPadding := strings.Repeat(" ", maxRowNumWidth+1) // +5 for "[0] ❭ "
	paddedHeader := headerPadding + fullHeader

	gridBody := lipgloss.JoinVertical(lipgloss.Center, finalRows...)

	return lipgloss.JoinVertical(lipgloss.Left, paddedHeader, gridBody)
}

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

func (m Model) renderProfileBar() string {
	hostname, _ := os.Hostname()
	currUser, _ := user.Current()
	username := "unknown"
	if currUser != nil {
		username = currUser.Username
	} else {
		// Fallback to environment variable if user lookup fails
		username = os.Getenv("USER")
	}

	// Clean up username if it contains full path (rare, but happens on some systems)
	if idx := strings.LastIndex(username, "\\"); idx != -1 {
		username = username[idx+1:]
	}

	osArch := fmt.Sprintf("(%s/%s)", runtime.GOOS, runtime.GOARCH)

	// Format: HOST: user@hostname (linux/amd64) |
	hostLabel := "HOST: " + username + "@" + hostname + " " + osArch + helpStyle.Render(" | ")

	profileLabel := lipgloss.NewStyle().Render("PROFILE: ")
	segments := []string{hostLabel + profileLabel + m.activeProfileName()}

	if m.pivotProfileName != "" {
		label := fmt.Sprintf("🔒 %s", m.pivotProfileName)
		segments = append(segments, lockBadgeStyle.Render(label))
	}

	if m.profileStatusMessage != "" {
		style := statusNegativeStyle
		if m.profileStatusPositive {
			style = statusPositiveStyle
		}
		segments = append(segments, style.Render(m.profileStatusMessage))
	}

	//segments = append(segments, helpStyle.Render(" \n\n		Press i for Inventory"))

	return lipgloss.NewStyle().PaddingTop(1).Render(lipgloss.JoinHorizontal(lipgloss.Left, segments...))
}

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
	s.WriteString(titleStyle.Render("Inventory Management") + "\n\n")

	// Render Visible Grid
	s.WriteString(listHeaderStyle.Render("Equipped Profiles") + "\n")
	s.WriteString(m.renderInventoryGrid(m.inventory.visible, 0))
	s.WriteString("\n\n")

	// Render Inventory Grid
	s.WriteString(listHeaderStyle.Render("Inventory") + "\n")
	s.WriteString(m.renderInventoryGrid(m.inventory.inventory, 1))
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
	if m.inventory.heldItem != nil {
		heldItemStatus = helpStyle.Render("Holding: ") + selectedItemStyle.Render(*m.inventory.heldItem)
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

func columnToLetter(col int) string {
	if col < 0 || col > 25 {
		return "?"
	}
	return string(rune('A' + col))
}

func (m Model) renderFooter() string {
	return footerStyle.Render("[ github.com/lucky7xz | {chronyx}.xyz ]")
}

func (m Model) renderProfileCounter() string {
	y := len(m.profiles)
	if y > 9 {
		y = 9
	}
	x := m.activeProfileIndex + 1
	if x > 9 {
		x = 9
	}
	counter := fmt.Sprintf("< %d / %d >", x, y)
	return titleStyle.Render(counter)
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
	raw = append(raw, helpStyle.Render("Press y to copy details • any key to close"))

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
