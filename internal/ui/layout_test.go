package ui

import (
	"strings"
	"testing"

	"github.com/lucky7xz/drako/internal/config"
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
	cfg := config.Config{X: 3, Y: 10}
	l := CalculateLayout(100, 40, cfg, "", "")
	if !l.ShowHeader {
		t.Error("tall grid in a 40-line terminal should keep the header and scroll the grid")
	}
	if !l.ShowFooter {
		t.Error("tall grid in a 40-line terminal should keep the footer")
	}
}

func TestCalculateLayout_NarrowTerminalHidesHeaderNotFooter(t *testing.T) {
	cfg := config.Config{X: 3, Y: 10}
	// termH is generous (won't trigger the height-based hide on its own);
	// termW is narrow enough that the grid-width estimate should hide the
	// header before cells would need to compress.
	l := CalculateLayout(50, 40, cfg, "", "")
	if l.ShowHeader {
		t.Error("narrow terminal should hide the header even when tall enough")
	}
	if !l.ShowFooter {
		t.Error("width-based hiding must never take the footer down with it")
	}
}

func TestCalculateLayout_WidthJustAboveThresholdKeepsHeader(t *testing.T) {
	cfg := config.Config{X: 5, Y: 1}
	// neededWidth here is min(5, MinVisibleGridCols)*comfortableCellFootprint
	// + LayoutSideMargin = 3*29+4 = 91 (MinVisibleGridCols caps the grid
	// term regardless of X); 95 sits just above that boundary.
	l := CalculateLayout(95, 40, cfg, "", "")
	if !l.ShowHeader {
		t.Error("terminal width just above the estimated threshold should keep the header")
	}
}

func TestCalculateLayout_WideCustomHeaderArtHidesEarlier(t *testing.T) {
	cfg := config.Config{X: 1, Y: 1}
	wideArt := strings.Repeat("X", 60)
	// The grid alone (1x1) would tolerate a much narrower terminal, but a
	// wide custom header_art must still force the hide before it would be
	// the widest element the grid gets centered against.
	l := CalculateLayout(50, 40, cfg, wideArt, "")
	if l.ShowHeader {
		t.Error("a header_art wider than the terminal must hide, even for a tiny grid")
	}
}

func TestCalculateLayout_HelpTextWidensHeaderHideThreshold(t *testing.T) {
	cfg := config.Config{X: 1, Y: 1}    // tiny grid so grid/headerArt terms don't dominate
	helpText := strings.Repeat("h", 60) // W = 60 + LayoutSideMargin(4) = 64
	l := CalculateLayout(55, 40, cfg, "", helpText)
	if l.ShowHeader {
		t.Error("help text wider than the terminal must hide the header before wrap/shift apply")
	}
}

func TestCalculateLayout_ShiftLeftThreshold(t *testing.T) {
	cfg := config.Config{X: 1, Y: 1}
	helpText := strings.Repeat("h", 60) // W = 64, shift boundary = W-10 = 54
	tests := []struct {
		name  string
		termW int
		want  bool
	}{
		{"5 above shift boundary", 59, false},
		{"exactly at boundary (strict <)", 54, false},
		{"1 below boundary", 53, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := CalculateLayout(tt.termW, 40, cfg, "", helpText)
			if l.ShiftLeft != tt.want {
				t.Errorf("termW=%d: ShiftLeft = %v, want %v", tt.termW, l.ShiftLeft, tt.want)
			}
			if l.ShiftLeft && l.ShowHeader {
				t.Error("ShiftLeft must never be true while ShowHeader is still true")
			}
		})
	}
}

func TestCalculateLayout_EmptyHelpTextNeverShifts(t *testing.T) {
	cfg := config.Config{X: 3, Y: 3}
	l := CalculateLayout(1, 40, cfg, "", "") // extreme narrowness
	if l.ShiftLeft {
		t.Error("empty help text (viewInfoMode's case) must never trigger ShiftLeft")
	}
}

// Regression guard for the "hide always before shift" invariant across a
// sweep of widths, not just the one boundary case.
func TestCalculateLayout_ShiftImpliesHeaderAlreadyHidden(t *testing.T) {
	cfg := config.Config{X: 3, Y: 3}
	helpText := "Grid Mode | Enter: Select, e: Explain, Tab: Path, r: Start-Lock, i: Inventory"
	for termW := 1; termW <= 150; termW++ {
		l := CalculateLayout(termW, 40, cfg, "", helpText)
		if l.ShiftLeft && l.ShowHeader {
			t.Fatalf("termW=%d: ShiftLeft=true but ShowHeader still true", termW)
		}
	}
}
