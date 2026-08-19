package profiles

import (
	"fmt"
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

func TestPlanReconcile_CoreIsOrdinary(t *testing.T) {
	moves := planReconcile(
		[]string{"core.profile.toml", "git.profile.toml"}, // equipped
		nil,
		[]string{"git"}, // desired: git only; core is unlisted → stashed like any deck
	)

	want := []move{{File: "core.profile.toml", From: Equipped, To: Inventory}}
	if !reflect.DeepEqual(moves, want) {
		t.Errorf("got %+v, want %+v", moves, want)
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

// profileFiles/profileNames build n synthetic profiles as on-disk filenames
// and as the bare references a spec would list.
func profileFiles(prefix string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%s%d%s", prefix, i, ProfileSuffix)
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

func TestCheckCap(t *testing.T) {
	cases := []struct {
		name                         string
		equipped, inventory, desired []string
		wantErr                      bool
	}{
		{
			name:      "filling up to the cap is fine",
			equipped:  profileFiles("eq", 3),
			inventory: profileFiles("inv", 6),
			desired:   append(profileNames("eq", 3), profileNames("inv", 6)...),
		},
		{
			name:      "growing past the cap is refused",
			equipped:  profileFiles("eq", 3),
			inventory: profileFiles("inv", 9),
			desired:   append(profileNames("eq", 3), profileNames("inv", 9)...),
			wantErr:   true,
		},
		{
			// A config from before the cap, or profiles copied in by hand.
			name:     "an over-cap list may shrink",
			equipped: profileFiles("eq", 15),
			desired:  profileNames("eq", 12),
		},
		{
			name:     "an over-cap list may reorder",
			equipped: profileFiles("eq", 15),
			desired:  profileNames("eq", 15),
		},
		{
			name:      "an over-cap list may not grow",
			equipped:  profileFiles("eq", 15),
			inventory: profileFiles("inv", 1),
			desired:   append(profileNames("eq", 15), profileNames("inv", 1)...),
			wantErr:   true,
		},
		{
			// Why the cap counts the plan's outcome rather than len(desired):
			// a spec may name profiles that exist in neither location.
			name:     "names matching no file don't count",
			equipped: profileFiles("eq", 3),
			desired:  append(profileNames("eq", 3), profileNames("ghost", 9)...),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			moves := planReconcile(tc.equipped, tc.inventory, tc.desired)
			err := checkCap(len(tc.equipped), moves)
			if tc.wantErr && err == nil {
				t.Fatal("expected the cap to refuse this plan")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("cap refused a valid plan: %v", err)
			}
		})
	}
}

func TestReconcile_OverCapMovesNothing(t *testing.T) {
	cfg := t.TempDir()
	inv := paths.InventoryDir(cfg)
	if err := os.MkdirAll(inv, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range profileFiles("inv", MaxEquipped+1) {
		mustWrite(t, filepath.Join(inv, f))
	}

	if _, err := Reconcile(cfg, profileNames("inv", MaxEquipped+1)); err == nil {
		t.Fatal("expected Reconcile to refuse equipping more than the cap")
	}

	// The refusal has to land before any Move: a half-applied plan would
	// leave the config dir in a state the user never asked for.
	equipped, err := List(cfg)
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
	inv := paths.InventoryDir(cfg)
	if err := os.MkdirAll(inv, 0o755); err != nil {
		t.Fatal(err)
	}
	want := MaxEquipped + 3
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

	equipped, err := List(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(equipped) != want {
		t.Errorf("%d profiles on disk, want %d", len(equipped), want)
	}
}

func TestPlannedEquippedCount(t *testing.T) {
	cfg := t.TempDir()
	inv := paths.InventoryDir(cfg)
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
			equipped, err := List(cfg)
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
