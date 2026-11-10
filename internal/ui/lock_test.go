package ui

import "testing"

func TestPumpDirectionForKey(t *testing.T) {
	tests := []struct {
		key      string
		expected int
	}{
		{"left", directionLeft},
		{"right", directionRight},
		{"h", directionLeft},
		{"l", directionRight},
		{"a", directionLeft},
		{"d", directionRight},
		{"x", directionNone},
		{"", directionNone},
	}

	for _, tt := range tests {
		if got := pumpDirectionForKey(tt.key); got != tt.expected {
			t.Errorf("pumpDirectionForKey(%q) = %v, want %v", tt.key, got, tt.expected)
		}
	}
}
