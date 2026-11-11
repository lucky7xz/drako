package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/settings"
)

// The key sheet: one reference for every screen's keys, opened with the help
// key (? by default). It replaces the per-screen help lines, so the footer
// only carries keyHint.

// here is the screen the user is on, as the key sheet files it and as q
// treats it: q quits only from the grid and backs out one step elsewhere
// (IsCancel). Explain closes on any key, so it has no q of its own.
func (m Model) here() (section, q string) {
	switch {
	case m.mode == infoMode:
		return "EXPLAIN", ""
	case m.batch.choosing():
		return "BATCH", "back"
	case m.batch.dropdown, m.mode == batchMode:
		return "BATCH", "cancel"
	case m.mode == dropdownMode:
		return "FOLDER", "close"
	case m.mode == pathMode, m.mode == childMode:
		return "PATH", "back"
	case m.mode == inventoryMode:
		return "INVENTORY", "back"
	}
	return "GRID", "quit"
}

// keyHint is the footer's reminder of the two keys everyone needs: the key
// sheet, and what q does here.
func (m Model) keyHint() string {
	if _, q := m.here(); q != "" {
		return m.Config.Keys.Help + " help · q " + q
	}
	return "any key closes"
}

// A binding is one entry on the key sheet: what you press, and what it does.
type binding struct {
	keys, does string
	gated      bool // glassroot blocks it
}

// A sheetSection is one screen's bindings.
type sheetSection struct {
	title    string
	bindings []binding
	gated    bool // glassroot blocks the whole screen
	here     bool // the screen the sheet was opened from
}

func bind(keys, does string) binding  { return binding{keys: keys, does: does} }
func gated(keys, does string) binding { return binding{keys: keys, does: does, gated: true} }

// sheetSections is every screen's keys, read off the config so a remapped key
// shows as remapped. Each section mirrors what that screen's handler does.
// Glassroot leaves out what it blocks (glassroot.go), so the
// sheet never offers a key that does nothing.
func sheetSections(cfg settings.Config, glassroot bool) []sheetSection {
	k := cfg.Keys
	upDown := moveLabel(k, "↑↓", "ws", "kj")
	leftRight := moveLabel(k, "←→", "ad", "hl")

	secs := []sheetSection{
		{title: "GRID", bindings: []binding{
			bind("enter", "run cell, open folder"),
			bind(k.Explain, "explain"),
			bind("1-9 1-9", "jump: column, row"),
			bind(moveLabel(k, "←↑↓→", "wasd", "hjkl"), "move"),
			gated(k.PathGridMode, "path mode"),
			gated(k.Inventory, "inventory"),
			gated(k.Lock, "pin deck"),
			bind(k.ProfilePrev, "previous deck"),
			bind(k.ProfileNext, "next deck"),
			bind(k.Leader+" 1-9", "switch deck"),
			bind(cfg.NumbModifier+"+1-9", "switch deck"),
			gated(k.Leader+" b", "batch mode"),
			bind("q", "quit"),
		}},
		{title: "FOLDER", bindings: []binding{
			bind(upDown, "select"),
			bind("1-9", "jump to item"),
			bind("enter", "run"),
			bind(k.Explain, "explain"),
			gated(k.Leader+" b", "batch these"),
			bind("esc q", "close"),
		}},
		{title: "BATCH", gated: true, bindings: []binding{
			bind("space", "mark"),
			bind("enter", "lay out, launch"),
			bind(leftRight, "pick layout field"),
			bind(upDown, "adjust it"),
			bind("esc q", "back"),
		}},
		{title: "PATH", gated: true, bindings: []binding{
			bind(leftRight, "pick folder"),
			bind(moveLabel(k, "↓", "s", "j"), "children"),
			bind(upDown, "pick child"),
			bind("enter", "cd"),
			bind(k.PathSearch, "search"),
			bind(k.ToggleHidden, "hidden files"),
			bind(k.PathGridMode+" esc q", "back to grid"),
		}},
		{title: "INVENTORY", gated: true, bindings: []binding{
			bind(upDown, "switch list"),
			bind(leftRight, "move along"),
			bind("enter", "lift, place"),
			bind(k.PathGridMode, "specs"),
			bind(k.EditFile, "edit"),
			bind(k.Delete, "trash"),
			bind("esc q", "back"),
		}},
		{title: "EXPLAIN", bindings: []binding{
			bind("pgup pgdn", "scroll"),
			gated("y", "copy"),
			bind("any key", "close"),
		}},
		{title: "ANYWHERE", bindings: []binding{
			bind(k.Help, "this sheet"),
			bind(cfg.NumbModifier+"+"+k.SessionLock, "lock session"),
			bind("ctrl+c", "quit"),
		}},
	}
	if !glassroot {
		return secs
	}

	var open []sheetSection
	for _, s := range secs {
		if s.gated {
			continue
		}
		var bs []binding
		for _, b := range s.bindings {
			if !b.gated {
				bs = append(bs, b)
			}
		}
		s.bindings = bs
		open = append(open, s)
	}
	return open
}

// moveLabel names a set of direction keys: the arrows, then the wasd and vim
// equivalents unless the config disables them, as InitControls does.
func moveLabel(k settings.InputConfig, arrows, wasd, vim string) string {
	label := arrows
	if !k.DisableWasd {
		label += " " + wasd
	}
	if !k.DisableVim {
		label += " " + vim
	}
	return label
}

// sheetHere is the sheet as this screen shows it: the current screen's
// section first, marked here.
func (m Model) sheetHere() []sheetSection {
	here, _ := m.here()

	secs := sheetSections(m.Config, m.GlassrootMode)
	for i, s := range secs {
		if s.title == here {
			s.here = true
			out := append([]sheetSection{s}, secs[:i]...)
			return append(out, secs[i+1:]...)
		}
	}
	return secs
}

// sheetState is the open key sheet; Model.sheet is nil while it is closed.
type sheetState struct {
	offset int // first visible row when the sheet is taller than the screen
}

// opensKeySheet reports whether msg opens the sheet: the help key, on a screen
// the sheet lists, when nothing else owns the key — not while typing, not
// mid-leader-sequence. Explain and the lock screen keep their own rules.
func (m Model) opensKeySheet(msg tea.KeyMsg) bool {
	if !IsHelp(m.Config.Keys, msg) || m.capturingText() || m.leader.pending {
		return false
	}
	switch m.mode {
	case gridMode, pathMode, childMode, dropdownMode, batchMode:
		return true
	case inventoryMode:
		return m.inventory.err == nil
	}
	return false
}

// updateKeySheet scrolls the open sheet a line (up/down) or a page (PgUp/PgDn);
// any other key closes it and does nothing else.
func (m Model) updateKeySheet(msg tea.KeyMsg) Model {
	_, viewportH, maxOffset := m.sheetMetrics()
	offset := m.sheet.offset
	switch {
	case msg.String() == "pgup":
		offset -= viewportH - 1
	case msg.String() == "pgdown":
		offset += viewportH - 1
	case IsUp(m.Config.Keys, msg):
		offset--
	case IsDown(m.Config.Keys, msg):
		offset++
	default:
		m.sheet = nil
		return m
	}
	m.sheet = &sheetState{offset: min(max(offset, 0), maxOffset)}
	return m
}

// sheetWidth is the width the sheet's rows are packed to: up to 60 columns,
// narrower when the terminal is, so the whole popup stays on screen. Popups
// float over the full terminal, so the frame is only the popup's border,
// padding and margin, and the scrollbar column.
func (m Model) sheetWidth() int {
	frame := m.styles.DropdownPopup.GetHorizontalFrameSize() + 2
	return max(8, min(60, m.termWidth-frame))
}

// sheetMetrics returns the sheet's rows, how many fit on screen, and the
// largest scroll offset. The view and the scroll keys both call it, so their
// numbers agree — the explainLayout pattern.
func (m Model) sheetMetrics() (lines []string, viewportH, maxOffset int) {
	lines = m.sheetLines(m.sheetWidth())
	frame := m.styles.DropdownPopup.GetVerticalFrameSize() + 4 // + title, two spacers, hint
	viewportH = max(3, min(len(lines), m.termHeight-frame))
	return lines, viewportH, max(0, len(lines)-viewportH)
}

// sheetLines lays the sections out width columns wide: a blank line, then
// each section's title — the current one in the accent colour behind a ▸, the
// rest dim — then its bindings as "[keys] does" chips packed into rows, kept
// apart by " · ".
func (m Model) sheetLines(width int) []string {
	bg := m.styles.DropdownPopup.GetBackground()
	fill := lipgloss.NewStyle().Background(bg)
	dim := fill.Foreground(m.styles.Help.GetForeground())
	key := fill.Foreground(m.styles.SelectedCursor.GetForeground())

	var lines []string
	for i, s := range m.sheetHere() {
		if i > 0 {
			lines = append(lines, "")
		}
		if s.here {
			lines = append(lines, key.Bold(true).Render("▸ "+s.title))
		} else {
			lines = append(lines, dim.Bold(true).Render("  "+s.title))
		}

		row, rowW := "", 0
		for _, b := range s.bindings {
			chip := truncateText(dim.Render("[")+key.Render(b.keys)+dim.Render("]")+fill.Render(" "+b.does), width-2)
			chipW := lipgloss.Width(chip)
			if rowW > 0 && rowW+3+chipW > width {
				lines = append(lines, row)
				row, rowW = "", 0
			}
			if rowW == 0 {
				row, rowW = fill.Render("  "), 2 // under the title, past the ▸
			} else {
				row, rowW = row+dim.Render(" · "), rowW+3
			}
			row, rowW = row+chip, rowW+chipW
		}
		if rowW > 0 {
			lines = append(lines, row)
		}
	}
	return lines
}

// renderKeySheet draws the sheet as a popup, the dropdown's look, scrolled to
// the current offset with a scrollbar when it doesn't fit. Rows are padded to
// the packing width, not their own, so the border stays still while scrolling.
func (m Model) renderKeySheet() string {
	lines, viewportH, maxOffset := m.sheetMetrics()
	width := m.sheetWidth()
	bg := m.styles.DropdownPopup.GetBackground()
	fill := lipgloss.NewStyle().Background(bg)

	scrolling := maxOffset > 0
	offset := min(m.sheet.offset, maxOffset)
	body := padLinesTo(lines[offset:offset+viewportH], width, fill)
	hint := "any key closes"
	if scrolling {
		bars := scrollbarColumn(len(lines), offset, viewportH, fill.Bold(true), fill.Foreground(hlComment))
		for i := range body {
			body[i] += fill.Render(" ") + bars[i]
		}
		hint = "PgUp/PgDn scroll · any other key closes"
	}

	// The title and hint start in the ▸ column, one level out from the sections.
	raw := append([]string{m.styles.Title.Padding(0).Background(bg).Render("Keys"), ""}, body...)
	raw = append(raw, "", m.styles.Help.Background(bg).Render(truncateText(hint, width)))
	return m.styles.DropdownPopup.Render(lipgloss.JoinVertical(lipgloss.Left, padLinesToWidth(raw, fill)...))
}
