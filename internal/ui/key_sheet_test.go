package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/lucky7xz/drako/internal/config"
)

// sheetConfig is a config as a real session sees it: user keys, then defaults.
func sheetConfig(edit func(k *config.InputConfig)) config.Config {
	cfg := config.Config{}
	edit(&cfg.Keys)
	cfg.ApplyDefaults()
	return cfg
}

// binds reports whether section title lists keys for does.
func binds(secs []sheetSection, title, keys, does string) bool {
	for _, s := range secs {
		if s.title != title {
			continue
		}
		for _, b := range s.bindings {
			if b.keys == keys && b.does == does {
				return true
			}
		}
	}
	return false
}

// does reports whether section title has any binding for does.
func does(secs []sheetSection, title, what string) bool {
	for _, s := range secs {
		if s.title != title {
			continue
		}
		for _, b := range s.bindings {
			if b.does == what {
				return true
			}
		}
	}
	return false
}

func TestSheetShowsRemappedKeys(t *testing.T) {
	secs := sheetSections(sheetConfig(func(k *config.InputConfig) { k.Explain = "x" }), false)
	for _, title := range []string{"GRID", "FOLDER"} {
		if !binds(secs, title, "x", "explain") {
			t.Errorf("%s does not show the remapped explain key x", title)
		}
		if binds(secs, title, "e", "explain") {
			t.Errorf("%s still shows the default explain key e", title)
		}
	}
}

func TestSheetMoveKeysFollowTheDisableFlags(t *testing.T) {
	for _, tt := range []struct {
		name string
		edit func(k *config.InputConfig)
		want string
	}{
		{"all", func(k *config.InputConfig) {}, "←↑↓→ wasd hjkl"},
		{"no vim", func(k *config.InputConfig) { k.DisableVim = true }, "←↑↓→ wasd"},
		{"arrows only", func(k *config.InputConfig) { k.DisableVim, k.DisableWasd = true, true }, "←↑↓→"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if secs := sheetSections(sheetConfig(tt.edit), false); !binds(secs, "GRID", tt.want, "move") {
				t.Errorf("GRID move keys are not %q", tt.want)
			}
		})
	}
}

// Glassroot blocks path mode, the inventory, the profile lock, batch and copy;
// the sheet must not offer keys that do nothing.
func TestSheetHidesWhatGlassrootBlocks(t *testing.T) {
	secs := sheetSections(sheetConfig(func(*config.InputConfig) {}), true)
	for _, s := range secs {
		switch s.title {
		case "PATH", "INVENTORY", "BATCH":
			t.Errorf("glassroot sheet lists the %s section", s.title)
		}
	}
	for _, c := range []struct{ title, does string }{
		{"GRID", "path mode"}, {"GRID", "inventory"}, {"GRID", "pin deck"},
		{"GRID", "batch mode"}, {"FOLDER", "batch these"}, {"EXPLAIN", "copy"},
	} {
		if does(secs, c.title, c.does) {
			t.Errorf("glassroot sheet offers %s %q", c.title, c.does)
		}
	}
	if !does(secs, "GRID", "explain") || !does(secs, "GRID", "previous deck") {
		t.Error("glassroot allows explain and deck switching; the sheet should keep them")
	}
}

func TestSheetPutsThisScreenFirst(t *testing.T) {
	for _, tt := range []struct {
		name  string
		setup func(m *Model)
		want  string
	}{
		{"grid", func(m *Model) {}, "GRID"},
		{"path", func(m *Model) { m.mode = pathMode }, "PATH"},
		{"children", func(m *Model) { m.mode = childMode }, "PATH"},
		{"folder", func(m *Model) { m.mode = dropdownMode }, "FOLDER"},
		{"batch", func(m *Model) { m.mode = batchMode }, "BATCH"},
		{"layout dialog", func(m *Model) { m.mode, m.batch.tabs = dropdownMode, []int{2} }, "BATCH"},
		{"inventory", func(m *Model) { m.mode = inventoryMode }, "INVENTORY"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := createTestModelForView(gridMode)
			m.Config.ApplyDefaults()
			tt.setup(&m)
			secs := m.sheetHere()
			if secs[0].title != tt.want || !secs[0].here {
				t.Errorf("first section is %q (here %v), want %q marked here", secs[0].title, secs[0].here, tt.want)
			}
			for _, s := range secs[1:] {
				if s.here {
					t.Errorf("%s is marked here too", s.title)
				}
			}
			if n := len(secs); n != len(sheetSections(m.Config, false)) {
				t.Errorf("reordering changed the section count to %d", n)
			}
		})
	}
}

// sheetModel is a grid-mode model with real defaults, sized w×h.
func sheetModel(w, h int) Model {
	m := makeScrollTestModel(2, 2, w, h)
	m.Config.ApplyDefaults()
	return m
}

// sendKeys drives msgs through the full Update, as the program would.
func sendKeys(t *testing.T, m Model, msgs ...tea.KeyMsg) Model {
	t.Helper()
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(Model)
	}
	return m
}

func TestHelpKeyOpensTheSheetAndAnyKeyClosesIt(t *testing.T) {
	m := sendKeys(t, sheetModel(100, 40), keyRunes("?"))
	if m.sheet == nil {
		t.Fatal("? did not open the key sheet")
	}
	// The closing key only closes: i must not also open the inventory.
	m = sendKeys(t, m, keyRunes("i"))
	if m.sheet != nil {
		t.Error("i did not close the key sheet")
	}
	if m.mode != gridMode {
		t.Errorf("closing the sheet with i also acted on it: mode %v", m.mode)
	}
}

func TestHelpKeyIsRemappable(t *testing.T) {
	m := sheetModel(100, 40)
	m.Config.Keys.Help = "z"
	if sendKeys(t, m, keyRunes("?")).sheet != nil {
		t.Error("? opened the sheet although help is remapped to z")
	}
	if sendKeys(t, m, keyRunes("z")).sheet == nil {
		t.Error("the remapped help key z did not open the sheet")
	}
}

// Where ? is text, or another handler owns the next key, it is not the help key.
func TestHelpKeyLeavesOtherOwnersAlone(t *testing.T) {
	t.Run("path search", func(t *testing.T) {
		m := sheetModel(100, 40)
		m.mode, m.path.Searching = pathMode, true
		m = sendKeys(t, m, keyRunes("?"))
		if m.sheet != nil || m.path.Filter != "?" {
			t.Errorf("? in a search field: sheet open %v, filter %q", m.sheet != nil, m.path.Filter)
		}
	})
	t.Run("after the leader", func(t *testing.T) {
		m := sheetModel(100, 40)
		m.leader.pending = true
		if sendKeys(t, m, keyRunes("?")).sheet != nil {
			t.Error("? right after the leader opened the sheet; the sequence must swallow it")
		}
	})
	t.Run("explain popup", func(t *testing.T) {
		m := sheetModel(100, 40)
		m.mode, m.previousMode, m.activeDetail = infoMode, gridMode, &DetailState{Title: "x"}
		m = sendKeys(t, m, keyRunes("?"))
		if m.sheet != nil || m.mode != gridMode {
			t.Errorf("? in explain: sheet open %v, mode %v; it should just close explain", m.sheet != nil, m.mode)
		}
	})
}

func TestSheetScrollsAndClampsOnASmallTerminal(t *testing.T) {
	m := sendKeys(t, sheetModel(60, 24), keyRunes("?"))
	_, _, maxOffset := m.sheetMetrics()
	if maxOffset == 0 {
		t.Fatal("the whole sheet fits in 60x24; this test needs it to scroll")
	}
	for range 20 {
		m = sendKeys(t, m, tea.KeyMsg{Type: tea.KeyPgDown})
	}
	if m.sheet == nil || m.sheet.offset != maxOffset {
		t.Fatalf("after many PgDn the offset should rest at %d", maxOffset)
	}
	if !strings.Contains(ansi.Strip(m.View()), "ANYWHERE") {
		t.Error("scrolled to the end, but the last section is not on screen")
	}
	m = sendKeys(t, m, tea.KeyMsg{Type: tea.KeyUp})
	if m.sheet.offset != maxOffset-1 {
		t.Errorf("up should scroll one line, offset %d", m.sheet.offset)
	}
	for range 20 {
		m = sendKeys(t, m, tea.KeyMsg{Type: tea.KeyPgUp})
	}
	if m.sheet.offset != 0 {
		t.Errorf("after many PgUp the offset should rest at 0, got %d", m.sheet.offset)
	}
}

// The sheet floats over the screen: a grid wider than the sheet stays
// visible on both sides of it.
func TestSheetOverlaysTheScreen(t *testing.T) {
	m := makeScrollTestModel(9, 3, 200, 50)
	m.Config.ApplyDefaults()
	for r := range m.gridNav.grid {
		for c := range m.gridNav.grid[r] {
			m.gridNav.grid[r][c] = fmt.Sprintf("wide-cell-%d-%d", r, c)
		}
	}
	m = sendKeys(t, m, keyRunes("?"))
	out := ansi.Strip(m.View())
	for _, want := range []string{"▸ GRID", "wide-cell-0-0", "wide-cell-0-8"} {
		if !strings.Contains(out, want) {
			t.Errorf("sheet view is missing %q:\n%s", want, out)
		}
	}
}

func TestSheetFitsTheTerminal(t *testing.T) {
	for _, size := range [][2]int{{60, 24}, {80, 30}, {110, 45}} {
		w, h := size[0], size[1]
		m := sendKeys(t, sheetModel(w, h), keyRunes("?"))
		lines := strings.Split(m.View(), "\n")
		if len(lines) != h {
			t.Errorf("%dx%d: %d lines, want %d", w, h, len(lines), h)
		}
		maxAllowed := w + appStyle.GetHorizontalMargins()
		for _, ln := range lines {
			if lw := lipgloss.Width(ln); lw > maxAllowed {
				t.Errorf("%dx%d: line is %d wide (max %d): %q", w, h, lw, maxAllowed, ansi.Strip(ln))
			}
		}
		if out := ansi.Strip(m.View()); !strings.Contains(out, "Keys") {
			t.Errorf("%dx%d: the sheet is not on screen:\n%s", w, h, out)
		}
	}
}

// Each key sits in a [chip], entries are kept apart by " · ", one key per
// chip, and the screen you are on leads with ▸ while the others are indented.
func TestSheetDrawsKeysAsChips(t *testing.T) {
	m := sendKeys(t, sheetModel(110, 45), keyRunes("?"))
	out := ansi.Strip(m.renderKeySheet())
	for _, want := range []string{
		"▸ GRID", "  FOLDER",
		"[enter] run cell, open folder · [e] explain",
		"[o] previous deck · [p] next deck",
		"[esc q] close",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("sheet is missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "(here)") {
		t.Errorf("the current screen is marked with ▸ now, not (here):\n%s", out)
	}
}

// The sheet's own title and closing hint start where the ▸ does, one level
// out from the section text, so the popup reads as one aligned block.
func TestSheetTitleAndHintLineUpWithTheMarker(t *testing.T) {
	m := sendKeys(t, sheetModel(110, 45), keyRunes("?"))
	col := func(line string) int { return strings.Index(line, strings.TrimLeft(line, " ┃")) }
	var marker, title, hint int = -1, -1, -1
	for _, line := range strings.Split(ansi.Strip(m.renderKeySheet()), "\n") {
		switch text := strings.Trim(line, " ┃"); {
		case strings.HasPrefix(text, "▸"):
			marker = col(line)
		case text == "Keys":
			title = col(line)
		case strings.HasSuffix(text, "closes"):
			hint = col(line)
		}
	}
	if marker < 0 || title != marker || hint != marker {
		t.Errorf("columns: ▸ %d, Keys %d, hint %d; want all equal", marker, title, hint)
	}
}
