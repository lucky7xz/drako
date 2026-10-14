package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
)

// The Rescue button shows the rescue grid, so nothing may still claim the deck
// it replaced: not the footer, not DRAKO_PROFILE (ActiveProfileName).
func TestRescueButtonLeavesNoActiveProfile(t *testing.T) {
	cfg := config.Config{X: 1, Y: 1}
	cfg.ApplyDefaults()
	m := Model{Config: cfg, styles: BuildStyles(cfg), mode: inventoryMode,
		profile: profileState{profiles: []config.ProfileInfo{{Name: "alpha"}, {Name: "beta"}}},
		inventory: inventoryModel{State: core.NewInventoryState(
			[]string{"alpha.profile.toml", "beta.profile.toml"}, nil, 9), focusedList: focusRescue}}

	next, _ := m.updateInventoryMode(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)

	if m.Config.Commands[0].Name != "Exit Rescue Mode" {
		t.Fatalf("setup: the rescue grid should be showing, got %q", m.Config.Commands[0].Name)
	}
	if got := m.profile.activeName(); got != "Rescue" {
		t.Errorf("footer says PROFILE: %s over the rescue grid", got)
	}
	if got := m.ActiveProfileName(); got != "" {
		t.Errorf("rescue cells would run with DRAKO_PROFILE=%s", got)
	}
}
