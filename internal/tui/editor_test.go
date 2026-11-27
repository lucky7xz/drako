package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucky7xz/drako/internal/equip"
	"github.com/lucky7xz/drako/internal/settings"
)

// editInventoryModel builds an inventory-mode model over a real temp config
// dir with one equipped and one stashed profile file.
func editInventoryModel(t *testing.T) (Model, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "inventory"), 0o755); err != nil {
		t.Fatal(err)
	}
	valid := "x = 1\ny = 1\n[[commands]]\nname = \"a\"\ncol = \"A\"\nrow = 0\n"
	os.WriteFile(filepath.Join(dir, "eq.profile.toml"), []byte(valid), 0o644)
	os.WriteFile(filepath.Join(dir, "inventory", "st.profile.toml"), []byte(valid), 0o644)

	m := Model{
		mode:    inventoryMode,
		profile: profileState{configDir: dir},
		Config:  settings.Config{Keys: settings.InputConfig{EditFile: "e", Delete: "delete", Lock: "r"}},
		inventory: inventoryModel{
			State:       equip.NewInventoryState([]string{"eq.profile.toml"}, []string{"st.profile.toml"}, equip.MaxEquipped),
			focusedList: equip.ListVisible,
		},
	}
	return m, dir
}

func TestInventoryEditGuards(t *testing.T) {
	t.Run("glassroot is a no-op", func(t *testing.T) {
		m, _ := editInventoryModel(t)
		m.GlassrootMode = true
		tm, cmd := m.updateInventoryMode(keyRune("e"))
		got := tm.(Model)
		if cmd != nil || got.mode != inventoryMode {
			t.Error("edit must be inert in glassroot")
		}
	})

	t.Run("held item blocks editing", func(t *testing.T) {
		m, _ := editInventoryModel(t)
		if err := m.inventory.State.PickUpItem(equip.ListVisible, 0); err != nil {
			t.Fatal(err)
		}
		tm, cmd := m.updateInventoryMode(keyRune("e"))
		got := tm.(Model)
		if cmd != nil {
			t.Error("expected no editor command while holding an item")
		}
		if !strings.Contains(got.inventory.status, "Place the held item") {
			t.Errorf("status = %q, want held-item message", got.inventory.status)
		}
	})

	t.Run("button focus is a no-op", func(t *testing.T) {
		m, _ := editInventoryModel(t)
		m.inventory.focusedList = 2 // apply button
		_, cmd := m.updateInventoryMode(keyRune("e"))
		if cmd != nil {
			t.Error("expected no editor command on a button row")
		}
	})

	// Staged moves are not on disk yet: st still sits in inventory/.
	t.Run("a staged deck opens from where it sits", func(t *testing.T) {
		m, _ := editInventoryModel(t)
		t.Setenv("VISUAL", "true")
		if err := m.inventory.State.PickUpItem(equip.ListInventory, 0); err != nil {
			t.Fatal(err)
		}
		if err := m.inventory.State.PlaceItem(equip.ListVisible, 1); err != nil {
			t.Fatal(err)
		}
		m.inventory.focusedList, m.inventory.cursor = equip.ListVisible, 1
		tm, cmd := m.updateInventoryMode(keyRune("e"))
		if cmd == nil {
			t.Fatalf("expected an editor command, status = %q", tm.(Model).inventory.status)
		}
	})

	t.Run("selected file launches the editor", func(t *testing.T) {
		m, _ := editInventoryModel(t)
		t.Setenv("VISUAL", "true") // /usr/bin/true: exits instantly if ever run
		_, cmd := m.updateInventoryMode(keyRune("e"))
		if cmd == nil {
			t.Fatal("expected an editor command for a valid selection")
		}
	})
}

func TestSelectedFile(t *testing.T) {
	m, dir := editInventoryModel(t)

	if rel, _, ok := m.inventory.selectedFile(dir); !ok || rel != "eq.profile.toml" {
		t.Errorf("equipped path = %q (%v)", rel, ok)
	}
	m.inventory.focusedList = equip.ListInventory
	if rel, _, ok := m.inventory.selectedFile(dir); !ok || rel != filepath.Join("inventory", "st.profile.toml") {
		t.Errorf("inventory path = %q (%v)", rel, ok)
	}
	m.inventory.focusedList = 3 // rescue button
	if _, _, ok := m.inventory.selectedFile(dir); ok {
		t.Error("button rows must not resolve to a path")
	}
}

// An edit that breaks a profile says so when the editor closes: a stashed one
// in the inventory's status line, an equipped one on its own error screen.
func TestAfterEditReportsABrokenProfile(t *testing.T) {
	t.Run("stashed", func(t *testing.T) {
		m, dir := openInventory(t)
		path := filepath.Join(dir, "inventory", "jukebox.profile.toml")
		if err := os.WriteFile(path, []byte("x = [\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		next, _ := m.afterEdit(editorFinishedMsg{path: path})
		got := next.(Model)
		if got.mode != inventoryMode || !strings.Contains(got.inventory.status, "jukebox.profile.toml has errors") {
			t.Errorf("mode %v, status %q; want the inventory saying jukebox has errors", got.mode, got.inventory.status)
		}
	})
	t.Run("equipped", func(t *testing.T) {
		m, dir := openInventory(t)
		path := filepath.Join(dir, "core.profile.toml")
		if err := os.WriteFile(path, []byte("x = [\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		next, _ := m.afterEdit(editorFinishedMsg{path: path})
		got := next.(Model)
		if got.mode != infoMode || got.activeDetail == nil || !strings.Contains(fmt.Sprint(*got.activeDetail), "core") {
			t.Errorf("mode %v, detail %v; want the error screen for core", got.mode, got.activeDetail)
		}
	})
}
