package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
	"github.com/lucky7xz/drako/internal/profiles"
)

func TestResolveEditor(t *testing.T) {
	t.Run("VISUAL wins over EDITOR", func(t *testing.T) {
		t.Setenv("VISUAL", "visual-ed")
		t.Setenv("EDITOR", "other-ed")
		ed, err := resolveEditor()
		if err != nil || ed[0] != "visual-ed" {
			t.Errorf("got %v (%v), want visual-ed", ed, err)
		}
	})
	t.Run("EDITOR with arguments splits", func(t *testing.T) {
		t.Setenv("VISUAL", "")
		t.Setenv("EDITOR", "code -w")
		ed, err := resolveEditor()
		if err != nil || len(ed) != 2 || ed[0] != "code" || ed[1] != "-w" {
			t.Errorf("got %v (%v), want [code -w]", ed, err)
		}
	})
	t.Run("falls back to a common editor", func(t *testing.T) {
		t.Setenv("VISUAL", "")
		t.Setenv("EDITOR", "")
		ed, err := resolveEditor()
		// The machine may or may not have nano/vim; either a fallback or
		// the set-$EDITOR error is acceptable — never an empty success.
		if err == nil && len(ed) == 0 {
			t.Error("empty editor with nil error")
		}
	})
}

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
		Config:  config.Config{Keys: config.InputConfig{EditFile: "e", Delete: "delete", Lock: "r"}},
		inventory: inventoryModel{
			State:       core.NewInventoryState([]string{"eq.profile.toml"}, []string{"st.profile.toml"}, profiles.MaxEquipped),
			focusedList: core.ListVisible,
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
		if err := m.inventory.State.PickUpItem(core.ListVisible, 0); err != nil {
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

	t.Run("selected file launches the editor", func(t *testing.T) {
		m, _ := editInventoryModel(t)
		t.Setenv("VISUAL", "true") // /usr/bin/true: exits instantly if ever run
		_, cmd := m.updateInventoryMode(keyRune("e"))
		if cmd == nil {
			t.Fatal("expected an editor command for a valid selection")
		}
	})
}

func TestSelectedFilePath(t *testing.T) {
	m, dir := editInventoryModel(t)

	if p, ok := m.inventory.selectedFilePath(dir); !ok || p != filepath.Join(dir, "eq.profile.toml") {
		t.Errorf("equipped path = %q (%v)", p, ok)
	}
	m.inventory.focusedList = core.ListInventory
	if p, ok := m.inventory.selectedFilePath(dir); !ok || p != filepath.Join(dir, "inventory", "st.profile.toml") {
		t.Errorf("inventory path = %q (%v)", p, ok)
	}
	m.inventory.focusedList = 3 // rescue button
	if _, ok := m.inventory.selectedFilePath(dir); ok {
		t.Error("button rows must not resolve to a path")
	}
}
