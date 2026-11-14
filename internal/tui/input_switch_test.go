package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/settings"
)

func TestDeckChordMatchesOnlyModifierDigit(t *testing.T) {
	cases := []struct {
		msg  tea.KeyMsg
		want bool
	}{
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3"), Alt: true}, true},
		{tea.KeyMsg{Type: tea.KeyF3, Alt: true}, false},
		{tea.KeyMsg{Type: tea.KeyF13, Alt: true}, false},
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")}, false},
	}
	for _, tc := range cases {
		grid := settings.Config{NumbModifier: "alt"}.Screen(settings.Grid)
		if got := grid.Act(tc.msg.String()) == settings.SwitchDeck; got != tc.want {
			t.Errorf("%q switches deck = %v, want %v", tc.msg.String(), got, tc.want)
		}
	}
}
