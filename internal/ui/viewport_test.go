package ui

import "testing"

func TestWindow(t *testing.T) {
	tests := []struct {
		name                   string
		cursor, total, visible int
		want                   gridWindow
	}{
		{"all fits is identity", 2, 5, 5, gridWindow{0, 5, 0, 0}},
		{"visible larger than total is identity", 0, 3, 10, gridWindow{0, 3, 0, 0}},
		{"cursor at start clamps to zero", 0, 10, 3, gridWindow{0, 3, 0, 7}},
		{"cursor centered mid-grid", 5, 10, 3, gridWindow{4, 7, 4, 3}},
		{"cursor at end clamps to tail", 9, 10, 3, gridWindow{7, 10, 7, 0}},
		{"even visible leans up", 5, 10, 4, gridWindow{3, 7, 3, 3}},
		{"zero visible treated as one", 5, 10, 0, gridWindow{5, 6, 5, 4}},
		{"cursor below range clamps", -3, 10, 3, gridWindow{0, 3, 0, 7}},
		{"cursor past range clamps", 42, 10, 3, gridWindow{7, 10, 7, 0}},
		{"empty grid", 0, 0, 3, gridWindow{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := window(tt.cursor, tt.total, tt.visible)
			if got != tt.want {
				t.Errorf("window(%d, %d, %d) = %+v, want %+v",
					tt.cursor, tt.total, tt.visible, got, tt.want)
			}
		})
	}
}

func TestWindowScrolling(t *testing.T) {
	if window(0, 5, 5).scrolling() {
		t.Error("full window should not report scrolling")
	}
	if !window(5, 10, 3).scrolling() {
		t.Error("clipped window should report scrolling")
	}
}

func TestVisibleCount(t *testing.T) {
	tests := []struct {
		name                             string
		budget, cellSize, total, reserve int
		want                             int
	}{
		{"exact fit returns total, no reserve", 9, 3, 3, 2, 3},
		{"roomy budget returns total", 100, 3, 5, 2, 5},
		{"one line short drops into scroll mode", 8, 3, 3, 2, 2},
		{"reserve shrinks scroll window", 12, 3, 10, 2, 3},
		{"cell wider than budget still shows one", 12, 29, 9, 0, 1},
		{"tiny budget still shows one", 0, 3, 10, 2, 1},
		{"negative budget still shows one", -4, 3, 10, 2, 1},
		{"zero cell size is identity", 10, 0, 7, 2, 7},
		{"zero total is identity", 10, 3, 0, 2, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := visibleCount(tt.budget, tt.cellSize, tt.total, tt.reserve)
			if got != tt.want {
				t.Errorf("visibleCount(%d, %d, %d, %d) = %d, want %d",
					tt.budget, tt.cellSize, tt.total, tt.reserve, got, tt.want)
			}
		})
	}
}

func TestCellFootprint(t *testing.T) {
	tests := []struct {
		name string
		grid [][]string
		want int
	}{
		{"empty grid", nil, 4},
		{"narrow cells", [][]string{{"ab", "abcd"}}, 8},
		{"width capped at GridMaxTextWidth", [][]string{{"this cell content is far too long to fit"}}, GridMaxTextWidth + 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cellFootprint(tt.grid); got != tt.want {
				t.Errorf("cellFootprint = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFitFootprint(t *testing.T) {
	tests := []struct {
		name                            string
		content, widthBudget, totalCols int
		want                            int
	}{
		{"roomy terminal: content wins", 29, 100, 3, 29},
		{"tight: space share wins, marker reserved", 29, 34, 3, 16},
		{"tight two columns: no marker reserve", 29, 32, 2, 16},
		{"never compressed below legibility floor", 29, 20, 3, 12},
		{"narrow content ignores the floor", 8, 20, 3, 8},
		{"single column gets the whole budget", 29, 20, 1, 20},
		{"empty grid guard", 4, 20, 0, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fitFootprint(tt.content, tt.widthBudget, tt.totalCols)
			if got != tt.want {
				t.Errorf("fitFootprint(%d, %d, %d) = %d, want %d",
					tt.content, tt.widthBudget, tt.totalCols, got, tt.want)
			}
		})
	}
}

func TestGridRowBudget(t *testing.T) {
	// appStyle has Margin(1,2): 2 vertical lines always spent.
	tests := []struct {
		name       string
		termHeight int
		chrome     []string
		want       int
	}{
		{"no chrome", 24, nil, 22},
		{"single lines", 24, []string{"a", "b"}, 20},
		{"empty string still costs one line", 24, []string{""}, 21},
		{"multiline chrome measured", 24, []string{"a\nb\nc"}, 19},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gridRowBudget(tt.termHeight, tt.chrome...)
			if got != tt.want {
				t.Errorf("gridRowBudget(%d, %v) = %d, want %d",
					tt.termHeight, tt.chrome, got, tt.want)
			}
		})
	}
}
