package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/launch"
	"github.com/muesli/termenv"
)

// screens builds one model per screen and popup, on a cols-wide grid sized
// w×h, so every way drako can draw itself is checked the same way.
func screens(t *testing.T, cols, w, h int) map[string]Model {
	t.Helper()
	grid := func(mode navMode) Model {
		m := makeScrollTestModel(cols, 3, w, h)
		m.Config.ApplyDefaults()
		m.mode = mode
		return m
	}
	out := map[string]Model{
		"grid":  grid(gridMode),
		"path":  grid(pathMode),
		"child": grid(childMode),
		"batch": grid(batchMode),
		"lock":  grid(lockedMode),
	}

	dropdown := grid(dropdownMode)
	dropdown.dropdown.items = []config.CommandItem{{Name: "first-item", Command: "true"}, {Name: "second-item", Command: "true"}}
	out["dropdown"] = dropdown

	layout := grid(batchMode)
	layout.batch.marked = []launch.Selection{{Cell: "R0C0"}, {Cell: "R0C1"}}
	layout.batch.tabs = []int{2}
	out["layout dialog"] = layout

	explain := grid(infoMode)
	explain.previousMode = gridMode
	explain.activeDetail = &DetailState{Title: "the-explained-cell", KeyLabel: "Command", Value: "echo explained"}
	out["explain"] = explain

	inv, _ := specsInventoryModel(t, []string{"eq"}, []string{"st"}, map[string]string{"dev": specList("st")})
	inv.termWidth, inv.termHeight = w, h
	out["inventory"] = inv
	out["specs"] = send(t, inv, keyType(tea.KeyTab))
	return out
}

// appStyle's margin sits inside the terminal: every frame is exactly the
// terminal's size, so the renderer never has to crop a line or a column.
func TestView_EveryScreenIsExactlyTheTerminal(t *testing.T) {
	for _, size := range [][2]int{{60, 24}, {80, 30}, {110, 45}} {
		w, h := size[0], size[1]
		for name, m := range screens(t, 3, w, h) {
			lines := strings.Split(m.View(), "\n")
			if len(lines) != h {
				t.Errorf("%s at %dx%d: %d lines, want %d", name, w, h, len(lines), h)
			}
			for i, ln := range lines {
				if lw := lipgloss.Width(ln); lw > w {
					t.Errorf("%s at %dx%d: line %d is %d wide", name, w, h, i, lw)
					break
				}
			}
		}
	}
}

// A popup floats over its screen: next to a popup narrower than the grid, the
// grid's first and last cells stay on screen, and so does the popup's content.
func TestView_PopupsFloatOverTheScreen(t *testing.T) {
	all := screens(t, 9, 200, 50)
	for name, want := range map[string]string{
		"dropdown":      "first-item",
		"layout dialog": "tabs",
		"explain":       "echo explained",
	} {
		m := all[name]
		for r := range m.gridNav.grid {
			for c := range m.gridNav.grid[r] {
				m.gridNav.grid[r][c] = fmt.Sprintf("wide-cell-%d-%d", r, c)
			}
		}
		out := ansi.Strip(m.View())
		for _, s := range []string{want, "wide-cell-0-0", "wide-cell-0-8"} {
			if !strings.Contains(out, s) {
				t.Errorf("%s: %q is not on screen:\n%s", name, s, out)
			}
		}
	}

	// The picker covers the middle of the inventory; the inventory's own
	// footer below it is what shows the screen is still there.
	out := ansi.Strip(all["specs"].View())
	for _, s := range []string{"Specs Found", config.AppName} {
		if !strings.Contains(out, s) {
			t.Errorf("specs: %q is not on screen:\n%s", s, out)
		}
	}
}

// The grid behind a dropdown looks exactly like it did before the dropdown
// opened, so the open folder's cell stays selected. The selected and plain
// cells differ only in colour, so the comparison needs a colour profile.
func TestView_FolderStaysSelectedBehindItsDropdown(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	m := makeScrollTestModel(3, 3, 100, 40)
	m.gridNav.cursorRow, m.gridNav.cursorCol = 1, 1
	before := m.renderGrid(30)
	m.mode = dropdownMode
	if m.renderGrid(30) != before {
		t.Error("the grid changed when its dropdown opened; the folder's cell should stay selected")
	}
}

// previousMode can be stale (the broken-profile notes never set it), so a
// dropdown that already closed must not come back as an empty box.
func TestView_StaleDropdownModeDrawsNoEmptyPopup(t *testing.T) {
	m := makeScrollTestModel(3, 3, 100, 40)
	m.Config.ApplyDefaults()
	m.mode, m.previousMode = infoMode, dropdownMode
	m.activeDetail = &DetailState{Title: "note", KeyLabel: "Error", Value: "x"}
	if n := len(m.popups(dropdownMode)); n != 1 {
		t.Errorf("%d popups over the screen, want just the note", n)
	}
}

// A profile note returns to the grid when it is dismissed, so it floats over
// the grid, not over a folder the reload behind it may have replaced. (The
// note is bigger than a dropdown and would hide it, so the decision itself is
// what is checked.)
func TestView_ProfileNoteFloatsOverTheGrid(t *testing.T) {
	m := makeScrollTestModel(3, 3, 100, 40)
	m.Config.ApplyDefaults()
	m.mode, m.previousMode = infoMode, dropdownMode
	m.dropdown.items = []config.CommandItem{{Name: "stale-item", Command: "true"}}
	m.profile.errorQueueActive = true
	m.activeDetail = &DetailState{Title: "Profile error: x", KeyLabel: "Error", Value: "bad"}
	if got := m.underneath(); got != gridMode {
		t.Errorf("a profile note floats over mode %v, want the grid", got)
	}
	if n := len(m.popups(m.underneath())); n != 1 {
		t.Errorf("%d popups, want just the note", n)
	}
}

// The inventory footer trims to the terminal, as the HOST bar does, and keeps
// its hint.
func TestView_InventoryFooterFitsANarrowTerminal(t *testing.T) {
	m, _ := specsInventoryModel(t, []string{"eq"}, []string{"st"}, nil)
	m.Config.ApplyDefaults()
	for _, w := range []int{32, 40} {
		m.termWidth, m.termHeight = w, 30
		out := m.View()
		for i, ln := range strings.Split(out, "\n") {
			if lipgloss.Width(ln) > w {
				t.Errorf("%d cols: line %d is %d wide: %q", w, i, lipgloss.Width(ln), ansi.Strip(ln))
				break
			}
		}
		if !strings.Contains(ansi.Strip(out), "q back") {
			t.Errorf("%d cols: the footer lost its hint", w)
		}
	}
}

// Explain fits a narrow terminal, and a long path keeps its end, the part
// that says where the command runs.
func TestView_ExplainFitsANarrowTerminal(t *testing.T) {
	m := makeScrollTestModel(1, 1, 30, 30)
	m.Config.ApplyDefaults()
	m.mode, m.previousMode = infoMode, gridMode
	m.activeDetail = &DetailState{
		Title: "cell", KeyLabel: "Command", Value: strings.Repeat("echo something long; ", 20),
		Meta: []DetailMeta{{Label: "CWD", Value: "/home/user/projects/client/very/deep/repo"}},
	}
	out := m.View()
	for i, ln := range strings.Split(out, "\n") {
		if lipgloss.Width(ln) > 30 {
			t.Errorf("line %d is %d wide: %q", i, lipgloss.Width(ln), ansi.Strip(ln))
			break
		}
	}
	if !strings.Contains(ansi.Strip(out), "deep/repo") {
		t.Errorf("the path lost its end:\n%s", ansi.Strip(out))
	}
}

// A popup is just its box: it floats over the screen, so a margin around it
// would only blank a ring of whatever the screen shows there.
func TestPopupsAreJustTheirBox(t *testing.T) {
	all := screens(t, 3, 110, 45)
	sheet := sendKeys(t, sheetModel(110, 45), keyRunes("?"))
	for name, popup := range map[string]string{
		"dropdown":      all["dropdown"].renderDropdownPopup(),
		"layout dialog": all["layout dialog"].renderLayoutPopup(),
		"explain":       all["explain"].renderInfoPopup(),
		"specs":         all["specs"].renderSpecsPopup(),
		"key sheet":     sheet.renderKeySheet(),
	} {
		lines := strings.Split(ansi.Strip(popup), "\n")
		first, last := lines[0], lines[len(lines)-1]
		if !strings.HasPrefix(first, "┏") || !strings.HasPrefix(last, "┗") {
			t.Errorf("%s: the box does not start at its edges:\n%q\n…\n%q", name, first, last)
		}
	}
}

// However much explain has to say, its popup never outgrows the screen: the
// script window gets whatever the pinned rows leave.
func TestExplainNeverTallerThanTheScreen(t *testing.T) {
	for _, size := range [][2]int{{60, 24}, {80, 30}, {30, 30}} {
		w, h := size[0], size[1]
		m := makeScrollTestModel(1, 1, w, h)
		m.Config.ApplyDefaults()
		m.mode, m.previousMode = infoMode, gridMode
		m.activeDetail = &DetailState{
			Title: "a cell", KeyLabel: "Command", Value: scriptOf(60),
			Description: "Does a thing, and explains at some length why it does it that way.",
			Meta:        []DetailMeta{{Label: "Address", Value: "core:A0"}, {Label: "CWD", Value: "/home/user/somewhere"}},
		}
		if got := lipgloss.Height(m.renderInfoPopup()); got > h {
			t.Errorf("%dx%d: explain is %d rows tall", w, h, got)
		}
	}
}
