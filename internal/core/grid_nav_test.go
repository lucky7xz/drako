package core

import "testing"

func TestFindLastPopulatedCol(t *testing.T) {
	tests := []struct {
		name     string
		grid     [][]string
		expected int
	}{
		{"Empty grid", [][]string{}, -1},
		{"Single empty row", [][]string{{}}, -1},
		{"Single item", [][]string{{"a"}}, 0},
		{"Sparse grid", [][]string{{"a", "", "b"}, {"", "c"}}, 2},
		{"Uneven rows", [][]string{{"a", "b"}, {"a"}}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FindLastPopulatedCol(tt.grid); got != tt.expected {
				t.Errorf("FindLastPopulatedCol() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFindLastPopulatedRow(t *testing.T) {
	tests := []struct {
		name     string
		grid     [][]string
		col      int
		expected int
	}{
		{"Empty grid", [][]string{}, 0, -1},
		{"Col out of bounds", [][]string{{"a"}}, -1, -1},
		{"Col valid", [][]string{{"a"}, {"b"}}, 0, 1},
		{"Col valid sparse", [][]string{{"a"}, {""}, {"b"}}, 0, 2},
		{"Col valid partial", [][]string{{"a", "b"}, {"c"}}, 1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FindLastPopulatedRow(tt.grid, tt.col); got != tt.expected {
				t.Errorf("FindLastPopulatedRow() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFindFirstPopulatedRow(t *testing.T) {
	tests := []struct {
		name     string
		grid     [][]string
		col      int
		expected int
	}{
		{"Empty grid", [][]string{}, 0, 0},
		{"Col out of bounds", [][]string{{"a"}}, -1, 0},
		{"First row populated", [][]string{{"a"}, {"b"}}, 0, 0},
		{"Second row populated", [][]string{{""}, {"b"}}, 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FindFirstPopulatedRow(tt.grid, tt.col); got != tt.expected {
				t.Errorf("FindFirstPopulatedRow() = %v, want %v", got, tt.expected)
			}
		})
	}
}
