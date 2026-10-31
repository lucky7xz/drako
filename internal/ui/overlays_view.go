package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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

// overlay draws popup centered over base, a finished w×h frame, leaving the
// rest of base visible around it. The renderer shows only the last h lines of
// a taller frame (bubbletea drops lines from the top), so those are the ones
// kept and centered on — and a popup taller than h keeps its last h lines the
// same way, so its way out (the hint and the border) stays on screen. Each
// covered line is cut around the popup with the ANSI-aware truncations, which
// keep the style codes on either side; a reset at both seams stops a color
// from running into or out of the popup.
func overlay(base, popup string, w, h int) string {
	lines := strings.Split(base, "\n")
	if len(lines) > h {
		lines = lines[len(lines)-h:]
	}
	popLines := strings.Split(popup, "\n")
	if len(popLines) > h {
		popLines = popLines[len(popLines)-h:]
	}
	popW := lipgloss.Width(popup)
	top := max(0, (h-len(popLines))/2)
	left := max(0, (w-popW)/2)

	for i, pl := range popLines {
		row := top + i
		if row >= len(lines) {
			break
		}
		under := lines[row]
		before := ansi.Truncate(under, left, "")
		pad := strings.Repeat(" ", left-ansi.StringWidth(before))
		after := ansi.TruncateLeft(under, left+popW, "")
		lines[row] = before + pad + ansi.ResetStyle + pl + ansi.ResetStyle + after
	}
	return strings.Join(lines, "\n")
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
	return padLinesTo(raw, maxW, bgFill)
}

// padLinesTo right-pads every line to an explicit width. A popup that sizes
// itself to its content resizes whenever the content changes; one that passes
// a fixed width here keeps its border still while the text inside it moves.
// Lines already wider than width are left alone — truncate the text before
// styling it, since cutting a rendered line would slice its escape sequences.
func padLinesTo(raw []string, width int, bgFill lipgloss.Style) []string {
	lines := make([]string, len(raw))
	for i, line := range raw {
		pad := max(0, width-lipgloss.Width(line))
		lines[i] = line + bgFill.Render(strings.Repeat(" ", pad))
	}
	return lines
}

// renderDropdownPopup is the open folder's items, floated over the grid.
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
		label := fmt.Sprintf("%d %s%s", i+1, m.itemMarkPrefix(i, item), item.Name)
		if i == m.dropdown.selectedIdx {
			raw = append(raw, cursorSel.Render("► ")+textSel.Render(label))
		} else {
			raw = append(raw, gap.Render("  ")+textNorm.Render(label))
		}
	}
	if m.batch.dropdown {
		// Same look as the grid's batch counter (styles.Title), so the mode
		// reads at a glance; the key hints stay in the quiet item style.
		counter := m.styles.Title.Background(bg).Render(m.batchCount())
		hints := textNorm.Render(" Space mark · Enter launch · Esc/q cancel")
		raw = append(raw, gap.Render("  ")+counter+hints)
	}
	lines := padLinesToWidth(raw, bgFill)

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return m.styles.DropdownPopup.Render(content)
}

// renderLayoutPopup is the batch layout dialog, floated over the grid. It takes
// the dropdown's place when the batch came from a folder, so one framed box is
// on screen at a time and the dialog looks the same from either entry point.
// It draws the fields as a row of grid cells — the same
// ┌──[A]──┐ header over the same retro-bordered box the grid uses — so the
// dialog reads as drako rather than as a form. The focused field's header rule
// turns double, which shows focus without relying on colour.
func (m Model) renderLayoutPopup() string {
	// Every segment renders with the popup background, or the resets that end
	// each nested Render punch holes in it — the same care renderDropdownPopup
	// takes, and for the same reason.
	bg := m.styles.DropdownPopup.GetBackground()
	bgFill := lipgloss.NewStyle().Background(bg)
	title := m.styles.Title.Background(bg)

	labels, bodies := m.layoutFields()
	content := m.layoutFieldWidth()

	var heads, cells []string
	for i := range labels {
		fill := "─"
		style := m.styles.Cell
		if i == m.batch.focus {
			fill, style = "═", m.styles.SelectedCell
		}
		// Centre the label, then turn the padding into rule — the same trick
		// renderGrid uses for its column headers, so the two always match.
		// The cell's border and padding add one column each side.
		head := bgFill.Width(content + 2).Align(lipgloss.Center).
			Render(title.Render("[" + labels[i] + "]"))
		heads = append(heads, bgFill.Render("┌")+strings.ReplaceAll(head, " ", fill)+bgFill.Render("┐"))

		body := bgFill.Width(content).Align(lipgloss.Center).Render(bodies[i])
		cells = append(cells, style.Background(bg).BorderBackground(bg).Render(body))
	}

	headRow := lipgloss.JoinHorizontal(lipgloss.Left, heads...)
	cellRows := strings.Split(lipgloss.JoinHorizontal(lipgloss.Top, cells...), "\n")
	counter := bgFill.Width(lipgloss.Width(headRow)).Align(lipgloss.Center).
		Render(title.Render(m.batchCount()))

	raw := append([]string{headRow}, cellRows...)
	raw = append(raw, "", counter)
	lines := padLinesToWidth(raw, bgFill)

	return m.styles.DropdownPopup.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

// The lock screen never runs CalculateLayout — no header, grid or help line to
// hide — so it budgets its own width. The heights count usable rows: the
// terminal less appStyle's margin.
const (
	lockBoxChrome   = 10 // Padding(_, 4) both sides plus the border
	lockRoomyHeight = 23 // below this the spacers go, then the padding
	lockTightHeight = 15 // below this the idle line goes too
)

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

	avail := max(1, m.termWidth-appStyle.GetHorizontalMargins()-lockBoxChrome)

	progress := min(max(m.lock.progress, 0), goal)
	barWidth := min(24, max(2, avail-2))
	filled := progress * barWidth / goal

	bar := "[" + strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled) + "]"

	// Shorter phrasing beats wrapping the full line over four rows.
	instructionText := "Pump ← → (A/D or H/L) to fill the slider and unlock"
	if lipgloss.Width(instructionText) > avail {
		instructionText = "Pump ← → to unlock"
	}

	// The box is as wide as its widest child, so every row is bounded.
	fit := func(s string) string { return truncateText(s, avail) }

	lockIcon := "🔒"
	title := fit(m.styles.Title.Render("Session Locked"))
	timeInfo := fit(m.styles.Help.Render(fmt.Sprintf("Idle for %d minute(s)", elapsedMins)))
	instructions := m.styles.Help.Render(strings.Join(WrapText(instructionText, avail), "\n"))
	progressLabel := fit(m.styles.Help.Render(fmt.Sprintf("%d / %d pumps", m.lock.progress, goal)))
	quitHint := fit(m.styles.Help.Render("Press Ctrl+C to quit"))

	usable := m.termHeight - appStyle.GetVerticalMargins()
	roomy := usable >= lockRoomyHeight
	tight := usable < lockTightHeight

	rows := []string{lockIcon, title}
	if !tight {
		rows = append(rows, timeInfo)
	}
	rows = append(rows, instructions, progressLabel, bar, quitHint)
	if roomy {
		rows = spaced(rows)
	}

	content := lipgloss.JoinVertical(lipgloss.Center, rows...)

	vPad := 2
	if !roomy {
		vPad = 1
	}
	if tight {
		vPad = 0
	}

	// Add a border box around the lock screen
	box := lipgloss.NewStyle().
		Padding(vPad, 4).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(warnAmber).
		Align(lipgloss.Center).
		Render(content)

	inner := m.termWidth - appStyle.GetHorizontalMargins()
	footer := lipgloss.NewStyle().
		Width(inner).
		Align(lipgloss.Center).
		Render(truncateText(m.renderFooter(), inner))

	return m.frame(lipgloss.JoinVertical(lipgloss.Center, box, footer))
}

// spaced puts a blank line between each row, and one above the first.
func spaced(rows []string) []string {
	out := make([]string, 0, len(rows)*2)
	for _, r := range rows {
		out = append(out, "", r)
	}
	return out
}

// infoViewportRows is the target height of the scrollable script block in the
// explain overlay. The actual height is clamped down to fit small terminals.
const infoViewportRows = 18

// The explain popup's last line, plain and while the script scrolls.
const (
	infoHelp       = "Press y to copy command/details to clipboard • any key to close"
	infoScrollHelp = "PgUp/PgDn scroll • y copy • any other key to close"
)

// infoWrapWidth is the wrap width for explain-popup content: what is left of
// the terminal once the popup's frame, the value's padding and the scrollbar
// have their columns, so the popup always fits on screen. Popups float over
// the full terminal, so appStyle's margin is not theirs to keep clear.
func (m Model) infoWrapWidth() int {
	return max(1, min(80, m.termWidth-m.styles.DropdownPopup.GetHorizontalFrameSize()-4))
}

// infoScrollMetrics returns the wrapped script lines, the clamped viewport
// height, and the max scroll offset for the current explain overlay. The view
// and the PgUp/PgDn handler both call it so their numbers stay in sync.
func (m Model) infoScrollMetrics() (valueLines []string, viewportH, maxOffset int) {
	if m.activeDetail == nil {
		return nil, 0, 0
	}
	ww := m.infoWrapWidth()
	valueLines = WrapScript(m.activeDetail.Value, ww)

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
	// blank spacer + the help line, which wraps on a narrow terminal; count
	// the longer of its two forms, since which one shows depends on this.
	pinned += 1 + max(len(WrapText(infoHelp, ww)), len(WrapText(infoScrollHelp, ww)))

	const chrome = 8 // popup border + padding + margin, plus a little slack
	budget := m.termHeight - pinned - chrome

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

// renderInfoPopup is the explain popup (and the profile notes that share it),
// floated over the screen it was opened from.
func (m Model) renderInfoPopup() string {
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
		return m.styles.DropdownPopup.Render("Error: No detail state")
	}

	valueLines, viewportH, maxOffset := m.infoScrollMetrics()
	// The error queue renders its (short) Value in full and steers every key,
	// so only the plain explain overlay gets a scrollable, windowed script.
	scrollable := !m.profile.errorQueueActive && len(valueLines) > viewportH

	// head: title + value label, pinned above the script.
	var head []string
	if strings.TrimSpace(m.activeDetail.Title) != "" {
		head = append(head, titleStyleLocal.Render(truncateText(m.activeDetail.Title, wrapWidth)))
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
			value := truncateLeft(meta.Value, max(1, wrapWidth-lipgloss.Width(meta.Label)-2))
			tail = append(tail, labelStyle.Render(meta.Label+": ")+valueStyle.Render(value))
		}
	}
	tail = append(tail, "")
	helpText := infoHelp
	if scrollable {
		helpText = infoScrollHelp
	}
	for _, ln := range WrapText(helpText, wrapWidth) {
		tail = append(tail, m.styles.Help.Render(ln))
	}

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

	return m.styles.DropdownPopup.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}
