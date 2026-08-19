package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
)

// Stashing the locked profile would leave pivot.toml aimed at something no
// longer equipped, which drops the next launch into the rescue grid — and it
// was also a way around the delete guard: stash, apply, let the reload clear
// the stale lock, then delete freely.
func TestInventoryLock_RefusesToLiftTheLockedProfile(t *testing.T) {
	m, _ := editInventoryModel(t)
	m.profile.pivotName = "eq"
	m.inventory.focusedList = core.ListVisible

	m = sendTop(t, m, keyType(tea.KeySpace))

	if m.inventory.State.HeldItem != nil {
		t.Error("the locked profile must not be liftable")
	}
	if len(m.inventory.State.Visible) != 1 || m.inventory.State.Visible[0] != "eq.profile.toml" {
		t.Errorf("Visible = %v, want it untouched", m.inventory.State.Visible)
	}
	if !strings.Contains(m.inventory.status, "Locked") {
		t.Errorf("status = %q, want it to mention the lock", m.inventory.status)
	}
	// The remedy has to point at the grid, since r no longer works here.
	if !strings.Contains(m.inventory.status, "grid") {
		t.Errorf("status = %q, want it to send the user to the grid", m.inventory.status)
	}
}

func TestInventoryLock_UnlockedProfileStillLifts(t *testing.T) {
	m, _ := editInventoryModel(t)
	m.profile.pivotName = "something-else"
	m.inventory.focusedList = core.ListVisible

	m = sendTop(t, m, keyType(tea.KeySpace))

	if m.inventory.State.HeldItem == nil {
		t.Fatal("an unlocked profile must still lift")
	}
	if *m.inventory.State.HeldItem != "eq.profile.toml" {
		t.Errorf("held = %q, want eq.profile.toml", *m.inventory.State.HeldItem)
	}
}

// The lock acts on the *active* profile, so offering it from the inventory
// would pin something other than the item under the cursor.
func TestInventoryLock_RKeyIsGridOnly(t *testing.T) {
	t.Run("inventory ignores it", func(t *testing.T) {
		m, dir := editInventoryModel(t)

		m = sendTop(t, m, keyRune("r"))

		if m.profile.pivotName != "" {
			t.Errorf("pivotName = %q, want it unchanged", m.profile.pivotName)
		}
		if _, err := os.Stat(filepath.Join(dir, "pivot.toml")); !os.IsNotExist(err) {
			t.Error("r in the inventory must not write pivot.toml")
		}
	})

	t.Run("the grid still toggles", func(t *testing.T) {
		m, dir := editInventoryModel(t)
		m.mode = gridMode
		m.gridNav = gridNav{grid: [][]string{{"A"}}}
		m.profile.profiles = []config.ProfileInfo{{Name: "eq"}}

		m = sendTop(t, m, keyRune("r"))

		if m.profile.pivotName != "eq" {
			t.Errorf("pivotName = %q, want %q", m.profile.pivotName, "eq")
		}
		if _, err := os.Stat(filepath.Join(dir, "pivot.toml")); err != nil {
			t.Errorf("the grid should have written pivot.toml: %v", err)
		}
	})
}
