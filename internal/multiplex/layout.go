package multiplex

import "slices"

// PanesPerTab caps how many cells share one tab. Past four, tiled panes stop
// being readable — so a batch spills into more tabs rather than smaller panes.
const PanesPerTab = 4

// MinTabs is the fewest tabs n cells can occupy.
func MinTabs(n int) int {
	if n <= 0 {
		return 0
	}
	return (n + PanesPerTab - 1) / PanesPerTab
}

// Distribute lays n cells into k tabs. At the fewest possible tabs it fills
// left to right, so a launch opens with a full first tab; above that it spreads
// evenly, remainder to the left, so dialing tabs up gives readable groups
// instead of one full tab and a tail of singletons. k is clamped to the range
// the cells actually allow.
func Distribute(n, k int) []int {
	if n <= 0 {
		return nil
	}
	k = min(max(k, MinTabs(n)), n)

	tabs := make([]int, k)
	if k == MinTabs(n) {
		for i, left := 0, n; left > 0; i, left = i+1, left-min(left, PanesPerTab) {
			tabs[i] = min(left, PanesPerTab)
		}
		return tabs
	}
	for i := range tabs {
		tabs[i] = n / k
		if i < n%k {
			tabs[i]++
		}
	}
	return tabs
}

// Shift moves one pane into (delta +1) or out of (delta -1) tab i, taking it
// from — or giving it to — the nearest tab that can absorb the change, looking
// right first and then left. The tab count and the pane total never change.
// ok is false when the move would empty or overfill a tab, or when no tab can
// compensate; the input is never modified.
func Shift(tabs []int, i, delta int) (out []int, ok bool) {
	if i < 0 || i >= len(tabs) || (delta != 1 && delta != -1) {
		return tabs, false
	}
	if want := tabs[i] + delta; want < 1 || want > PanesPerTab {
		return tabs, false
	}
	j := compensator(tabs, i, -delta)
	if j < 0 {
		return tabs, false
	}
	out = slices.Clone(tabs)
	out[i] += delta
	out[j] -= delta
	return out, true
}

// compensator is the nearest tab to i — right first, then left — that can take
// delta without emptying or overfilling.
func compensator(tabs []int, i, delta int) int {
	fits := func(j int) bool {
		v := tabs[j] + delta
		return v >= 1 && v <= PanesPerTab
	}
	for j := i + 1; j < len(tabs); j++ {
		if fits(j) {
			return j
		}
	}
	for j := i - 1; j >= 0; j-- {
		if fits(j) {
			return j
		}
	}
	return -1
}
