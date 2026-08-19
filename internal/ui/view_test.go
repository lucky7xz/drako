package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
	"github.com/lucky7xz/drako/internal/profiles"
)

// Helper to create a model for view testing
func createTestModelForView(mode navMode) Model {
	// Minimal config with small grid to avoid "Terminal too small" overlay
	cfg := config.Config{
		X:            2,
		Y:            2,
		Theme:        "default",
		DefaultShell: "/bin/bash",
	}

	// Create a grid
	grid := [][]string{
		{"Cmd1", "Cmd2"},
		{"Cmd3", "Cmd4"},
	}

	m := Model{
		mode:       mode,
		termWidth:  100, // Generous width
		termHeight: 50,  // Generous height
		Config:     cfg,
		gridNav:    gridNav{grid: grid},
		// Inventory model needed for inventory mode
		inventory: inventoryModel{
			State: core.NewInventoryState(
				[]string{"A.profile.toml"},
				[]string{"B.profile.toml"},
				profiles.MaxEquipped,
			),
			focusedList: 0,
		},
		profile: profileState{
			profiles: []config.ProfileInfo{
				{Name: "Core"},
			},
		},
	}

	// Force initialization of layout-dependent fields if any?
	// View() usually calculates layout on the fly.

	return m
}

func TestView_GridMode(t *testing.T) {
	m := createTestModelForView(gridMode)
	output := m.View()

	// Check for Grid-specific elements
	if !strings.Contains(output, "Grid Mode") {
		t.Errorf("View output missing 'Grid Mode' indicator. Got:\n%s", output)
	}
	if !strings.Contains(output, "Cmd1") {
		t.Error("View output missing grid content 'Cmd1'")
	}
	// Check for Header Column indicators (e.g., [A], [B])
	if !strings.Contains(output, "[A]") {
		t.Error("View output missing column header '[A]'")
	}
	// Check for Row Number indicators (e.g., 0❭)
	if !strings.Contains(output, "0❭") {
		t.Errorf("View output missing row number '0❭'. Got:\n%s", output)
	}
}

// makeScrollTestModel builds a grid-mode model with a cols×rows grid of
// "R<r>C<c>" cells and the given terminal size.
func makeScrollTestModel(cols, rows, termW, termH int) Model {
	grid := make([][]string, rows)
	for r := range grid {
		grid[r] = make([]string, cols)
		for c := range grid[r] {
			grid[r][c] = fmt.Sprintf("R%dC%d", r, c)
		}
	}
	cfg := config.Config{X: cols, Y: rows, Theme: "default", DefaultShell: "/bin/bash"}
	return Model{
		mode:       gridMode,
		termWidth:  termW,
		termHeight: termH,
		Config:     cfg,
		styles:     BuildStyles(cfg), // real styles: cell height must be measured honestly
		gridNav:    gridNav{grid: grid},
		profile:    profileState{profiles: []config.ProfileInfo{{Name: "Core"}}},
	}
}

func TestView_SmallGridNoScroll(t *testing.T) {
	m := makeScrollTestModel(2, 2, 100, 50)
	output := m.View()

	for _, cell := range []string{"R0C0", "R0C1", "R1C0", "R1C1"} {
		if !strings.Contains(output, cell) {
			t.Errorf("small grid should render every cell, missing %q", cell)
		}
	}
	for _, glyph := range []string{"▴", "▾", "◂", "▸"} {
		if strings.Contains(output, glyph) {
			t.Errorf("small grid should not show scroll marker %q", glyph)
		}
	}
}

func TestView_TallGridScrolls(t *testing.T) {
	tests := []struct {
		name      string
		cursorRow int
		contains  []string
		absent    []string
	}{
		{"cursor top", 0, []string{"R0C0", "▾"}, []string{"R9C0", "▴"}},
		{"cursor bottom", 9, []string{"R9C0", "▴"}, []string{"R0C0", "▾"}},
		{"cursor middle", 5, []string{"R5C0", "5❭", "▾ ▴"}, []string{"R0C0", "R9C0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := makeScrollTestModel(1, 10, 100, 24)
			m.gridNav.cursorRow = tt.cursorRow
			output := m.View()
			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q. Got:\n%s", want, output)
				}
			}
			for _, unwanted := range tt.absent {
				if strings.Contains(output, unwanted) {
					t.Errorf("output should not contain %q. Got:\n%s", unwanted, output)
				}
			}
		})
	}
}

func TestView_WideGridScrolls(t *testing.T) {
	tests := []struct {
		name      string
		cursorCol int
		contains  []string
		absent    []string
	}{
		{"cursor left", 0, []string{"[A]", "R0C0", "▸"}, []string{"[I]", "◂"}},
		{"cursor right", 8, []string{"[I]", "R0C8", "◂"}, []string{"[A]", "▸"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := makeScrollTestModel(9, 1, 60, 50)
			m.gridNav.cursorCol = tt.cursorCol
			output := m.View()
			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q. Got:\n%s", want, output)
				}
			}
			for _, unwanted := range tt.absent {
				if strings.Contains(output, unwanted) {
					t.Errorf("output should not contain %q. Got:\n%s", unwanted, output)
				}
			}
		})
	}
}

// The size gate floors the scroll window at 2x2 (capped by the grid): the
// overlay fires before the window could shrink to a useless single cell.
// For the R-cell model (footprint 8, row height 3, prefix 2) that boundary
// is exactly 24x13.
func TestView_WindowFloor(t *testing.T) {
	tests := []struct {
		name         string
		termW, termH int
		wantOverlay  bool
	}{
		{"one column short of 2x2", 23, 30, true},
		{"one line short of 2x2", 100, 12, true},
		{"exactly the 2x2 floor", 24, 13, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := makeScrollTestModel(9, 9, tt.termW, tt.termH)
			output := m.View()
			gotOverlay := strings.Contains(output, "Terminal too small")
			if gotOverlay != tt.wantOverlay {
				t.Fatalf("overlay = %v, want %v. Got:\n%s", gotOverlay, tt.wantOverlay, output)
			}
			if !tt.wantOverlay {
				for _, want := range []string{"0❭", "1❭", "[A]", "[B]", "▾", "▸"} {
					if !strings.Contains(output, want) {
						t.Errorf("floor render missing %q. Got:\n%s", want, output)
					}
				}
			}
		})
	}
}

// Wide-celled small grids (like the core profile: 3 columns of ~22-wide
// cells) must keep rendering a 2-column window in narrow terminals instead
// of hitting the overlay.
func TestView_WideCellsStillRenderNarrow(t *testing.T) {
	wide := strings.Repeat("x", 22)
	grid := [][]string{
		{wide, wide, wide},
		{wide, wide, wide},
		{wide, wide, wide},
	}
	cfg := config.Config{X: 3, Y: 3, Theme: "default", DefaultShell: "/bin/bash"}
	m := Model{
		mode:       gridMode,
		termWidth:  60,
		termHeight: 30,
		Config:     cfg,
		styles:     BuildStyles(cfg),
		gridNav:    gridNav{grid: grid},
		profile:    profileState{profiles: []config.ProfileInfo{{Name: "Core"}}},
	}
	output := m.View()
	if strings.Contains(output, "Terminal too small") {
		t.Fatalf("3x3 grid with wide cells at 60 cols should render a 2-col window, not the overlay. Got:\n%s", output)
	}
	for _, want := range []string{"[A]", "[B]", "▸"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q", want)
		}
	}
	if strings.Contains(output, "[C]") {
		t.Error("third column should be scrolled out at 60 cols")
	}
}

// Long cell text must never lock a profile out of a terminal that other
// profiles render in: cells compress to the available space (text
// ellipsizes) and the gate's width demand is capped at a constant.
func TestView_LongTextCompressesNotBlocks(t *testing.T) {
	long := strings.Repeat("x", 25)
	grid := [][]string{
		{long, long, long},
		{long, long, long},
		{long, long, long},
	}
	cfg := config.Config{X: 3, Y: 3, Theme: "default", DefaultShell: "/bin/bash"}
	m := Model{
		mode:       gridMode,
		termWidth:  44,
		termHeight: 30,
		Config:     cfg,
		styles:     BuildStyles(cfg),
		gridNav:    gridNav{grid: grid},
		profile:    profileState{profiles: []config.ProfileInfo{{Name: "Core"}}},
	}

	if _, reqW, _ := m.belowMinimum(); reqW != 32 {
		t.Errorf("gate width demand should cap at the constant 32, got %d", reqW)
	}

	output := m.View()
	if strings.Contains(output, "Terminal too small") {
		t.Fatalf("long-text 3x3 grid at 44 cols should compress, not overlay. Got:\n%s", output)
	}
	for _, want := range []string{"...", "[A]", "[B]", "▸"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q. Got:\n%s", want, output)
		}
	}
}

// truncateText must clip by rendered width, not per-rune arithmetic:
// emoji with variation selectors (⬆️ = ⬆ + VS16) fool rune-based counting,
// and an over-wide "truncated" string wraps inside its cell, breaking row
// alignment.
func TestTruncateTextEmojiWidth(t *testing.T) {
	tests := []struct {
		s   string
		max int
	}{
		{"⬆️ System Update", 14},
		{"🧹 Maintenance ⋮ ", 12},
		{"👀 Process Monitor", 8},
		{"plain ascii text", 9},
	}
	for _, tt := range tests {
		got := truncateText(tt.s, tt.max)
		if w := lipgloss.Width(got); w > tt.max {
			t.Errorf("truncateText(%q, %d) = %q renders %d wide", tt.s, tt.max, got, w)
		}
	}
}

func TestView_BothAxesScroll(t *testing.T) {
	m := makeScrollTestModel(9, 10, 60, 24)
	m.gridNav.cursorRow = 5
	m.gridNav.cursorCol = 4
	output := m.View()

	for _, want := range []string{
		"R5C4", "5❭", "[B]", "[G]",
		"▴", "▾", "◂", "▸",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q. Got:\n%s", want, output)
		}
	}
	for _, unwanted := range []string{"[A]", "[I]", "R0C0", "R9C8"} {
		if strings.Contains(output, unwanted) {
			t.Errorf("output should not contain %q. Got:\n%s", unwanted, output)
		}
	}
	if h := lipgloss.Height(output); h > m.termHeight+2 {
		t.Errorf("scrolled view is %d lines tall, must not exceed termHeight+appStyle margins = %d", h, m.termHeight+2)
	}
}

func TestView_InventoryMode(t *testing.T) {
	m := createTestModelForView(inventoryMode)
	output := m.View()

	if !strings.Contains(output, "Inventory Management") {
		t.Errorf("View output missing 'Inventory Management' title. Got:\n%s", output)
	}
	if !strings.Contains(output, "Equipped Items") {
		t.Error("View output missing 'Equipped Items' header")
	}
	if !strings.Contains(output, "Inventory Items") {
		t.Error("View output missing 'Inventory Items' header")
	}
	if !strings.Contains(output, "A.profile.toml") {
		t.Error("View output missing visible item 'A.profile.toml'")
	}
	if !strings.Contains(output, "[ Apply Changes ]") {
		t.Error("View output missing Apply button")
	}
}

func TestView_LockedMode(t *testing.T) {
	m := createTestModelForView(lockedMode)
	output := m.View()

	if !strings.Contains(output, "Session Locked") {
		t.Errorf("View output missing 'Session Locked' title. Got:\n%s", output)
	}
	if !strings.Contains(output, "Pump") {
		t.Error("View output missing 'Pump' instruction")
	}
}

// The active help line's wrap boundary is derived from the same W used to
// widen the header-hide threshold; picking a termW inside (W-10, W)
// exercises wrap without shift. The final render is appStyle.Render(...) of
// a Place()'d block: Place sizes content to exactly m.termWidth, then
// appStyle's own outer margin adds appStyle.GetHorizontalMargins() more —
// that overhead is pre-existing (present at any width, unrelated to this
// feature), so the bound accounts for it rather than asserting termW exactly.
func TestView_NarrowTerminalWrapsHelpText(t *testing.T) {
	helpText := "Grid Mode | Enter: Select, e: Explain, Tab: Path, r: Start-Lock, i: Inventory"
	W := lipgloss.Width(helpText) + LayoutSideMargin
	termW := W - 5
	m := makeScrollTestModel(1, 1, termW, 30)
	output := m.View()

	maxAllowed := termW + appStyle.GetHorizontalMargins()
	for _, ln := range strings.Split(output, "\n") {
		if w := lipgloss.Width(ln); w > maxAllowed {
			t.Errorf("line exceeds allowed width %d: %q (%d wide)", maxAllowed, ln, w)
		}
	}
	if !strings.Contains(output, "Grid Mode") || !strings.Contains(output, "Inventory") {
		t.Error("wrapped help text should still contain its content, just split across lines")
	}
}

// Below W-10 the block should hug the left edge instead of leaving centered
// slack: the block's widest line should trail only appStyle's static outer
// margin, not a variable centering offset. Narrower sibling lines (e.g. the
// profile counter, centered relative to the grid by an inner JoinVertical)
// are unaffected by this outer-Place change and may still show their own
// relative indentation — this checks the widest/dominant line specifically,
// via the minimum leading whitespace across all non-empty lines.
func TestView_ShiftLeftHugsLeftEdge(t *testing.T) {
	helpText := "Grid Mode | Enter: Select, e: Explain, Tab: Path, r: Start-Lock, i: Inventory"
	W := lipgloss.Width(helpText) + LayoutSideMargin
	termW := W - 15
	m := makeScrollTestModel(1, 1, termW, 30)
	output := m.View()

	minLeading := -1
	for _, ln := range strings.Split(output, "\n") {
		stripped := ansi.Strip(ln)
		trimmed := strings.TrimLeft(stripped, " ")
		if trimmed == "" {
			continue
		}
		if leading := len(stripped) - len(trimmed); minLeading == -1 || leading < minLeading {
			minLeading = leading
		}
	}
	if maxStaticMargin := appStyle.GetHorizontalMargins()/2 + 1; minLeading > maxStaticMargin {
		t.Errorf("expected some line to hug the left edge (leading <= %d, appStyle's static margin), smallest leading was %d", maxStaticMargin, minLeading)
	}
}

// The NET/STATUS/THEME line must truncate, not overflow or wrap-split ANSI,
// when net.traffic carries an unusually long already-colored string.
func TestView_StatusLineTruncatesNotOverflows(t *testing.T) {
	m := makeScrollTestModel(1, 1, 40, 30)
	m.net.traffic = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).
		Render("↓ 999999.9kb/s ↑ 999999.9kb/s way too long for 40 cols")
	m.net.online = "online (active)"
	output := m.View()

	// Scoped to the NET/STATUS/THEME line's own content specifically —
	// renderCombinedFooter truncates that one. Its rendered line gets
	// right-padded with trailing spaces to match the footer block's widest
	// sibling line (lipgloss.JoinVertical rectangularizes the block), so
	// trailing spaces are trimmed before measuring; profileBar/pathBar/
	// childDirs are separate, pre-existing, out-of-scope elements that may
	// be wider and drive that padding independently of this truncation.
	availWidth := m.termWidth - LayoutSideMargin
	found := false
	for _, ln := range strings.Split(output, "\n") {
		if !strings.Contains(ln, "NET:") {
			continue
		}
		found = true
		// TrimSpace on both sides: trailing space is JoinVertical's block
		// padding, leading space is the outer static margin / left-hug
		// positioning — neither is part of the truncated content itself.
		content := strings.TrimSpace(ansi.Strip(ln))
		if w := lipgloss.Width(content); w > availWidth {
			t.Errorf("status line content exceeds available width %d: %q (%d wide)", availWidth, content, w)
		}
		if !strings.HasSuffix(content, "...") {
			t.Errorf("expected truncated status line to end in '...', got: %q", content)
		}
	}
	if !found {
		t.Fatal("expected a NET: status line in the output")
	}
}

// The counter used to clamp both numbers to 9, so cycling past the ninth
// profile froze it at "< 9 / 9 >" while the PROFILE: name kept moving.
func TestRenderProfileCounter(t *testing.T) {
	cases := []struct {
		name        string
		total       int
		activeIndex int
		want        string
	}{
		{name: "under the cap", total: 4, activeIndex: 2, want: "< 3 / 4 >"},
		{name: "at the cap", total: 9, activeIndex: 8, want: "< 9 / 9 >"},
		{name: "past the cap is flagged", total: 15, activeIndex: 11, want: "< 12 / 15 ⚠ >"},
		{name: "no profiles is the rescue grid", total: 0, activeIndex: 0, want: "< 0 / 0 ⚠ >"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Config{X: 2, Y: 2, Theme: "default"}
			m := Model{
				Config:  cfg,
				styles:  BuildStyles(cfg),
				profile: profileState{profiles: make([]config.ProfileInfo, tc.total), activeIndex: tc.activeIndex},
			}
			got := strings.TrimSpace(ansi.Strip(m.renderProfileCounter()))
			if got != tc.want {
				t.Errorf("counter = %q, want %q", got, tc.want)
			}
		})
	}
}
