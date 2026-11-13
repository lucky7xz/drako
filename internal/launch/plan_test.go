package launch

import (
	"slices"
	"testing"

	"github.com/lucky7xz/drako/internal/profile"
)

func boolPtr(b bool) *bool { return &b }

// planFor plans sel among the test cells, under bash.
func planFor(sel Selection, active string) Plan {
	return For("bash", testCells(), sel, active)
}

func testCells() []profile.Command {
	return []profile.Command{
		{Name: "hello", Command: "echo hi"},
		{
			Name:               "careful",
			Command:            "sudo reboot",
			AutoCloseExecution: boolPtr(false),
			DebugExecution:     boolPtr(true),
		},
		{Name: "empty-cell"},
		{Name: "blank-cell", Command: "   "},
		{
			Name: "folder",
			Items: []profile.CommandItem{
				{Name: "child", Command: "ls -la", AutoCloseExecution: boolPtr(false)},
				{Name: "bare-child"},
			},
		},
	}
}

func TestFor_TopLevelCommand(t *testing.T) {
	plan := planFor(Selection{N: 1}, "work")
	if !plan.Runnable() {
		t.Fatalf("plan = %+v, want something to run", plan)
	}
	if want := []string{"bash", "-lc", "echo hi"}; !slices.Equal(plan.Argv, want) {
		t.Errorf("Argv = %v, want %v", plan.Argv, want)
	}
	if !plan.AutoClose || plan.Debug {
		t.Errorf("defaults: AutoClose=%v Debug=%v, want true/false", plan.AutoClose, plan.Debug)
	}
	if plan.Name != "hello" || plan.Profile != "work" {
		t.Errorf("Name=%q Profile=%q, want hello/work", plan.Name, plan.Profile)
	}
}

func TestFor_FlagOverrides(t *testing.T) {
	plan := planFor(Selection{N: 2}, "")
	if plan.AutoClose || !plan.Debug {
		t.Errorf("overrides: AutoClose=%v Debug=%v, want false/true", plan.AutoClose, plan.Debug)
	}
}

func TestFor_DropdownItem(t *testing.T) {
	plan := planFor(Selection{N: 5, Item: 1}, "")
	if want := []string{"bash", "-lc", "ls -la"}; !slices.Equal(plan.Argv, want) {
		t.Fatalf("Argv = %v, want %v", plan.Argv, want)
	}
	if plan.AutoClose {
		t.Error("item-level AutoClose override not honored")
	}
	if plan.Name != "child" {
		t.Errorf("Name = %q, want the item's name", plan.Name)
	}
}

func TestFor_ConfiguredButEmpty(t *testing.T) {
	for _, sel := range []Selection{{N: 3}, {N: 4}, {N: 5, Item: 2}} {
		if plan := planFor(sel, ""); plan.Runnable() {
			t.Errorf("%+v: plan = %+v, want nothing to run", sel, plan)
		}
	}
}

func TestFor_NotFound(t *testing.T) {
	// A number past the profile's commands, or an item past the folder's end.
	for _, sel := range []Selection{{N: 9}, {N: 5, Item: 9}} {
		plan := planFor(sel, "")
		if plan.Runnable() {
			t.Errorf("%+v: plan = %+v, want nothing to run", sel, plan)
		}
		if plan.Name != "" && sel.Item == 0 {
			t.Errorf("%+v: Name = %q, want none: there is no such cell", sel, plan.Name)
		}
	}
}

// drako run resolves its cell by address and plans it here; a blank command
// must come out as nothing to run, the same as it does for the grid.
func TestShell(t *testing.T) {
	plan := Shell("sh", "cell", "echo hi", "work")
	if want := []string{"sh", "-c", "echo hi"}; !slices.Equal(plan.Argv, want) {
		t.Errorf("Argv = %v, want %v", plan.Argv, want)
	}
	if plan.Name != "cell" || plan.Profile != "work" || !plan.AutoClose || plan.Debug {
		t.Errorf("plan = %+v, want cell/work with a cell's default flags", plan)
	}
	for _, blank := range []string{"", "   ", "\t\n"} {
		if Shell("sh", "cell", blank, "work").Runnable() {
			t.Errorf("command %q must leave nothing to run", blank)
		}
	}
}
