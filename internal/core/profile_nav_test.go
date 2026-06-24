package core

import "testing"

func TestCalculateNextProfileIndex(t *testing.T) {
	tests := []struct {
		name      string
		current   int
		direction int
		total     int
		expected  int
	}{
		{"Next wrap", 2, 1, 3, 0},
		{"Prev wrap", 0, -1, 3, 2},
		{"Simple next", 0, 1, 3, 1},
		{"Simple prev", 1, -1, 3, 0},
		{"Zero total", 0, 1, 0, 0},
		{"Single item next", 0, 1, 1, 0},
		{"Single item prev", 0, -1, 1, 0},
		{"Large step", 0, 5, 3, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateNextProfileIndex(tt.current, tt.direction, tt.total); got != tt.expected {
				t.Errorf("CalculateNextProfileIndex(%d, %d, %d) = %d, want %d",
					tt.current, tt.direction, tt.total, got, tt.expected)
			}
		})
	}
}
