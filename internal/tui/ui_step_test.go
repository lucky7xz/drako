package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/lucky7xz/drako/internal/equip"
)

// uiModel is a grid of rows×3 cells on a w×h terminal, with the default keys
// and a real path, as a session has them.
func uiModel(t *testing.T, rows, w, h int) Model {
	t.Helper()
	m := makeScrollTestModel(3, rows, w, h)
	m.Config.ApplyDefaults()
	m.path = InitPathModel(t.TempDir())
	return m
}

// plain is the screen as text: what a terminal without colour shows.
func plain(m Model) []string {
	return strings.Split(ansi.Strip(m.View()), "\n")
}

func lineWith(lines []string, what string) (int, string) {
	for i, l := range lines {
		if strings.Contains(l, what) {
			return i, l
		}
	}
	return -1, ""
}

// L-01: the footer used to be shown whole or not at all, by an estimate of
// its height; at 80x24 a new user saw a grid and nothing else. Now the hint
// line is on screen at every size drako accepts, and the frame still fits.
func TestGridScreen_HintLineAtEverySize(t *testing.T) {
	for _, rows := range []int{1, 3, 9} {
		for _, w := range []int{60, 80, 120} {
			for h := 8; h <= 46; h++ {
				m := uiModel(t, rows, w, h)
				if small, _, _ := m.belowMinimum(); small {
					continue
				}
				lines := plain(m)
				if len(lines) != h {
					t.Errorf("%d rows at %dx%d: frame is %d lines", rows, w, h, len(lines))
				}
				if i, _ := lineWith(lines, "? help · q quit"); i < 0 {
					t.Errorf("%d rows at %dx%d: no hint line", rows, w, h)
				}
				if i, _ := lineWith(lines, "R0C0"); i < 0 {
					t.Errorf("%d rows at %dx%d: the grid is gone", rows, w, h)
				}
			}
		}
	}
}

// At 80x24, the size most terminals open at, a 3x3 deck has room for the
// whole footer.
func TestGridScreen_DefaultTerminalShowsTheWholeFooter(t *testing.T) {
	lines := plain(uiModel(t, 3, 80, 24))
	for _, want := range []string{"NET:", "HOST:", "? help", "R2C2"} {
		if i, _ := lineWith(lines, want); i < 0 {
			t.Errorf("80x24 should show %q:\n%s", want, strings.Join(lines, "\n"))
		}
	}
}

// What goes first when the terminal shrinks: the header art, then the blank
// lines, then NET, then the path bar. The hint line stays.
func TestGridScreen_FooterGivesWayInOrder(t *testing.T) {
	has := func(h int, what string) bool {
		i, _ := lineWith(plain(uiModel(t, 3, 80, h)), what)
		return i >= 0
	}
	path := "  /  " // the path bar's separators
	firstWith := func(what string) int {
		for h := 8; h <= 46; h++ {
			m := uiModel(t, 3, 80, h)
			if small, _, _ := m.belowMinimum(); !small && has(h, what) {
				return h
			}
		}
		return 99
	}
	hint, bar, net := firstWith("? help"), firstWith(path), firstWith("NET:")
	if !(hint <= bar && bar <= net) {
		t.Errorf("first shown at height: hint %d, path bar %d, NET %d; want hint, then path bar, then NET", hint, bar, net)
	}
	if hint >= 99 || net >= 99 {
		t.Errorf("hint (%d) and NET (%d) must both show at some height up to 46", hint, net)
	}
}

// A status message is the answer to the last key: it shows at the smallest
// size too.
func TestGridScreen_StatusMessageAtTheSmallestSize(t *testing.T) {
	for h := 8; h <= 30; h++ {
		m := uiModel(t, 3, 80, h)
		if small, _, _ := m.belowMinimum(); small {
			continue
		}
		m.profile.statusMessage = "Pinned core"
		lines := plain(m)
		if i, _ := lineWith(lines, "Pinned core"); i < 0 || len(lines) != h {
			t.Errorf("80x%d: status at line %d of %d", h, i, len(lines))
		}
	}
}

// Path mode works in the path bar and the folders under it, so those are on
// screen at every size, where the footer used to be hidden whole.
func TestPathMode_BarAndAFolderAtEverySize(t *testing.T) {
	for h := 8; h <= 30; h++ {
		m := uiModel(t, 3, 80, h)
		if small, _, _ := m.belowMinimum(); small {
			continue
		}
		m.path.ChildDirs = []string{"first-dir", "second-dir", "third-dir"}
		for _, mode := range []navMode{pathMode, childMode} {
			m.mode = mode
			lines := plain(m)
			bar, _ := lineWith(lines, "  /  ")
			dir, _ := lineWith(lines, "first-dir")
			if bar < 0 || dir < 0 || len(lines) != h {
				t.Errorf("80x%d mode %v: path bar at %d, first folder at %d, frame %d lines", h, mode, bar, dir, len(lines))
			}
		}
	}
}

// P-02: the footer was centred on its widest line, so it moved sideways as
// the path changed. Its left edge is the hint line's now.
func TestGridScreen_FooterKeepsItsLeftEdge(t *testing.T) {
	edge := func(m Model) int {
		_, l := lineWith(plain(m), "HOST:")
		return strings.Index(l, "HOST:")
	}
	m := uiModel(t, 3, 150, 40)
	want := edge(m)
	m.path.PathComponents = []string{"a"}
	short := edge(m)
	m.path.PathComponents = strings.Fields(strings.Repeat("a-rather-long-folder-name ", 12))
	m.path.SelectedPathIndex = 6
	long := edge(m)
	m.net.traffic = "↓ 999999.9kb/s ↑ 999999.9kb/s and then some more to be the widest line by far"
	busy := edge(m)
	if short != want || long != want || busy != want || want <= 0 {
		t.Errorf("HOST starts at column %d, then %d (short path), %d (long path), %d (long NET)", want, short, long, busy)
	}
}

// C-01: the cursor was a colour and nothing else. The selected cell is a
// closed box now, the only one on screen, and no name has moved.
func TestGridScreen_CursorIsABoxWithoutColour(t *testing.T) {
	m := uiModel(t, 3, 100, 30) // too short for the header art, which has corners of its own
	m.gridNav.cursorRow, m.gridNav.cursorCol = 1, 1
	lines := plain(m)
	screen := strings.Join(lines, "\n")
	for _, corner := range []string{"┏", "┓", "┗", "┛"} {
		if n := strings.Count(screen, corner); n != 1 {
			t.Errorf("want one %s on screen, got %d", corner, n)
		}
	}
	_, row := lineWith(lines, "R1C1")
	if !strings.Contains(row, "┃ R1C1") || strings.Count(row, "┃") != 2 {
		t.Errorf("the selected cell should stand between two ┃:\n%s", row)
	}
	// The names stand in the columns they stand in without the cursor there.
	m.gridNav.cursorRow = 2
	_, without := lineWith(plain(m), "R1C1")
	if strings.ReplaceAll(row, "┃", " ") != without {
		t.Errorf("the box moved a name:\n%q\n%q", row, without)
	}
}

// C-02 and the path bar: every cursor has a sign besides colour.
func TestCursorSigns_PathBarAndButtons(t *testing.T) {
	m := uiModel(t, 3, 100, 40)
	m.mode = pathMode
	m.path.PathComponents = []string{"home", "work", "deck"}
	m.path.SelectedPathIndex = 1
	if _, bar := lineWith(plain(m), "home"); !strings.Contains(bar, "home  / [work] /  deck") {
		t.Errorf("the selected folder should stand in brackets, the others where they were:\n%s", bar)
	}

	inv := stagedModel()
	inv.inventory.focusedList = focusApply
	out := strings.Join(plain(inv), "\n")
	if !strings.Contains(out, "► [ Apply Changes ]") || strings.Contains(out, "► [ Rescue Mode ]") {
		t.Errorf("the focused button carries the mark:\n%s", out)
	}
	// The mark takes the place of padding: the button does not move.
	at := func(m Model, what string) int {
		_, l := lineWith(plain(m), what)
		return lipgloss.Width(l[:strings.Index(l, what)])
	}
	focused := at(inv, "[ Apply Changes ]")
	inv.inventory.focusedList = focusRescue
	if plainAt := at(inv, "[ Apply Changes ]"); plainAt != focused {
		t.Errorf("the Apply button starts at column %d focused and %d not", focused, plainAt)
	}
}

// V-06: rows were wrapped as if every cell were as wide as the first, so a
// row of longer names ran off the right edge.
func TestInventory_RowsWrapByTheCellsOwnWidth(t *testing.T) {
	m := stagedModel()
	m.inventory.State.Visible = []string{"a.profile.toml"}
	for _, n := range []string{"a-much-longer-deck-name", "another-long-deck-name", "yet-another-deck-name", "and-one-more-long-name", "the-last-long-deck-name"} {
		m.inventory.State.Visible = append(m.inventory.State.Visible, n+equip.ProfileSuffix)
	}
	m.inventory.seen[0] = m.inventory.State.Visible
	for _, w := range []int{60, 80, 110, 150} {
		m.termWidth = w
		for i, l := range plain(m) {
			if lw := lipgloss.Width(l); lw > w {
				t.Errorf("width %d: line %d is %d wide: %q", w, i, lw, l)
			}
		}
		out := strings.Join(plain(m), "\n")
		for _, f := range m.inventory.State.Visible {
			if !strings.Contains(out, f) {
				t.Errorf("width %d: %s is cut off or missing", w, f)
			}
		}
	}
}

// V-06: the screen was centred on its widest line and moved sideways when a
// deck was lifted. Its left edge is the margin now.
func TestInventory_KeepsItsLeftEdge(t *testing.T) {
	edge := func(m Model) int {
		_, l := lineWith(plain(m), "Equipped Items")
		return strings.Index(l, "Equipped Items")
	}
	m := stagedModel()
	m.termWidth = 150
	before := edge(m)
	m = send(t, m, keyType(tea.KeyEnter), keyType(tea.KeyDown)) // lift alpha, carry it to the stash
	if after := edge(m); after != before || before > 4 {
		t.Errorf("Equipped Items starts at column %d, then %d with a deck lifted", before, after)
	}
}

// The inventory's footer carries the key hint, so it is there on a short
// terminal too, and the frame still fits.
func TestInventory_HintLineOnAShortTerminal(t *testing.T) {
	for h := 16; h <= 40; h++ {
		m := stagedModel()
		m.termWidth, m.termHeight = 80, h
		lines := plain(m)
		if i, _ := lineWith(lines, "? help · q back"); i < 0 || len(lines) != h {
			t.Errorf("80x%d: hint at line %d, frame %d lines", h, i, len(lines))
		}
	}
}
