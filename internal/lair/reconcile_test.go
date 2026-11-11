package lair

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/lucky7xz/drako/internal/equip"
)

func TestReconcile_AppliesMovesOnDisk(t *testing.T) {
	cfg := t.TempDir()
	inv := InventoryDir(cfg)
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
		t.Errorf("equip.Equipped = %v, want [git]", res.Equipped)
	}
	if !reflect.DeepEqual(res.Stashed, []string{"docker"}) {
		t.Errorf("Stashed = %v, want [docker]", res.Stashed)
	}
}

// Stashes run before equips, so a move failing partway never leaves more decks
// equipped than the plan was allowed to end with.
func TestReconcile_FailedStashNeverExceedsTheCap(t *testing.T) {
	cfg := t.TempDir()
	inv := InventoryDir(cfg)
	for _, f := range profileFiles("eq", equip.MaxEquipped) {
		mustWrite(t, filepath.Join(cfg, f))
	}
	// A directory where eq0 would land makes its stash fail.
	if err := os.MkdirAll(filepath.Join(inv, "eq0"+equip.ProfileSuffix), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(inv, "new.profile.toml"))

	desired := append(profileNames("eq", equip.MaxEquipped)[1:], "new")
	if _, err := Reconcile(cfg, desired); err == nil {
		t.Fatal("Reconcile should report the failed stash")
	}
	if n, _ := List(cfg, equip.Equipped); len(n) > equip.MaxEquipped {
		t.Errorf("%d decks equipped after the failure, cap is %d", len(n), equip.MaxEquipped)
	}
}

func TestReconcile_OverCapMovesNothing(t *testing.T) {
	cfg := t.TempDir()
	inv := InventoryDir(cfg)
	if err := os.MkdirAll(inv, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range profileFiles("inv", equip.MaxEquipped+1) {
		mustWrite(t, filepath.Join(inv, f))
	}

	if _, err := Reconcile(cfg, profileNames("inv", equip.MaxEquipped+1)); err == nil {
		t.Fatal("expected Reconcile to refuse equipping more than the cap")
	}

	// The refusal has to land before any Move: a half-applied plan would
	// leave the config dir in a state the user never asked for.
	equipped, err := List(cfg, equip.Equipped)
	if err != nil {
		t.Fatal(err)
	}
	if len(equipped) != 0 {
		t.Errorf("refused Reconcile moved %d files onto disk", len(equipped))
	}
}

// The spec CLI equips over the cap once the user has confirmed it.
func TestReconcileOverCap_EquipsPastTheCap(t *testing.T) {
	cfg := t.TempDir()
	inv := InventoryDir(cfg)
	if err := os.MkdirAll(inv, 0o755); err != nil {
		t.Fatal(err)
	}
	want := equip.MaxEquipped + 3
	for _, f := range profileFiles("inv", want) {
		mustWrite(t, filepath.Join(inv, f))
	}

	res, err := ReconcileOverCap(cfg, profileNames("inv", want))
	if err != nil {
		t.Fatalf("ReconcileOverCap: %v", err)
	}
	if len(res.Equipped) != want {
		t.Errorf("equipped %d profiles, want %d", len(res.Equipped), want)
	}

	equipped, err := List(cfg, equip.Equipped)
	if err != nil {
		t.Fatal(err)
	}
	if len(equipped) != want {
		t.Errorf("%d profiles on disk, want %d", len(equipped), want)
	}
}

func TestPlannedEquippedCount(t *testing.T) {
	cfg := t.TempDir()
	inv := InventoryDir(cfg)
	if err := os.MkdirAll(inv, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range profileFiles("eq", 2) {
		mustWrite(t, filepath.Join(cfg, f))
	}
	for _, f := range profileFiles("inv", 4) {
		mustWrite(t, filepath.Join(inv, f))
	}

	cases := []struct {
		name    string
		desired []string
		want    int
	}{
		{name: "equipping everything", desired: append(profileNames("eq", 2), profileNames("inv", 4)...), want: 6},
		{name: "stashing everything", desired: nil, want: 0},
		{name: "keeping what is equipped", desired: profileNames("eq", 2), want: 2},
		{name: "names matching no file are ignored", desired: append(profileNames("eq", 2), profileNames("ghost", 5)...), want: 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := PlannedEquippedCount(cfg, tc.desired)
			if err != nil {
				t.Fatalf("PlannedEquippedCount: %v", err)
			}
			if got != tc.want {
				t.Errorf("count = %d, want %d", got, tc.want)
			}
			// Preview only: nothing may reach disk.
			equipped, err := List(cfg, equip.Equipped)
			if err != nil {
				t.Fatal(err)
			}
			if len(equipped) != 2 {
				t.Errorf("PlannedEquippedCount moved files: %d equipped, want 2", len(equipped))
			}
		})
	}
}

func mustWrite(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// profileFiles/profileNames build n synthetic profiles as on-disk filenames
// and as the bare references a spec would list.
func profileFiles(prefix string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%s%d%s", prefix, i, equip.ProfileSuffix)
	}
	return out
}

func profileNames(prefix string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%s%d", prefix, i)
	}
	return out
}
