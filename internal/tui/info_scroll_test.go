package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
)

// scriptOf returns an n-line Value string for the explain overlay.
func scriptOf(n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = "line"
	}
	return strings.Join(lines, "\n")
}

func TestExplainLayout_ClampsAndOffset(t *testing.T) {
	m := Model{
		mode:       infoMode,
		termWidth:  100,
		termHeight: 60,
		activeDetail: &DetailState{
			Title:    "T",
			KeyLabel: "Command",
			Value:    scriptOf(50),
		},
	}
	l := m.explainLayout()
	if len(l.script) != 50 {
		t.Fatalf("script = %d lines, want 50", len(l.script))
	}
	if l.viewportH != infoViewportRows {
		t.Errorf("viewportH = %d, want %d on a tall terminal", l.viewportH, infoViewportRows)
	}
	if want := 50 - infoViewportRows; l.maxOffset != want {
		t.Errorf("maxOffset = %d, want %d", l.maxOffset, want)
	}
}

func TestExplainLayout_ShortFits(t *testing.T) {
	m := Model{
		termWidth:    100,
		termHeight:   60,
		activeDetail: &DetailState{Value: scriptOf(5)},
	}
	if l := m.explainLayout(); l.maxOffset != 0 {
		t.Errorf("maxOffset = %d, want 0 for a short script", l.maxOffset)
	}
}

func TestExplainLayout_ClampsToSmallTerminal(t *testing.T) {
	m := Model{
		termWidth:    100,
		termHeight:   16,
		activeDetail: &DetailState{Value: scriptOf(50)},
	}
	viewportH := m.explainLayout().viewportH
	if viewportH >= infoViewportRows {
		t.Errorf("viewportH = %d, want it clamped below %d on a short terminal", viewportH, infoViewportRows)
	}
	if viewportH < 3 {
		t.Errorf("viewportH = %d, want at least the floor of 3", viewportH)
	}
}

func TestUpdateInfoMode_Paging(t *testing.T) {
	m := Model{
		mode:         infoMode,
		previousMode: gridMode,
		termWidth:    100,
		termHeight:   60,
		activeDetail: &DetailState{Value: scriptOf(50)},
	}
	l := m.explainLayout()
	viewportH, maxOffset := l.viewportH, l.maxOffset

	// One PgDn advances by (viewportH-1) and keeps the overlay open.
	tm, _ := m.updateInfoMode(tea.KeyMsg{Type: tea.KeyPgDown})
	m = tm.(Model)
	if m.mode != infoMode {
		t.Fatalf("overlay closed on PgDn; mode = %v", m.mode)
	}
	if m.activeDetail.ScrollOffset != viewportH-1 {
		t.Errorf("offset = %d, want %d after one PgDn", m.activeDetail.ScrollOffset, viewportH-1)
	}

	// Repeated PgDn clamps at maxOffset.
	for range 20 {
		tm, _ = m.updateInfoMode(tea.KeyMsg{Type: tea.KeyPgDown})
		m = tm.(Model)
	}
	if m.activeDetail.ScrollOffset != maxOffset {
		t.Errorf("offset = %d, want clamp at maxOffset %d", m.activeDetail.ScrollOffset, maxOffset)
	}

	// Repeated PgUp floors at 0, overlay still open.
	for range 20 {
		tm, _ = m.updateInfoMode(tea.KeyMsg{Type: tea.KeyPgUp})
		m = tm.(Model)
	}
	if m.activeDetail.ScrollOffset != 0 {
		t.Errorf("offset = %d, want floor at 0", m.activeDetail.ScrollOffset)
	}
	if m.mode != infoMode {
		t.Fatalf("overlay closed during paging; mode = %v", m.mode)
	}
}

func TestUpdateInfoMode_OtherKeyCloses(t *testing.T) {
	m := Model{
		mode:         infoMode,
		previousMode: gridMode,
		termWidth:    100,
		termHeight:   60,
		activeDetail: &DetailState{Value: scriptOf(50)},
	}
	tm, _ := m.updateInfoMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = tm.(Model)
	if m.mode != gridMode {
		t.Errorf("mode = %v, want gridMode after a non-paging key", m.mode)
	}
	if m.activeDetail != nil {
		t.Error("activeDetail not cleared on close")
	}
}

func TestScrollbarColumn_ThumbPosition(t *testing.T) {
	plain := lipgloss.NewStyle()
	hasThumb := func(cell string) bool { return strings.Contains(cell, "█") }

	top := scrollbarColumn(50, 0, 18, plain, plain)
	if len(top) != 18 {
		t.Fatalf("len = %d, want 18", len(top))
	}
	if !hasThumb(top[0]) {
		t.Error("thumb should touch the top row at offset 0")
	}

	bottom := scrollbarColumn(50, 50-18, 18, plain, plain)
	if !hasThumb(bottom[17]) {
		t.Error("thumb should touch the bottom row at max offset")
	}

	thumbLen := 0
	for _, c := range top {
		if hasThumb(c) {
			thumbLen++
		}
	}
	if thumbLen < 1 || thumbLen > 18 {
		t.Errorf("thumb length = %d, want within [1,18]", thumbLen)
	}
}

// The explain popup shows the command as it will run: indentation and
// repeated spaces are part of the script, not prose to reflow.
func TestExplainLayout_KeepsScriptWhitespace(t *testing.T) {
	script := "if true; then\n    printf '%s    %s\\n' a b\nfi"
	m := Model{
		mode: infoMode, termWidth: 100, termHeight: 60,
		activeDetail: &DetailState{Title: "T", KeyLabel: "Command", Value: script},
	}
	if got := strings.Join(m.explainLayout().script, "\n"); got != script {
		t.Errorf("popup shows\n%s\nwant\n%s", got, script)
	}
}

func TestWrapScript_BreaksLongLinesAtSpaces(t *testing.T) {
	got := WrapScript("  echo aaaa bbbb", 11)
	want := []string{"  echo aaaa", "bbbb"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("WrapScript = %q, want %q", got, want)
	}
}
