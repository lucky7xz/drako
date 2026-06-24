package profiles

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/lucky7xz/drako/internal/paths"
)

func TestPlanReconcile_EquipsDesiredFromInventory(t *testing.T) {
	moves := planReconcile(
		nil,                          // equipped: none
		[]string{"git.profile.toml"}, // inventory
		[]string{"git"},              // desired
	)

	want := []move{{File: "git.profile.toml", From: Inventory, To: Equipped}}
	if !reflect.DeepEqual(moves, want) {
		t.Errorf("got %+v, want %+v", moves, want)
	}
}

func TestPlanReconcile_StashesEquippedNotDesired(t *testing.T) {
	moves := planReconcile(
		[]string{"git.profile.toml", "docker.profile.toml"}, // equipped
		nil,             // inventory
		[]string{"git"}, // desired: only git stays
	)

	want := []move{{File: "docker.profile.toml", From: Equipped, To: Inventory}}
	if !reflect.DeepEqual(moves, want) {
		t.Errorf("got %+v, want %+v", moves, want)
	}
}

func TestPlanReconcile_NeverStashesCore(t *testing.T) {
	moves := planReconcile(
		[]string{"core.profile.toml", "git.profile.toml"}, // equipped
		nil,
		[]string{"git"}, // desired: git only; core is unlisted but must stay equipped
	)

	// core is protected, git is desired & already equipped → no moves at all.
	if len(moves) != 0 {
		t.Errorf("expected no moves (core protected), got %+v", moves)
	}
}

func TestReconcile_AppliesMovesOnDisk(t *testing.T) {
	cfg := t.TempDir()
	inv := paths.InventoryDir(cfg)
	if err := os.MkdirAll(inv, 0o755); err != nil {
		t.Fatal(err)
	}
	// Start: docker equipped, git in inventory.
	mustWrite(t, filepath.Join(cfg, "docker.profile.toml"))
	mustWrite(t, filepath.Join(inv, "git.profile.toml"))

	res, err := Reconcile(cfg, []string{"git"})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	if _, err := os.Stat(filepath.Join(cfg, "git.profile.toml")); err != nil {
		t.Errorf("git should be equipped: %v", err)
	}
	if _, err := os.Stat(filepath.Join(inv, "docker.profile.toml")); err != nil {
		t.Errorf("docker should be stashed: %v", err)
	}

	if !reflect.DeepEqual(res.Equipped, []string{"git"}) {
		t.Errorf("Equipped = %v, want [git]", res.Equipped)
	}
	if !reflect.DeepEqual(res.Stashed, []string{"docker"}) {
		t.Errorf("Stashed = %v, want [docker]", res.Stashed)
	}
}

func mustWrite(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
