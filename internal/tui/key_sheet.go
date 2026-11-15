package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucky7xz/drako/internal/settings"
)

// The key sheet: one reference for every screen's keys, opened with the help
// key (? by default). It replaces the per-screen help lines, so the footer
// only carries keyHint.

// here is the screen the user is on, as the key table files it: its section
// of the key sheet, and what q does there. q quits only from the grid and
// backs out one step elsewhere; explain closes on any key, so it has no q of
// its own.
func (m Model) here() (section, q string) {
	s := m.Config.Screen(m.shownScreen())
	return s.Sheet, s.Q
}

// keyHint is the footer's reminder of the two keys everyone needs: the key
// sheet, and what q does here.
//
// While the leader waits for its second key the hint shows the leader and
// nothing else, so a press is seen to have landed; what may follow is on the
// key sheet. It is padded to the usual hint's width: the footer is centred
// on this line and must not move for half a second.
func (m Model) keyHint() string {
	hint := "any key closes"
	if _, q := m.here(); q != "" {
		hint = m.Config.Keys.Help + " help · q " + q
	}
	if m.leader.pending {
		waiting := m.Config.Keys.Leader + " …"
		return waiting + strings.Repeat(" ", max(0, lipgloss.Width(hint)-lipgloss.Width(waiting)))
	}
	return hint
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

// sheetLayout is the order the sheet prints in: its sections, and in each the
// binds it shows, named by the screen of the key table they stand on. Only
// the order is the sheet's own; the keys and the words come from the table
// (settings/keymap.go), so the sheet cannot say something the handlers do not
// do.
var sheetLayout = []struct {
	title string
	gated bool // glassroot blocks the whole screen
	picks []sheetPick
}{
	{title: "GRID", picks: []sheetPick{
		{settings.Grid, settings.Confirm}, {settings.Grid, settings.Explain}, {settings.Grid, settings.Jump},
		{settings.Grid, settings.Move}, {settings.Grid, settings.PathMode}, {settings.Grid, settings.OpenInventory},
		{settings.Grid, settings.PinDeck}, {settings.Grid, settings.PrevDeck}, {settings.Grid, settings.NextDeck},
		{settings.GridLeader, settings.PickDeck}, {settings.Grid, settings.SwitchDeck},
		{settings.GridLeader, settings.Batch}, {settings.Grid, settings.Back},
	}},
	{title: "FOLDER", picks: []sheetPick{
		{settings.Folder, settings.MoveV}, {settings.Folder, settings.Jump}, {settings.Folder, settings.Confirm},
		{settings.Folder, settings.Explain}, {settings.FolderLeader, settings.Batch}, {settings.Folder, settings.Back},
	}},
	{title: "BATCH", gated: true, picks: []sheetPick{
		{settings.BatchMarking, settings.Mark}, {settings.BatchMarking, settings.Launch},
		{settings.BatchLayout, settings.MoveH}, {settings.BatchLayout, settings.MoveV}, {settings.BatchLayout, settings.Back},
	}},
	{title: "PATH", gated: true, picks: []sheetPick{
		{settings.PathBar, settings.MoveH}, {settings.PathBar, settings.Descend}, {settings.PathChildren, settings.MoveV},
		{settings.PathBar, settings.Confirm}, {settings.PathBar, settings.Search}, {settings.PathBar, settings.ToggleHidden},
		{settings.PathBar, settings.Back},
	}},
	{title: "INVENTORY", gated: true, picks: []sheetPick{
		{settings.Inventory, settings.MoveV}, {settings.Inventory, settings.MoveH}, {settings.Inventory, settings.Confirm},
		{settings.Inventory, settings.Specs}, {settings.Inventory, settings.Edit}, {settings.Inventory, settings.Trash},
		{settings.Inventory, settings.Back},
	}},
	{title: "EXPLAIN", picks: []sheetPick{
		{settings.ExplainPopup, settings.Scroll}, {settings.ExplainPopup, settings.Copy}, {settings.ExplainPopup, settings.Close},
	}},
	{title: "ANYWHERE", picks: []sheetPick{
		{settings.Grid, settings.Help}, {settings.Grid, settings.LockSession}, {settings.Grid, settings.Quit},
	}},
}

// A sheetPick is one bind of the key table: what act does on screen.
type sheetPick struct {
	screen settings.ScreenID
	act    settings.Action
}

// sheetSections is every screen's keys as the key table has them under cfg,
// so a remapped key shows as remapped. Glassroot leaves out what it blocks
// (glassroot.go), so the sheet never offers a key that does nothing.
func sheetSections(cfg settings.Config, glassroot bool) []sheetSection {
	var secs []sheetSection
	for _, l := range sheetLayout {
		if glassroot && l.gated {
			continue
		}
		sec := sheetSection{title: l.title, gated: l.gated}
		for _, p := range l.picks {
			b, _ := cfg.Screen(p.screen).Bind(p.act)
			if glassroot && b.Gated {
				continue
			}
			sec.bindings = append(sec.bindings, binding{keys: b.Label(), does: b.Does, gated: b.Gated})
		}
		secs = append(secs, sec)
	}
	return secs
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

// updateKeySheet scrolls the open sheet a line (up/down) or a page (PgUp/PgDn);
// any other key closes it and does nothing else.
func (m Model) updateKeySheet(msg tea.KeyMsg) Model {
	_, viewportH, maxOffset := m.sheetMetrics()
	offset, paged := pageOffset(msg.String(), m.sheet.offset, viewportH, maxOffset)
	switch {
	case paged:
	case IsUp(m.Config.Keys, msg):
		offset--
	case IsDown(m.Config.Keys, msg):
		offset++
	default:
		m.sheet = nil
		return m
	}
	m.sheet = &sheetState{offset: clampOffset(offset, maxOffset)}
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
	viewportH, maxOffset = fitRows(len(lines), m.termHeight-frame)
	return lines, viewportH, maxOffset
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

	body := m.scrollRows(lines, m.sheet.offset, viewportH, width)
	hint := "any key closes"
	if maxOffset > 0 {
		hint = "PgUp/PgDn scroll · any other key closes"
	}

	// The title and hint start in the ▸ column, one level out from the sections.
	raw := append([]string{m.styles.Title.Padding(0).Background(bg).Render("Keys"), ""}, body...)
	raw = append(raw, "", lipgloss.NewStyle().Background(bg).Foreground(m.styles.DropdownPopup.GetForeground()).Render(truncateText(hint, width)))
	return m.framePopup(raw)
}
