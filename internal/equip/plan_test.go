package equip

import (
	"fmt"
	"reflect"
	"testing"
)

func TestPlanReconcile_EquipsDesiredFromInventory(t *testing.T) {
	moves := PlanReconcile(
		nil,                          // equipped: none
		[]string{"git.profile.toml"}, // inventory
		[]string{"git"},              // desired
	)

	want := []Move{{File: "git.profile.toml", From: Inventory, To: Equipped}}
	if !reflect.DeepEqual(moves, want) {
		t.Errorf("got %+v, want %+v", moves, want)
	}
}

func TestPlanReconcile_StashesEquippedNotDesired(t *testing.T) {
	moves := PlanReconcile(
		[]string{"git.profile.toml", "docker.profile.toml"}, // equipped
		nil,             // inventory
		[]string{"git"}, // desired: only git stays
	)

	want := []Move{{File: "docker.profile.toml", From: Equipped, To: Inventory}}
	if !reflect.DeepEqual(moves, want) {
		t.Errorf("got %+v, want %+v", moves, want)
	}
}

func TestPlanReconcile_CoreIsOrdinary(t *testing.T) {
	moves := PlanReconcile(
		[]string{"core.profile.toml", "git.profile.toml"}, // equipped
		nil,
		[]string{"git"}, // desired: git only; core is unlisted → stashed like any deck
	)

	want := []Move{{File: "core.profile.toml", From: Equipped, To: Inventory}}
	if !reflect.DeepEqual(moves, want) {
		t.Errorf("got %+v, want %+v", moves, want)
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
			moves := PlanReconcile(tc.equipped, tc.inventory, tc.desired)
			err := CheckCap(len(tc.equipped), moves)
			if tc.wantErr && err == nil {
				t.Fatal("expected the cap to refuse this plan")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("cap refused a valid plan: %v", err)
			}
		})
	}
}

func TestPlace(t *testing.T) {
	entry := func(file string) Entry {
		return Entry{File: file, Norm: NormalizeName(file)}
	}
	full := make([]Entry, MaxEquipped)
	for i := range full {
		full[i] = entry(fmt.Sprintf("e%d.profile.toml", i))
	}
	cases := []struct {
		name              string
		equipped, stashed []Entry
		file              string
		wantLoc           Location
		wantName          string
	}{
		{"new, room left: equipped", []Entry{entry("a.profile.toml")}, nil, "b.profile.toml", Equipped, "b.profile.toml"},
		{"new, nine equipped: inventory", full, nil, "b.profile.toml", Inventory, "b.profile.toml"},
		{"already equipped: replaced in place, even when full", full, nil, "e3.profile.toml", Equipped, "e3.profile.toml"},
		{"already stashed: replaced in the inventory", nil, []Entry{entry("b.profile.toml")}, "b.profile.toml", Inventory, "b.profile.toml"},
		{"same deck, other spelling: the existing name is kept", []Entry{entry("Work.profile.toml")}, nil, "work.profile.toml", Equipped, "Work.profile.toml"},
		{"equipped wins over stashed", []Entry{entry("b.profile.toml")}, []Entry{entry("B.profile.toml")}, "b.profile.toml", Equipped, "b.profile.toml"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			loc, name := Place(tc.equipped, tc.stashed, tc.file)
			if loc != tc.wantLoc || name != tc.wantName {
				t.Errorf("Place = %v, %q; want %v, %q", loc, name, tc.wantLoc, tc.wantName)
			}
		})
	}
}
