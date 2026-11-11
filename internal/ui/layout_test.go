package ui

import (
	"strings"
	"testing"

	"github.com/lucky7xz/drako/internal/settings"
)

// A big grid must not demand a terminal that fits it whole: scrolling only
// needs a minimum window, so layout math caps the grid's claim at
// MinVisibleGridRows/Cols and the size gate at a 2x2 window.

func TestBelowMinimum_BigGridScrollsInsteadOfBlocking(t *testing.T) {
	m := makeScrollTestModel(9, 10, 100, 25)
	if tooSmall, _, _ := m.belowMinimum(); tooSmall {
		t.Error("9x10 grid in 100x25 terminal should scroll, not hit the size overlay")
	}
}

func TestCalculateLayout_TallGridKeepsHeader(t *testing.T) {
	cfg := settings.Config{X: 3, Y: 10}
	l := CalculateLayout(100, 40, cfg, "")
	if !l.ShowHeader {
		t.Error("tall grid in a 40-line terminal should keep the header and scroll the grid")
	}
	if !l.ShowFooter {
		t.Error("tall grid in a 40-line terminal should keep the footer")
	}
}

func TestCalculateLayout_NarrowTerminalHidesHeaderNotFooter(t *testing.T) {
	cfg := settings.Config{X: 3, Y: 10}
	// termH is generous (won't trigger the height-based hide on its own);
	// termW is narrow enough that the grid-width estimate should hide the
	// header before cells would need to compress.
	l := CalculateLayout(50, 40, cfg, "")
	if l.ShowHeader {
		t.Error("narrow terminal should hide the header even when tall enough")
	}
	if !l.ShowFooter {
		t.Error("width-based hiding must never take the footer down with it")
	}
}

func TestCalculateLayout_WidthJustAboveThresholdKeepsHeader(t *testing.T) {
	cfg := settings.Config{X: 5, Y: 1}
	// neededWidth here is min(5, MinVisibleGridCols)*comfortableCellFootprint
	// + LayoutSideMargin = 3*29+4 = 91 (MinVisibleGridCols caps the grid
	// term regardless of X); 95 sits just above that boundary.
	l := CalculateLayout(95, 40, cfg, "")
	if !l.ShowHeader {
		t.Error("terminal width just above the estimated threshold should keep the header")
	}
}

func TestCalculateLayout_WideCustomHeaderArtHidesEarlier(t *testing.T) {
	cfg := settings.Config{X: 1, Y: 1}
	wideArt := strings.Repeat("X", 60)
	// The grid alone (1x1) would tolerate a much narrower terminal, but a
	// wide custom header_art must still force the hide before it would be
	// the widest element the grid gets centered against.
	l := CalculateLayout(50, 40, cfg, wideArt)
	if l.ShowHeader {
		t.Error("a header_art wider than the terminal must hide, even for a tiny grid")
	}
}

// With no help line on screen, only the grid and the header art decide the
// width at which the header hides: a tiny grid keeps it on a narrow terminal.
func TestCalculateLayout_OnlyGridAndArtHideTheHeaderByWidth(t *testing.T) {
	cfg := settings.Config{X: 1, Y: 1} // needs 1*29 + 4 = 33 columns
	if l := CalculateLayout(40, 40, cfg, strings.Repeat("X", 30)); !l.ShowHeader {
		t.Error("a 1x1 grid and 30-column art fit in 40 columns; the header should stay")
	}
}
