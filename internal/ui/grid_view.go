package ui

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
	maxRowNumWidth := len(fmt.Sprintf("%d", len(renderedRows)))
	rowPrefix := strings.Repeat(" ", maxRowNumWidth+1) // Padding for continuation lines: "[0] ❭ "
	for i, row := range renderedRows {
		rowNum := fmt.Sprintf("%*d❭", maxRowNumWidth, i+1)
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

func columnToLetter(col int) string {
	if col < 0 || col > 25 {
		return "?"
	}
	return string(rune('A' + col))
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

	return lipgloss.NewStyle().PaddingTop(1).Render(lipgloss.JoinHorizontal(lipgloss.Left, segments...))
}
