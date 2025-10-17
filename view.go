package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	if m.termWidth == 0 {
		return "Initializing..."
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

	header := renderHeaderArt(m.spinner.View())
	counter := m.renderProfileCounter()
	grid := m.renderGrid()
	mainContent := lipgloss.JoinVertical(lipgloss.Center, header, counter, grid)

	var helpText string
	switch m.mode {
	case pathMode:
		helpText = "Path Mode | ←/→/ad: Select, ↓/s: Children, Enter: cd, Tab: Grid, r: Start-Lock"
	case childMode:
		helpText = "Child Mode | ↑/↓/ws: Select, Enter: cd, Tab: Grid, r: Start-Lock"
	default:
		helpText = "Grid Mode | Enter: Select, e: Explain, Tab: Path, r: Start-Lock, i: Inventory"
	}
	help := helpStyle.Render(helpText)

	netLabel := lipgloss.NewStyle().Render("NET: ")
	netText := netLabel + m.traffic
	statusText := fmt.Sprintf("STATUS: %s", m.onlineStatus)
	themeText := "THEME: "
	themeName := themeNameStyle.Render(m.config.Theme)
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
	pathBar := m.renderPathBar()
	childDirs := m.renderChildDirs()

	appFooter := m.renderFooter()

	footer := lipgloss.JoinVertical(
		lipgloss.Left,
		help,
		networkStatusBar,
		profileBar,
		pathBar,
		childDirs,
	)

	// Center the footer separately
	centeredFooter := lipgloss.NewStyle().
		Width(m.termWidth).
		Align(lipgloss.Center).
		Render(appFooter)

	finalContent := lipgloss.JoinVertical(
		lipgloss.Center,
		mainContent,
		footer,
		centeredFooter,
	)

	return appStyle.Render(
		lipgloss.Place(m.termWidth, m.termHeight,
			lipgloss.Center, lipgloss.Center,
			finalContent,
		),
	)
}

func (m model) renderGrid() string {
	const maxTextWidth = 25 // Max width for the text inside.

	maxContentWidth := 0
	for _, row := range m.grid {
		for _, cell := range row {
			contentWidth := lipgloss.Width(cell)
			if contentWidth > maxContentWidth {
				maxContentWidth = contentWidth
			}
		}
	}

	if maxContentWidth > maxTextWidth {
		maxContentWidth = maxTextWidth
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

func (m model) renderProfileBar() string {
	profileLabel := lipgloss.NewStyle().Render("PROFILE: ")
	segments := []string{profileLabel + m.activeProfileName()}

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

func (m model) renderPathBar() string {
	var renderedParts []string
	for i, component := range m.pathComponents {
		var style lipgloss.Style
		if m.mode == pathMode && i == m.selectedPathIndex {
			style = selectedPathStyle
		} else {
			style = pathStyle
		}
		renderedParts = append(renderedParts, style.Render(component))
	}

	separator := pathSeparatorStyle.Render("/")
	return statusBarStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(renderedParts, separator)))
}

func (m model) renderChildDirs() string {
	if m.mode != childMode && m.mode != pathMode {
		return ""
	}
	if m.childDirsError != nil {
		return offlineStyle.Render("  [cannot read directory: permission denied or path invalid]")
	}
	if len(m.childDirs) == 0 {
		return helpStyle.Render("  [no sub-directories]")
	}

	var rows []string
	for i, dir := range m.childDirs {
		if m.mode == childMode && i == m.selectedChildIndex {
			rows = append(rows, selectedChildDirStyle.Render("› "+dir))
		} else {
			rows = append(rows, childDirStyle.Render("  "+dir))
		}
	}

	maxVisible := 5
	start := 0
	if m.mode == childMode && m.selectedChildIndex >= maxVisible {
		start = m.selectedChildIndex - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(rows) {
		end = len(rows)
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows[start:end]...)
}

func (m model) viewInventoryMode() string {
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

func (m model) renderInventoryGrid(profiles []string, listID int) string {
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

	return lipgloss.JoinHorizontal(lipgloss.Left, cells...)
}

func columnToLetter(col int) string {
	if col < 0 || col > 25 {
		return "?"
	}
	return string('A' + col)
}

func (m model) renderFooter() string {
	return footerStyle.Render("[ github.com/lucky7xz | {yx}.xyz ]")
}

func (m model) renderProfileCounter() string {
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

func (m model) viewDropdownMode() string {
	// Render the base grid view
	header := renderHeaderArt(m.spinner.View())
	grid := m.renderGrid()
	mainContent := lipgloss.JoinVertical(lipgloss.Center, header, grid)

	var helpText string
	helpText = "Dropdown Mode | ↑/↓/ws: Select, Enter: Execute, Esc/q: Cancel"
	help := helpStyle.Render(helpText)

	netLabel := lipgloss.NewStyle().Render("NET: ")
	netText := netLabel + m.traffic
	statusText := fmt.Sprintf("STATUS: %s", m.onlineStatus)
	themeText := "THEME: "
	themeName := themeNameStyle.Render(m.config.Theme)
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
	pathBar := m.renderPathBar()
	childDirs := m.renderChildDirs()
	appFooter := m.renderFooter()

	footer := lipgloss.JoinVertical(
		lipgloss.Left,
		help,
		networkStatusBar,
		profileBar,
		pathBar,
		childDirs,
	)

	// Center the footer separately
	centeredFooter := lipgloss.NewStyle().
		Width(m.termWidth).
		Align(lipgloss.Center).
		Render(appFooter)

	finalContent := lipgloss.JoinVertical(
		lipgloss.Center,
		mainContent,
		footer,
		centeredFooter,
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

func (m model) renderDropdownPopup() string {
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

func (m model) viewInfoMode() string {
	header := renderHeaderArt(m.spinner.View())

	// Build info lines with same background rules to avoid black gaps
	bg := dropdownPopupStyle.GetBackground()
	bgFill := lipgloss.NewStyle().Background(bg)
	titleStyleLocal := titleStyle.Background(bg)
	labelStyle := helpStyle.Background(bg)
	valueStyle := itemStyle.Background(bg)

	// Wrap width for info popup content
	wrapWidth := m.termWidth - 10
	if wrapWidth > 80 { wrapWidth = 80 }
	if wrapWidth < 20 { wrapWidth = 20 }

	// Local helpers to wrap text by visual width
	wrapWord := func(word string, width int) []string {
		if width <= 0 { return []string{word} }
		var out []string
		var b strings.Builder
		cur := 0
		for _, r := range word {
			w := lipgloss.Width(string(r))
			if cur+w > width && b.Len() > 0 {
				out = append(out, b.String())
				b.Reset()
				cur = 0
			}
			b.WriteRune(r)
			cur += w
		}
		if b.Len() > 0 { out = append(out, b.String()) }
		return out
	}
	wrapLine := func(s string, width int) []string {
		if width <= 0 { return []string{s} }
		fields := strings.Fields(s)
		if len(fields) == 0 { return []string{""} }
		var lines []string
		var line string
		for _, word := range fields {
			ww := lipgloss.Width(word)
			if line == "" {
				if ww <= width {
					line = word
				} else {
					lines = append(lines, wrapWord(word, width)...)
					line = ""
				}
				continue
			}
			if lipgloss.Width(line)+1+ww <= width {
				line += " " + word
			} else {
				lines = append(lines, line)
				if ww <= width {
					line = word
				} else {
					lines = append(lines, wrapWord(word, width)...)
					line = ""
				}
			}
		}
		if line != "" { lines = append(lines, line) }
		return lines
	}

	var raw []string
	if strings.TrimSpace(m.infoTitle) != "" {
		raw = append(raw, titleStyleLocal.Render(m.infoTitle))
	}
	if strings.TrimSpace(m.infoCommand) != "" {
		raw = append(raw, "")
		raw = append(raw, labelStyle.Render("Command:"))
		for _, para := range strings.Split(m.infoCommand, "\n") {
			para = strings.TrimSpace(para)
			if para == "" { raw = append(raw, valueStyle.Render("")); continue }
			for _, ln := range wrapLine(para, wrapWidth) {
				raw = append(raw, valueStyle.Render(ln))
			}
		}
	}
	if strings.TrimSpace(m.infoDescription) != "" {
		raw = append(raw, "")
		raw = append(raw, labelStyle.Render("Description:"))
		for _, para := range strings.Split(m.infoDescription, "\n") {
			para = strings.TrimSpace(para)
			if para == "" { raw = append(raw, valueStyle.Render("")); continue }
			for _, ln := range wrapLine(para, wrapWidth) {
				raw = append(raw, valueStyle.Render(ln))
			}
		}
	}
	raw = append(raw, "")
	raw = append(raw, labelStyle.Render("Exec:")+" "+valueStyle.Render(m.infoExecMode))
	raw = append(raw, labelStyle.Render("Auto-close:")+" "+valueStyle.Render(fmt.Sprintf("%v", m.infoAutoClose)))
	raw = append(raw, labelStyle.Render("CWD:")+" "+valueStyle.Render(m.infoCwd))
	raw = append(raw, "")
	raw = append(raw, helpStyle.Render("Press y to copy command • any key to close"))

	// Compute max width and pad
	maxW := 0
	for _, line := range raw {
		if w := lipgloss.Width(line); w > maxW {
			maxW = w
		}
	}
	if maxW == 0 { maxW = 1 }

	var lines []string
	for _, line := range raw {
		pad := maxW - lipgloss.Width(line)
		if pad < 0 { pad = 0 }
		lines = append(lines, line+bgFill.Render(strings.Repeat(" ", pad)))
	}

	popup := dropdownPopupStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	content := lipgloss.JoinVertical(lipgloss.Center, header, popup)
	return appStyle.Render(lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center, content))
}
