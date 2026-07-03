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
	for _, row := range m.gridNav.grid {
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
	if len(m.gridNav.grid) > 0 {
		for c := 0; c < len(m.gridNav.grid[0]); c++ {
			headerLabel := fmt.Sprintf("[%s]", columnToLetter(c))
			styledLabel := m.styles.Title.Render(headerLabel)

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
	for r, row := range m.gridNav.grid {
		var renderedCells []string
		for c, cell := range row {
			var style lipgloss.Style
			if m.mode == gridMode && r == m.gridNav.cursorRow && c == m.gridNav.cursorCol {
				style = m.styles.SelectedCell
			} else {
				style = m.styles.Cell
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

func columnToLetter(col int) string {
	if col < 0 || col > 25 {
		return "?"
	}
	return string(rune('A' + col))
}

func (m Model) renderProfileCounter() string {
	y := len(m.profile.profiles)
	if y > 9 {
		y = 9
	}
	x := m.profile.activeIndex + 1
	if x > 9 {
		x = 9
	}
	counter := fmt.Sprintf("< %d / %d >", x, y)
	return m.styles.Title.Render(counter)
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
	hostLabel := "HOST: " + username + "@" + hostname + " " + osArch + m.styles.Help.Render(" | ")

	profileLabel := lipgloss.NewStyle().Render("PROFILE: ")
	segments := []string{hostLabel + profileLabel + m.profile.activeName() + m.styles.Help.Render(" | ")}

	if m.profile.pivotName != "" {
		label := fmt.Sprintf("🔒 %s", m.profile.pivotName)
		segments = append(segments, m.styles.LockBadge.Render(label))
	}

	if m.GlassrootMode {
		// Glassroot indicator next to lock/profile
		glassIndicator := lipgloss.NewStyle().Foreground(lipgloss.Color("#A8E6CF")).Render("🧊 G-ROOT")
		segments = append(segments, glassIndicator)
	}

	if m.profile.statusMessage != "" {
		style := m.styles.StatusNegative
		if m.profile.statusPositive {
			style = m.styles.StatusPositive
		}
		segments = append(segments, style.Render(m.profile.statusMessage))
	}

	return lipgloss.NewStyle().PaddingTop(1).Render(lipgloss.JoinHorizontal(lipgloss.Left, segments...))
}
