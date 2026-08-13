package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/multiplex"
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
	helpText := "Dropdown Mode | ↑/↓/ws: Select, Enter: Execute, Esc/q: Cancel"
	layout := CalculateLayout(m.termWidth, m.termHeight, m.Config, m.styles.HeaderArt, helpText)
	header := ""
	if layout.ShowHeader {
		header = m.styles.renderHeaderArt(m.spinner.View())
	}

	var footer string
	if layout.ShowFooter {
		footer = m.renderCombinedFooter(helpText)
	}

	hAlign := lipgloss.Center
	if layout.ShiftLeft {
		hAlign = lipgloss.Left
	}

	// hAlign governs these inner joins too, not just the outer Place()
	// below — otherwise the grid would stay centered relative to a wider
	// wrapped footer even while the block as a whole shifts left.
	grid := m.renderGrid(gridRowBudget(m.termHeight, header, footer))
	mainContent := lipgloss.JoinVertical(hAlign, header, grid)

	finalContent := lipgloss.JoinVertical(
		hAlign,
		mainContent,
		footer,
	)

	// Render dropdown popup
	dropdownPopup := m.renderDropdownPopup()

	// Place the dropdown in the center of the screen — it's a small floating
	// overlay, not part of the header/grid/footer width cascade, so it stays
	// centered regardless of layout.ShiftLeft.
	popupOverlay := lipgloss.Place(m.termWidth, m.termHeight,
		lipgloss.Center, lipgloss.Center,
		dropdownPopup,
	)

	return appStyle.Render(
		lipgloss.Place(m.termWidth, m.termHeight,
			hAlign, lipgloss.Center,
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

	// Build lines, then right-pad them into a solid block. Items are always
	// numbered — digits jump the selector; in a dropdown batch each runnable
	// item also carries its mark glyph.
	var raw []string
	for i, item := range m.dropdown.items {
		label := fmt.Sprintf("%d %s%s", i+1, m.itemMarkPrefix(item), item.Name)
		if i == m.dropdown.selectedIdx {
			raw = append(raw, cursorSel.Render("► ")+textSel.Render(label))
		} else {
			raw = append(raw, gap.Render("  ")+textNorm.Render(label))
		}
	}
	if m.batch.dropdown {
		// Same look as the grid's batch counter (styles.Title), so the mode
		// reads at a glance; the key hints stay in the quiet item style.
		counter := m.styles.Title.Background(bg).Render(fmt.Sprintf("[ BATCH %d/%d ]", len(m.batch.marked), multiplex.MaxCommands))
		hints := textNorm.Render(" Space mark · Enter launch · Esc/q cancel")
		raw = append(raw, gap.Render("  ")+counter+hints)
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

// infoViewportRows is the target height of the scrollable script block in the
// explain overlay. The actual height is clamped down to fit small terminals.
const infoViewportRows = 18

// infoWrapWidth is the wrap width for explain-popup content.
func (m Model) infoWrapWidth() int {
	w := m.termWidth - 10
	if w > 80 {
		w = 80
	}
	if w < 20 {
		w = 20
	}
	return w
}

// infoScrollMetrics returns the wrapped script lines, the clamped viewport
// height, and the max scroll offset for the current explain overlay. The view
// and the PgUp/PgDn handler both call it so their numbers stay in sync.
func (m Model) infoScrollMetrics() (valueLines []string, viewportH, maxOffset int) {
	if m.activeDetail == nil {
		return nil, 0, 0
	}
	ww := m.infoWrapWidth()
	valueLines = WrapText(m.activeDetail.Value, ww)

	// Count the pinned (non-script) rows exactly as viewInfoMode lays them out,
	// so the viewport clamp matches what actually reaches the screen.
	d := m.activeDetail
	pinned := 0
	if strings.TrimSpace(d.Title) != "" {
		pinned++
	}
	if strings.TrimSpace(d.Value) != "" {
		pinned += 2 // blank spacer + the "Command:" label
	}
	if strings.TrimSpace(d.Description) != "" {
		pinned += 2 + len(WrapText(d.Description, ww))
	}
	if len(d.Meta) > 0 {
		pinned += 1 + len(d.Meta)
	}
	pinned += 2 // blank spacer + help line

	headerH := 0
	if CalculateLayout(m.termWidth, m.termHeight, m.Config, m.styles.HeaderArt, "").ShowHeader {
		headerH = lipgloss.Height(m.styles.renderHeaderArt(m.spinner.View()))
	}

	const chrome = 8 // popup border + padding + margin, plus a little slack
	budget := m.termHeight - headerH - pinned - chrome

	viewportH = infoViewportRows
	if viewportH > budget {
		viewportH = budget
	}
	if viewportH < 3 {
		viewportH = 3
	}

	maxOffset = len(valueLines) - viewportH
	if maxOffset < 0 {
		maxOffset = 0
	}
	return valueLines, viewportH, maxOffset
}

// scrollbarColumn returns viewportH cells — a █ thumb over a ░ track — for a
// list of `total` lines currently scrolled to `offset`. The thumb length is
// proportional to the visible fraction and its position maps offset linearly
// so the bottom offset lands the thumb flush against the track's end.
func scrollbarColumn(total, offset, viewportH int, thumb, track lipgloss.Style) []string {
	thumbLen := viewportH * viewportH / total
	if thumbLen < 1 {
		thumbLen = 1
	}
	if thumbLen > viewportH {
		thumbLen = viewportH
	}
	maxOffset := total - viewportH
	if maxOffset < 1 {
		maxOffset = 1
	}
	thumbStart := offset * (viewportH - thumbLen) / maxOffset
	if thumbStart < 0 {
		thumbStart = 0
	}
	if thumbStart+thumbLen > viewportH {
		thumbStart = viewportH - thumbLen
	}
	cells := make([]string, viewportH)
	for i := range cells {
		if i >= thumbStart && i < thumbStart+thumbLen {
			cells[i] = thumb.Render("█")
		} else {
			cells[i] = track.Render("░")
		}
	}
	return cells
}

func (m Model) viewInfoMode() string {
	// No footer/help line is rendered in this view, so helpText is "" —
	// ShiftLeft is structurally always false here (see CalculateLayout),
	// and this view's own Place call stays unconditionally centered.
	layout := CalculateLayout(m.termWidth, m.termHeight, m.Config, m.styles.HeaderArt, "")
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
	wrapWidth := m.infoWrapWidth()

	// Safety check if activeDetail is nil (should not happen in infoMode ideally)
	if m.activeDetail == nil {
		return appStyle.Render(lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center, "Error: No detail state"))
	}

	valueLines, viewportH, maxOffset := m.infoScrollMetrics()
	// The error queue renders its (short) Value in full and steers every key,
	// so only the plain explain overlay gets a scrollable, windowed script.
	scrollable := !m.profile.errorQueueActive && len(valueLines) > viewportH

	// head: title + value label, pinned above the script.
	var head []string
	if strings.TrimSpace(m.activeDetail.Title) != "" {
		head = append(head, titleStyleLocal.Render(m.activeDetail.Title))
	}
	hasValue := strings.TrimSpace(m.activeDetail.Value) != ""
	if hasValue {
		head = append(head, "")
		label := "Value:"
		if m.activeDetail.KeyLabel != "" {
			label = m.activeDetail.KeyLabel + ":"
		}
		head = append(head, labelStyle.Render(label))
	}

	// tail: description, meta, help line, pinned below the script.
	var tail []string
	if strings.TrimSpace(m.activeDetail.Description) != "" {
		tail = append(tail, "")
		tail = append(tail, labelStyle.Render("Description:"))
		for _, ln := range WrapText(m.activeDetail.Description, wrapWidth) {
			tail = append(tail, valueStyle.Render(ln))
		}
	}
	if len(m.activeDetail.Meta) > 0 {
		tail = append(tail, "")
		for _, meta := range m.activeDetail.Meta {
			tail = append(tail, labelStyle.Render(meta.Label+": ")+valueStyle.Render(meta.Value))
		}
	}
	tail = append(tail, "")
	helpText := "Press y to copy command/details to clipboard • any key to close"
	if scrollable {
		helpText = "PgUp/PgDn scroll • y copy • any other key to close"
	}
	tail = append(tail, m.styles.Help.Render(helpText))

	// Only real command values get shell highlighting; error/profile overlays
	// (KeyLabel "Error"/"Profile") keep plain rendering so paths and TOML errors
	// aren't mis-colored as shell. Both forms keep valueStyle's Padding(0,1) width.
	highlight := m.activeDetail.KeyLabel == "Command"
	renderVal := func(ln string) string {
		if highlight {
			return highlightShell(ln, bg)
		}
		return valueStyle.Render(ln)
	}

	// value/script block: windowed with a scrollbar when it overflows.
	var valueRows []string
	if hasValue && scrollable {
		offset := m.activeDetail.ScrollOffset
		if offset < 0 {
			offset = 0
		}
		if offset > maxOffset {
			offset = maxOffset
		}
		window := valueLines[offset : offset+viewportH]

		// Widen the script block to the widest pinned line so the scrollbar
		// lands at the popup's right edge rather than mid-content.
		blockW := 0
		for _, ln := range valueLines {
			if w := lipgloss.Width(ln); w > blockW {
				blockW = w
			}
		}
		for _, ln := range append(append([]string{}, head...), tail...) {
			if w := lipgloss.Width(ln) - 2; w > blockW {
				blockW = w
			}
		}

		// Bar cells must be padding-free (valueStyle carries Item's Padding(0,1),
		// which would widen thumb rows and stagger the column). bgFill sets only
		// the background.
		thumbStyle := bgFill.Bold(true)
		trackStyle := bgFill.Foreground(hlComment)
		bars := scrollbarColumn(len(valueLines), offset, viewportH, thumbStyle, trackStyle)
		for i, ln := range window {
			pad := blockW - lipgloss.Width(ln)
			if pad < 0 {
				pad = 0
			}
			valueRows = append(valueRows,
				renderVal(ln)+bgFill.Render(strings.Repeat(" ", pad)+" ")+bars[i])
		}
	} else if hasValue {
		for _, ln := range valueLines {
			valueRows = append(valueRows, renderVal(ln))
		}
	}

	raw := append(append(append([]string{}, head...), valueRows...), tail...)

	lines := padLinesToWidth(raw, bgFill)

	popup := m.styles.DropdownPopup.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	content := lipgloss.JoinVertical(lipgloss.Center, header, popup)
	return appStyle.Render(lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center, content))
}
