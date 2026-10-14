package ui

import (
	"testing"

	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
)

// Trashing an equipped profile renames a root .toml, so a reload follows. It
// must not throw away the inventory screen, its verdict, or the staged lists.
func TestReloadKeepsTheInventoryOpen(t *testing.T) {
	cfg := config.Config{X: 1, Y: 1}
	cfg.ApplyDefaults()
	m := Model{Config: cfg, styles: BuildStyles(cfg), mode: inventoryMode,
		inventory: inventoryModel{
			State:  core.NewInventoryState([]string{"b.profile.toml", "a.profile.toml"}, nil, 9),
			status: "Trashed c → trash/", statusOK: true,
		}}

	m = m.applyReloadedBundle(config.ConfigBundle{Config: cfg})

	if m.mode != inventoryMode {
		t.Fatalf("a clean reload left the inventory, mode = %v", m.mode)
	}
	if m.inventory.status != "Trashed c → trash/" || m.inventory.State.Visible[0] != "b.profile.toml" {
		t.Errorf("verdict or staged order lost: %q %v", m.inventory.status, m.inventory.State.Visible)
	}
}
