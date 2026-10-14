package ui

import (
	"testing"

	"github.com/lucky7xz/drako/internal/config"
)

func gridOfRows(n int) Model {
	cfg := config.Config{X: 2, Y: n}
	cfg.ApplyDefaults()
	grid := make([][]string, n)
	for i := range grid {
		grid[i] = []string{"cell", "cell"}
	}
	return Model{Config: cfg, styles: BuildStyles(cfg), gridNav: gridNav{grid: grid}}
}

// The 10th row is labelled "10", so the size gate must budget a second digit.
func TestBelowMinimumBudgetsTwoDigitRowLabels(t *testing.T) {
	_, w9, _ := gridOfRows(9).belowMinimum()
	_, w10, _ := gridOfRows(10).belowMinimum()
	if w10 != w9+1 {
		t.Errorf("required width: 9 rows = %d, 10 rows = %d; want one more column for the 2-digit label", w9, w10)
	}
}
