package multiplex

import (
	"slices"
	"testing"
)

func TestMinTabs(t *testing.T) {
	want := map[int]int{1: 1, 2: 1, 3: 1, 4: 1, 5: 2, 6: 2, 7: 2, 8: 2, 9: 3}
	for n, k := range want {
		if got := MinTabs(n); got != k {
			t.Errorf("MinTabs(%d) = %d, want %d", n, got, k)
		}
	}
	if got := MinTabs(0); got != 0 {
		t.Errorf("MinTabs(0) = %d, want 0", got)
	}
}

// At the fewest possible tabs, cells fill left to right — a full first tab is
// the default a launch opens with.
func TestDistribute_GreedyAtMinimum(t *testing.T) {
	cases := []struct {
		n, k int
		want []int
	}{
		{1, 1, []int{1}},
		{4, 1, []int{4}},
		{5, 2, []int{4, 1}},
		{6, 2, []int{4, 2}},
		{9, 3, []int{4, 4, 1}},
	}
	for _, c := range cases {
		if got := Distribute(c.n, c.k); !slices.Equal(got, c.want) {
			t.Errorf("Distribute(%d, %d) = %v, want %v", c.n, c.k, got, c.want)
		}
	}
}

// Above the minimum the knob spreads cells evenly instead, so dialing tabs up
// gives readable groups rather than one full tab and a tail of singletons.
func TestDistribute_BalancedAboveMinimum(t *testing.T) {
	cases := []struct {
		n, k int
		want []int
	}{
		{6, 3, []int{2, 2, 2}},
		{6, 4, []int{2, 2, 1, 1}},
		{6, 6, []int{1, 1, 1, 1, 1, 1}},
		{9, 4, []int{3, 2, 2, 2}},
		{9, 5, []int{2, 2, 2, 2, 1}},
		{9, 9, []int{1, 1, 1, 1, 1, 1, 1, 1, 1}},
	}
	for _, c := range cases {
		if got := Distribute(c.n, c.k); !slices.Equal(got, c.want) {
			t.Errorf("Distribute(%d, %d) = %v, want %v", c.n, c.k, got, c.want)
		}
	}
}

// Every layout the dialog can reach must be launchable: no empty tab, no tab
// over the ceiling, and every marked cell placed exactly once.
func TestDistribute_InvariantsAcrossTheWholeRange(t *testing.T) {
	for n := 1; n <= MaxCommands; n++ {
		for k := MinTabs(n); k <= n; k++ {
			got := Distribute(n, k)
			if len(got) != k {
				t.Errorf("Distribute(%d, %d) = %v, want %d tabs", n, k, got, k)
				continue
			}
			total := 0
			for _, panes := range got {
				if panes < 1 || panes > PanesPerTab {
					t.Errorf("Distribute(%d, %d) = %v has a tab outside 1..%d", n, k, got, PanesPerTab)
				}
				total += panes
			}
			if total != n {
				t.Errorf("Distribute(%d, %d) = %v places %d cells, want %d", n, k, got, total, n)
			}
		}
	}
}

func TestDistribute_ClampsImpossibleTabCounts(t *testing.T) {
	// Nine cells cannot fit in one tab, nor spread over ten.
	if got, want := Distribute(9, 1), Distribute(9, MinTabs(9)); !slices.Equal(got, want) {
		t.Errorf("Distribute(9, 1) = %v, want the minimum layout %v", got, want)
	}
	if got, want := Distribute(9, 20), Distribute(9, 9); !slices.Equal(got, want) {
		t.Errorf("Distribute(9, 20) = %v, want the one-per-tab layout %v", got, want)
	}
	if got := Distribute(0, 1); got != nil {
		t.Errorf("Distribute(0, 1) = %v, want nil", got)
	}
}

func TestShift_MovesOnePaneToTheNeighbour(t *testing.T) {
	got, ok := Shift([]int{4, 2}, 0, -1)
	if !ok || !slices.Equal(got, []int{3, 3}) {
		t.Errorf("down on tab 0 = %v (ok=%v), want [3 3]", got, ok)
	}
	got, ok = Shift([]int{3, 3}, 0, +1)
	if !ok || !slices.Equal(got, []int{4, 2}) {
		t.Errorf("up on tab 0 = %v (ok=%v), want [4 2]", got, ok)
	}
}

// The immediate neighbour may be at the ceiling; the pane carries on to the
// next tab that has room rather than the move being refused.
func TestShift_CascadesPastAFullTab(t *testing.T) {
	got, ok := Shift([]int{4, 4, 1}, 0, -1)
	if !ok || !slices.Equal(got, []int{3, 4, 2}) {
		t.Errorf("down on tab 0 = %v (ok=%v), want [3 4 2]", got, ok)
	}
}

// On the last tab there is nothing to the right, so the search turns back —
// past tab 1, which cannot spare its only pane, to tab 0 which can.
func TestShift_SearchesLeftAndSkipsATabThatCannotSpare(t *testing.T) {
	got, ok := Shift([]int{4, 1, 1}, 2, +1)
	if !ok || !slices.Equal(got, []int{3, 1, 2}) {
		t.Errorf("up on the last tab = %v (ok=%v), want [3 1 2]", got, ok)
	}
}

func TestShift_RefusesRatherThanBreakingTheLayout(t *testing.T) {
	cases := []struct {
		name string
		tabs []int
		i, d int
	}{
		{"emptying the focused tab", []int{1, 4}, 0, -1},
		{"overfilling the focused tab", []int{4, 2}, 0, +1},
		{"no donor that can spare a pane", []int{1, 1, 1}, 1, +1},
		{"no tab with room to take one", []int{4, 4}, 0, -1},
		{"index out of range", []int{2, 2}, 5, +1},
		{"a delta that is not one pane", []int{2, 2}, 0, +2},
	}
	for _, c := range cases {
		before := slices.Clone(c.tabs)
		got, ok := Shift(c.tabs, c.i, c.d)
		if ok {
			t.Errorf("%s: Shift(%v, %d, %d) = %v, want refusal", c.name, before, c.i, c.d, got)
		}
		if !slices.Equal(c.tabs, before) {
			t.Errorf("%s: a refused shift must not touch the input, got %v", c.name, c.tabs)
		}
	}
}

func TestShift_PreservesTabCountAndTotal(t *testing.T) {
	for _, tabs := range [][]int{{4, 2}, {4, 4, 1}, {2, 2, 2}, {1, 2, 3, 3}} {
		for i := range tabs {
			for _, d := range []int{-1, +1} {
				got, ok := Shift(tabs, i, d)
				if !ok {
					continue
				}
				if len(got) != len(tabs) {
					t.Errorf("Shift(%v, %d, %d) changed the tab count: %v", tabs, i, d, got)
				}
				sum := func(v []int) int {
					out := 0
					for _, x := range v {
						out += x
					}
					return out
				}
				if sum(got) != sum(tabs) {
					t.Errorf("Shift(%v, %d, %d) = %v changed the pane total", tabs, i, d, got)
				}
				if got[i] != tabs[i]+d {
					t.Errorf("Shift(%v, %d, %d) = %v did not move the focused tab", tabs, i, d, got)
				}
			}
		}
	}
}

func TestShift_DoesNotAliasTheInput(t *testing.T) {
	tabs := []int{4, 2}
	got, ok := Shift(tabs, 0, -1)
	if !ok {
		t.Fatal("expected the shift to succeed")
	}
	got[0] = 99
	if tabs[0] != 4 {
		t.Errorf("Shift must return a fresh slice; input became %v", tabs)
	}
}

// Not every layout can be fine-tuned. A single tab has no neighbour, and two
// full tabs have nowhere to move a pane to. The tab count is the only way out
// of those, which is why the dialog keeps that knob as well.
func TestShift_RigidLayoutsRefuseEveryMove(t *testing.T) {
	for _, tabs := range [][]int{{1}, {4}, {4, 4}} {
		for i := range tabs {
			for _, d := range []int{-1, +1} {
				if got, ok := Shift(tabs, i, d); ok {
					t.Errorf("Shift(%v, %d, %d) = %v, want refusal", tabs, i, d, got)
				}
			}
		}
	}
	// Eight cells escape [4 4] by asking for a third tab, not by shifting.
	if got := Distribute(8, 3); !slices.Equal(got, []int{3, 3, 2}) {
		t.Errorf("Distribute(8, 3) = %v, want [3 3 2]", got)
	}
}
