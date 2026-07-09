package ui

import (
	"testing"

	"github.com/lucky7xz/drako/internal/config"
)

// A big grid must not demand a terminal that fits it whole: scrolling only
// needs a minimum window, so layout math caps the grid's claim at
// MinVisibleGridRows/Cols.

func TestIsBelowMinimum_BigGridScrollsInsteadOfBlocking(t *testing.T) {
	cfg := config.Config{X: 9, Y: 10}
	tooSmall, _, _ := IsBelowMinimum(100, 25, cfg)
	if tooSmall {
		t.Error("9x10 grid in 100x25 terminal should scroll, not hit the size overlay")
	}
}

func TestCalculateLayout_TallGridKeepsHeader(t *testing.T) {
	cfg := config.Config{X: 3, Y: 10}
	l := CalculateLayout(100, 40, cfg)
	if !l.ShowHeader {
		t.Error("tall grid in a 40-line terminal should keep the header and scroll the grid")
	}
	if !l.ShowFooter {
		t.Error("tall grid in a 40-line terminal should keep the footer")
	}
}
