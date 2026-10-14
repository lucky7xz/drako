package ui

import (
	"os"
	"path/filepath"
	"testing"

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
