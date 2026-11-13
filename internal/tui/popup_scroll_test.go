package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/lucky7xz/drako/internal/profile"
	"github.com/lucky7xz/drako/internal/settings"
)

func TestFitRowsAndPaging(t *testing.T) {
	if v, mx := fitRows(5, 20); v != 5 || mx != 0 {
		t.Errorf("rows that fit: got (%d, %d), want (5, 0)", v, mx)
	}
	if v, mx := fitRows(50, 20); v != 20 || mx != 30 {
		t.Errorf("rows that overflow: got (%d, %d), want (20, 30)", v, mx)
	}
	if v, _ := fitRows(50, -4); v != 3 {
		t.Errorf("a tiny terminal still shows 3 rows, got %d", v)
	}
	if next, ok := pageOffset("pgdown", 0, 20, 30); !ok || next != 19 {
		t.Errorf("PgDn = (%d, %v), want (19, true)", next, ok)
	}
	if next, _ := pageOffset("pgdown", 19, 20, 30); next != 30 {
		t.Errorf("PgDn clamps at the end, got %d", next)
	}
	if next, _ := pageOffset("pgup", 5, 20, 30); next != 0 {
		t.Errorf("PgUp clamps at the top, got %d", next)
	}
	if _, ok := pageOffset("q", 5, 20, 30); ok {
		t.Error("q is not a paging key")
	}
}

// D-01: a folder with more items than the terminal has rows used to lose its
// top off screen. The list now scrolls with the selector, which is always
// drawn, and the popup fits the terminal.
func TestDropdownScrollsWithItsSelector(t *testing.T) {
	items := make([]profile.CommandItem, 30)
	for i := range items {
		items[i] = profile.CommandItem{Name: fmt.Sprintf("item-%02d", i+1), Command: "true"}
	}
	cfg := settings.Config{X: 1, Y: 1, Theme: "default", Commands: []profile.Command{{Name: "f", Col: "a", Items: items}}}
	m := Model{Config: cfg, styles: BuildStyles(cfg, ""), mode: dropdownMode,
		termWidth: 100, termHeight: 20, dropdown: dropdownState{items: items}}

	for _, idx := range []int{0, 14, 29} {
		m.dropdown.selectedIdx = idx
		out := m.renderDropdownPopup()
		if h := lipgloss.Height(out); h > 20 {
			t.Errorf("selector on %d: popup is %d rows tall on a 20-row terminal", idx+1, h)
		}
		want := fmt.Sprintf("►  %d item-%02d", idx+1, idx+1)
		if !strings.Contains(ansi.Strip(out), want) {
			t.Errorf("selector on %d: %q is not on screen:\n%s", idx+1, want, ansi.Strip(out))
		}
	}

	m.termHeight = 60
	if out := ansi.Strip(m.renderDropdownPopup()); strings.Contains(out, "█") || !strings.Contains(out, "item-30") {
		t.Error("a list that fits shows every item and no scrollbar")
	}
}
