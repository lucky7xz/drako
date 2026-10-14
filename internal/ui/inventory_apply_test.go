package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
)

// Apply commits the arrangement on screen when it was pressed. Keys handled
// before bubbletea runs the Cmd must not change what gets written to disk.
func TestApplyUsesArrangementAtPressTime(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"alpha", "beta"} {
		if err := os.WriteFile(filepath.Join(dir, name+".profile.toml"), []byte("x = 1\ny = 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	inv := inventoryModel{State: core.NewInventoryState(
		[]string{"alpha.profile.toml", "beta.profile.toml"}, nil, 9)}

	cmd := ApplyInventoryChangesCmd(dir, inv)
	if err := inv.State.PickUpItem(core.ListVisible, 1); err != nil { // user lifts beta
		t.Fatal(err)
	}
	if msg := cmd(); msg != (reloadProfilesMsg{}) {
		t.Fatalf("apply failed: %#v", msg)
	}

	if _, err := os.Stat(filepath.Join(dir, "beta.profile.toml")); err != nil {
		t.Errorf("beta was equipped when Apply was pressed but got stashed: %v", err)
	}
}

// A lifted profile is in neither list, so applying would stash it. Apply waits
// for the item to be placed, like edit, delete and the spec picker do.
func TestApplyRefusedWhileHolding(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"alpha", "beta"} {
		if err := os.WriteFile(filepath.Join(dir, name+".profile.toml"), []byte("x = 1\ny = 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.Config{X: 1, Y: 1}
	cfg.ApplyDefaults()
	m := Model{Config: cfg, styles: BuildStyles(cfg), mode: inventoryMode,
		profile: profileState{configDir: dir},
		inventory: inventoryModel{State: core.NewInventoryState(
			[]string{"alpha.profile.toml", "beta.profile.toml"}, nil, 9)}}
	if err := m.inventory.State.PickUpItem(core.ListVisible, 1); err != nil { // lift beta
		t.Fatal(err)
	}
	m.inventory.focusedList = focusApply

	next, cmd := m.updateInventoryMode(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		cmd()
	}
	if _, err := os.Stat(filepath.Join(dir, "beta.profile.toml")); err != nil {
		t.Errorf("Apply while holding beta stashed it: %v", err)
	}
	if next.(Model).inventory.status == "" {
		t.Error("a refused Apply should say why")
	}
}

// The inventory rebuild after an edit can fail (here inventory/ is a file);
// afterEdit must show that error, not dereference the missing state.
func TestAfterEditSurvivesUnreadableInventory(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	dir := filepath.Join(xdg, "drako")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(dir, "good.profile.toml")
	if err := os.WriteFile(good, []byte("x = 1\ny = 1\n[[commands]]\nname = \"hi\"\ncommand = \"echo hi\"\nrow = 0\ncol = \"a\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := InitialModel(false, Session{})
	if err := os.RemoveAll(filepath.Join(dir, "inventory")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "inventory"), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.mode, m.inventory.focusedList = inventoryMode, 0 // cursor on the equipped list

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("afterEdit panicked: %v", r)
		}
	}()
	next, _ := m.afterEdit(editorFinishedMsg{path: good})
	if next.(Model).inventory.err == nil {
		t.Error("the failed rebuild should surface as the inventory error")
	}
}
