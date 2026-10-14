package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
)

func TestIsProfileSwitchMatchesOnlyModifierDigit(t *testing.T) {
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
		if got, _ := IsProfileSwitch(config.InputConfig{}, tc.msg, "alt"); got != tc.want {
			t.Errorf("IsProfileSwitch(%q) = %v, want %v", tc.msg.String(), got, tc.want)
		}
	}
}
