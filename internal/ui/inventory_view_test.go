package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/core"
)

// namesN builds n synthetic profile names ("profile-0".."profile-(n-1)").
func namesN(n int) []string {
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("profile-%d", i)
	}
	return names
}

// makeInventoryTestModel builds a model with n synthetic profile names in
// the inventory list, real styles, and the given terminal size. focusedList
// selects which list (0=equipped, 1=inventory) currently owns the cursor.
func makeInventoryTestModel(n, cursor, focusedList, termW, termH int) Model {
	cfg := config.Config{X: 2, Y: 2, Theme: "default", DefaultShell: "/bin/bash"}
	return Model{
		mode:       inventoryMode,
		termWidth:  termW,
		termHeight: termH,
		Config:     cfg,
		styles:     BuildStyles(cfg),
		gridNav:    gridNav{grid: [][]string{{"a", "b"}, {"c", "d"}}},
		inventory: inventoryModel{
			State:       core.NewInventoryState(nil, namesN(n)),
			cursor:      cursor,
			focusedList: focusedList,
		},
		profile: profileState{profiles: []config.ProfileInfo{{Name: "Core"}}},
	}
}

// makeInventoryTestModelBothLists is like makeInventoryTestModel but seeds
// both the Equipped (equip-0..) and Inventory (profile-0..) lists, for tests
// covering the focus-priority budget split between them.
func makeInventoryTestModelBothLists(equippedN, inventoryN, cursor, focusedList, termW, termH int) Model {
	equippedNames := make([]string, equippedN)
	for i := range equippedNames {
		equippedNames[i] = fmt.Sprintf("equip-%d", i)
	}
	cfg := config.Config{X: 2, Y: 2, Theme: "default", DefaultShell: "/bin/bash"}
	return Model{
		mode:       inventoryMode,
		termWidth:  termW,
		termHeight: termH,
		Config:     cfg,
		styles:     BuildStyles(cfg),
		gridNav:    gridNav{grid: [][]string{{"a", "b"}, {"c", "d"}}},
		inventory: inventoryModel{
			State:       core.NewInventoryState(equippedNames, namesN(inventoryN)),
			cursor:      cursor,
			focusedList: focusedList,
		},
		profile: profileState{profiles: []config.ProfileInfo{{Name: "Core"}}},
	}
}

func TestRenderInventoryGrid_UnlimitedBudgetShowsEverything(t *testing.T) {
	m := makeInventoryTestModel(30, 0, 1, 80, 24)
	out := m.renderInventoryGrid(namesN(30), 1, unlimitedRowBudget)
	for i := 0; i < 30; i++ {
		if !strings.Contains(out, fmt.Sprintf("profile-%d", i)) {
			t.Errorf("unlimited budget should show every profile, missing profile-%d", i)
		}
	}
	if strings.Contains(out, "▾") || strings.Contains(out, "▴") {
		t.Error("unlimited budget should never show scroll markers")
	}
}

func TestRenderInventoryGrid_TightBudgetScrollsAndShowsMarker(t *testing.T) {
	m := makeInventoryTestModel(30, 0, 1, 80, 24)
	// One row of cells is a few lines tall; a budget of 4 lines can only
	// show a couple of rows out of many.
	out := m.renderInventoryGrid(namesN(30), 1, 4)
	if !strings.Contains(out, "▾") {
		t.Error("scrolled-down content should exist below a tight budget, expected ▾ marker")
	}
}

func TestRenderInventoryGrid_FocusedCentersOnCursor(t *testing.T) {
	// Cursor deep in the list; a tight budget should still keep the
	// cursor's own row visible (its cell content present in the output).
	m := makeInventoryTestModel(30, 25, 1, 80, 24)
	out := m.renderInventoryGrid(namesN(30), 1, 4)
	if !strings.Contains(out, "profile-25") {
		t.Errorf("cursor's row must stay visible when centered around it. Got:\n%s", out)
	}
}

func TestRenderInventoryGrid_UnfocusedStartsFromTopNotUnbounded(t *testing.T) {
	// listID=1 but focusedList=0 (unfocused): must still respect the
	// budget (no overflow), anchored at the top rather than cursor-centered.
	m := makeInventoryTestModel(30, 25, 0, 80, 24)
	out := m.renderInventoryGrid(namesN(30), 1, 4)
	if !strings.Contains(out, "profile-0") {
		t.Error("unfocused list should start from the top")
	}
	if strings.Contains(out, "profile-25") {
		t.Error("unfocused list at the top should not show a cursor-centered row deep in the list")
	}
}

func TestViewInventoryMode_ManyProfilesScrollInsteadOfOverflowing(t *testing.T) {
	m := makeInventoryTestModel(50, 40, 1, 80, 24)
	output := m.View()

	// The whole rendered page must fit the terminal height budget (plus the
	// known static appStyle margin overhead — see the width-cascade tests
	// in view_test.go for why that overhead exists).
	lineCount := strings.Count(output, "\n") + 1
	maxAllowed := m.termHeight + appStyle.GetVerticalMargins()
	if lineCount > maxAllowed {
		t.Errorf("inventory page is %d lines, want <= %d (termHeight=%d)", lineCount, maxAllowed, m.termHeight)
	}
	if !strings.Contains(output, "profile-40") {
		t.Error("cursor's profile must stay visible even when the list doesn't fit")
	}
	if !strings.Contains(output, "Equipped Items") || !strings.Contains(output, "Inventory Items") {
		t.Error("section headers must always render, regardless of scrolling")
	}
	if !strings.Contains(output, "[ Apply Changes ]") || !strings.Contains(output, "[ Rescue Mode ]") {
		t.Error("buttons must always render, regardless of scrolling")
	}
}

func TestViewInventoryMode_FewProfilesNoBlankLinePadding(t *testing.T) {
	m := makeInventoryTestModel(2, 0, 1, 80, 40)
	output := m.View()

	// No two consecutive blank lines anywhere in the body: declutter removed
	// the "\n\n" separators between sections.
	if strings.Contains(output, "\n\n") {
		t.Errorf("expected no blank-line gaps between sections. Got:\n%s", output)
	}
}

func TestViewInventoryMode_EquippedScrollsWhenFocusedAndLarge(t *testing.T) {
	m := makeInventoryTestModelBothLists(30, 0, 25, core.ListVisible, 80, 24)
	output := m.View()

	if !strings.Contains(output, "▾") && !strings.Contains(output, "▴") {
		t.Error("a large focused Equipped Items list should show a scroll marker")
	}
	if !strings.Contains(output, "equip-25") {
		t.Errorf("cursor's equipped item must stay visible when Equipped is focused. Got:\n%s", output)
	}
}

func TestViewInventoryMode_EquippedCappedWhenUnfocusedAndLarge(t *testing.T) {
	// Inventory is focused (gets priority); Equipped has plenty of items but
	// isn't focused, so it must be capped rather than pushing Inventory out.
	m := makeInventoryTestModelBothLists(30, 1, 0, core.ListInventory, 80, 24)
	output := m.View()

	if !strings.Contains(output, "▾") {
		t.Error("an unfocused Equipped Items list larger than its cap should show a ▾ marker")
	}
	if strings.Contains(output, "equip-29") {
		t.Error("an unfocused, capped Equipped Items list should not render items deep past its cap")
	}
	if !strings.Contains(output, "Inventory Items") || !strings.Contains(output, "profile-0") {
		t.Error("the focused Inventory list must still render its content")
	}
}

func TestViewInventoryMode_InventoryCappedWhenUnfocusedAndEquippedIsFocused(t *testing.T) {
	// Roles swapped from the case above: proves the split follows focus,
	// not a hardcoded list.
	m := makeInventoryTestModelBothLists(1, 30, 0, core.ListVisible, 80, 24)
	output := m.View()

	if !strings.Contains(output, "▾") {
		t.Error("an unfocused Inventory Items list larger than its cap should show a ▾ marker")
	}
	if strings.Contains(output, "profile-29") {
		t.Error("an unfocused, capped Inventory Items list should not render items deep past its cap")
	}
	if !strings.Contains(output, "Equipped Items") || !strings.Contains(output, "equip-0") {
		t.Error("the focused Equipped list must still render its content")
	}
}
