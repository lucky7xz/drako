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

// popupFill is a style that carries only the popup's background. Every
// segment inside a popup renders with it, or the reset that ends each nested
// Render punches a hole in the background.
func (m Model) popupFill() lipgloss.Style {
	return lipgloss.NewStyle().Background(m.styles.DropdownPopup.GetBackground())
}

// framePopup pads lines into a solid block as wide as the widest one, on the
// popup's background, and draws the popup frame around it.
func (m Model) framePopup(lines []string) string {
	return m.styles.DropdownPopup.Render(lipgloss.JoinVertical(lipgloss.Left, padLinesToWidth(lines, m.popupFill())...))
}

// renderDropdownPopup is the open folder's items, floated over the grid.
func (m Model) renderDropdownPopup() string {
	// Ensure every segment renders with the popup background to avoid black gaps
	bg := m.styles.DropdownPopup.GetBackground()
	cursorSel := m.styles.SelectedCursor.Background(bg)
	textNorm := m.styles.Item.Background(bg)
	textSel := m.styles.SelectedItem.Background(bg)
	gap := m.popupFill()

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
	return m.framePopup(raw)
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
	bgFill := m.popupFill()
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
	return m.framePopup(raw)
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

// explainLayout is the explain popup laid out for the current terminal: the
// styled rows pinned above and below the script, the script wrapped, and how
// much of it fits. Built once per use, the view draws it and PgUp/PgDn page
// through it, so the numbers they work from cannot disagree.
type explainLayout struct {
	head, tail []string // styled rows above and below the script
	script     []string // the script, wrapped; nil without a value
	viewportH  int      // script rows on screen at once
	maxOffset  int
	scrollable bool
}

// explainLayout lays out the current activeDetail, which must be set.
func (m Model) explainLayout() explainLayout {
	d := m.activeDetail
	ww := m.infoWrapWidth()
	bg := m.styles.DropdownPopup.GetBackground()
	label := m.styles.Help.Background(bg)
	value := m.styles.Item.Background(bg)

	var l explainLayout
	if strings.TrimSpace(d.Title) != "" {
		l.head = append(l.head, m.styles.Title.Background(bg).Render(truncateText(d.Title, ww)))
	}
	if strings.TrimSpace(d.Value) != "" {
		name := "Value:"
		if d.KeyLabel != "" {
			name = d.KeyLabel + ":"
		}
		l.head = append(l.head, "", label.Render(name))
		l.script = WrapScript(d.Value, ww)
	}
	if strings.TrimSpace(d.Description) != "" {
		l.tail = append(l.tail, "", label.Render("Description:"))
		for _, ln := range WrapText(d.Description, ww) {
			l.tail = append(l.tail, value.Render(ln))
		}
	}
	if len(d.Meta) > 0 {
		l.tail = append(l.tail, "")
		for _, meta := range d.Meta {
			v := truncateLeft(meta.Value, max(1, ww-lipgloss.Width(meta.Label)-2))
			l.tail = append(l.tail, label.Render(meta.Label+": ")+value.Render(v))
		}
	}

	// The last pinned rows are a blank and the help line. Which help shows
	// depends on whether the script scrolls, which depends on the room the
	// help leaves — so the room is counted for the longer of the two.
	plain, scrolling := WrapText(infoHelp, ww), WrapText(infoScrollHelp, ww)
	pinned := len(l.head) + len(l.tail) + 1 + max(len(plain), len(scrolling))
	room := m.termHeight - m.styles.DropdownPopup.GetVerticalFrameSize() - pinned
	l.viewportH = max(3, min(infoViewportRows, room))
	l.maxOffset = max(0, len(l.script)-l.viewportH)
	// The error queue renders its (short) Value in full and steers every key,
	// so only the plain explain overlay gets a scrollable, windowed script.
	l.scrollable = !m.profile.errorQueueActive && len(l.script) > l.viewportH

	help := plain
	if l.scrollable {
		help = scrolling
	}
	l.tail = append(l.tail, "")
	for _, ln := range help {
		l.tail = append(l.tail, m.styles.Help.Render(ln))
	}
	return l
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
	// Safety check if activeDetail is nil (should not happen in infoMode ideally)
	if m.activeDetail == nil {
		return m.styles.DropdownPopup.Render("Error: No detail state")
	}
	l := m.explainLayout()

	// Every segment carries the popup background, to avoid black gaps.
	bg := m.styles.DropdownPopup.GetBackground()
	bgFill := m.popupFill()
	valueStyle := m.styles.Item.Background(bg)

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

	// The script: in full, or windowed with a scrollbar when it overflows.
	var valueRows []string
	if !l.scrollable {
		for _, ln := range l.script {
			valueRows = append(valueRows, renderVal(ln))
		}
	} else {
		offset := min(max(m.activeDetail.ScrollOffset, 0), l.maxOffset)

		// Widen the script block to the widest pinned line so the scrollbar
		// lands at the popup's right edge rather than mid-content.
		blockW := 0
		for _, ln := range l.script {
			blockW = max(blockW, lipgloss.Width(ln))
		}
		for _, ln := range append(append([]string{}, l.head...), l.tail...) {
			blockW = max(blockW, lipgloss.Width(ln)-2)
		}

		// Bar cells must be padding-free (valueStyle carries Item's Padding(0,1),
		// which would widen thumb rows and stagger the column). bgFill sets only
		// the background.
		bars := scrollbarColumn(len(l.script), offset, l.viewportH, bgFill.Bold(true), bgFill.Foreground(hlComment))
		for i, ln := range l.script[offset : offset+l.viewportH] {
			pad := max(0, blockW-lipgloss.Width(ln))
			valueRows = append(valueRows,
				renderVal(ln)+bgFill.Render(strings.Repeat(" ", pad)+" ")+bars[i])
		}
	}

	raw := append(append(append([]string{}, l.head...), valueRows...), l.tail...)
	return m.framePopup(raw)
}
