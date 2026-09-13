package ui

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/profiles"
)

// renderGrid renders the grid into at most budgetLines terminal lines. When
// the grid is bigger than the terminal, a center-locked window around the
// cursor is shown, with indicators counting the clipped rows/columns.
func (m Model) renderGrid(budgetLines int) string {
	// --- Derive the visible window on both axes ---
	totalRows := len(m.gridNav.grid)
	totalCols := 0
	if totalRows > 0 {
		totalCols = len(m.gridNav.grid[0])
	}

	// Calculate the padding needed for the largest row number. Row labels
	// are 1-based, so the last row is labeled totalRows, not totalRows-1.
	maxRowNumWidth := len(fmt.Sprintf("%d", totalRows))

	widthBudget := m.termWidth - appStyle.GetHorizontalMargins() - (maxRowNumWidth + 1)
	totalCellWidth := fitFootprint(cellFootprint(m.gridNav.grid), widthBudget, totalCols)
	maxContentWidth := totalCellWidth - 4

	// reserve 2: the right scroll marker appended to the header line
	colWin := window(m.gridNav.cursorCol, totalCols, visibleCount(widthBudget, totalCellWidth, totalCols, 2))

	rowHeight := lipgloss.Height(m.styles.Cell.Render("x"))
	rowBudget := budgetLines - 1 // column header line
	// reserve 1: the marker line under the grid while scrolling
	rowWin := window(m.gridNav.cursorRow, totalRows, visibleCount(rowBudget, rowHeight, totalRows, 1))

	// --- Build Header ---
	var headerParts []string
	for c := colWin.start; c < colWin.end; c++ {
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
	fullHeader := lipgloss.JoinHorizontal(lipgloss.Left, headerParts...)

	// --- Build Grid (visible window only, absolute r/c throughout) ---
	rowPrefix := strings.Repeat(" ", maxRowNumWidth+1) // Padding for continuation lines: "[0] ❭ "
	var finalRows []string
	for r := rowWin.start; r < rowWin.end; r++ {
		var renderedCells []string
		for c := colWin.start; c < colWin.end; c++ {
			inGridOrBatch := m.mode == gridMode || m.mode == batchMode
			var style lipgloss.Style
			switch {
			case inGridOrBatch && r == m.gridNav.cursorRow && c == m.gridNav.cursorCol:
				style = m.styles.SelectedCell
			case inGridOrBatch && m.gridNav.timer != nil && c == m.gridNav.cursorCol:
				style = m.styles.ColumnPendingCell
			default:
				style = m.styles.Cell
			}

			cellName := m.gridNav.grid[r][c]
			truncatedContent := truncateText(m.batchCellPrefix(cellName)+cellName, maxContentWidth)

			// The cell style itself has padding, so we just need to render the content.
			paddedContent := lipgloss.NewStyle().
				Width(maxContentWidth).
				Align(lipgloss.Left).
				Render(truncatedContent)

			renderedCells = append(renderedCells, style.Render(paddedContent))
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, renderedCells...)

		rowNum := fmt.Sprintf("%*d❭", maxRowNumWidth, r+1)
		if (m.mode == gridMode || m.mode == batchMode) && m.gridNav.flashActive && r == m.gridNav.cursorRow {
			rowNum = m.styles.FlashedRowLabel.Render(rowNum)
		}
		// Split the row into lines and place the label on the vertical
		// middle line (row above/below get the blank prefix instead), so a
		// wrapped multi-line cell doesn't glue the number to its top line.
		lines := strings.Split(row, "\n")
		mid := len(lines) / 2
		for j, line := range lines {
			if j == mid {
				lines[j] = rowNum + line
			} else {
				lines[j] = rowPrefix + line
			}
		}
		finalRows = append(finalRows, strings.Join(lines, "\n"))
	}

	// --- Scroll markers ---
	// A marker points at hidden cells: it shows the directions the cursor
	// can still move toward. Both vertical directions share one line under
	// the grid; a blank slot keeps its size constant while scrolling.
	if rowWin.scrolling() {
		down, up := " ", " "
		if rowWin.hiddenAfter > 0 {
			down = "▾"
		}
		if rowWin.hiddenBefore > 0 {
			up = "▴"
		}
		finalRows = append(finalRows, m.styles.SelectedCursor.Render(down+" "+up))
	}

	// Create padding for the header to align it with the grid body; the
	// horizontal scroll markers flank this line.
	paddedHeader := rowPrefix + fullHeader
	if colWin.scrolling() {
		left := " "
		if colWin.hiddenBefore > 0 {
			left = m.styles.SelectedCursor.Render("◂")
		}
		right := ""
		if colWin.hiddenAfter > 0 {
			right = " " + m.styles.SelectedCursor.Render("▸")
		}
		paddedHeader = strings.Repeat(" ", maxRowNumWidth-1) + left + " " + fullHeader + right
	}

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
	total := len(m.profile.profiles)
	x := m.profile.activeIndex + 1
	// No profiles on disk means the rescue grid: "< 0 / 0 >".
	if m.profile.activeIndex < 0 || m.profile.activeIndex >= total {
		x = 0
	}
	// Flag both ends of the healthy range: past the cap, cycling still reaches
	// every profile but only the first MaxEquipped have a 1-9 chord; at zero
	// there is nothing to switch to and the grid is the rescue fallback.
	marker := ""
	if total > profiles.MaxEquipped || total == 0 {
		marker = "⚠ "
	}
	counter := fmt.Sprintf("< %d / %d %s>", x, total, marker)
	return m.styles.Title.Render(counter)
}

// renderProfileBar shows HOST | PROFILE | lock/glassroot badges. When it
// doesn't fit availWidth, PROFILE drops first — HOST and the badges stay,
// since those are what you'd check when something's gone sideways.
func (m Model) renderProfileBar(availWidth int) string {
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
	sep := m.styles.Help.Render(" | ")

	hostSeg := "HOST: " + username + "@" + hostname + " " + osArch
	profileSeg := lipgloss.NewStyle().Render("PROFILE: ") + m.profile.activeName()

	var tail string
	if m.profile.pivotName != "" {
		tail += m.styles.LockBadge.Render(fmt.Sprintf("🔒 %s", m.profile.pivotName))
	}
	if m.GlassrootMode {
		tail += lipgloss.NewStyle().Foreground(lipgloss.Color("#A8E6CF")).Render("🧊 G-ROOT")
	}

	full := hostSeg + sep + profileSeg
	if tail != "" {
		full += sep + tail
	}
	if lipgloss.Width(full) <= availWidth {
		return full
	}

	slim := hostSeg
	if tail != "" {
		slim += sep + tail
	}
	return truncateText(slim, availWidth)
}

// renderStatusMessage is its own row rather than a profileBar segment — a
// batch/tmux error is often the most important thing on screen, and
// shouldn't compete for space with HOST/PROFILE/lock.
func (m Model) renderStatusMessage(availWidth int) string {
	if m.profile.statusMessage == "" {
		return ""
	}
	style := m.styles.StatusNegative
	if m.profile.statusPositive {
		style = m.styles.StatusPositive
	}
	return truncateText(style.Render(m.profile.statusMessage), availWidth)
}
