package core

import "testing"

func TestPumpDirectionForKey(t *testing.T) {
	tests := []struct {
		key      string
		expected int
	}{
		{"left", DirectionLeft},
		{"right", DirectionRight},
		{"h", DirectionLeft},
		{"l", DirectionRight},
		{"a", DirectionLeft},
		{"d", DirectionRight},
		{"x", DirectionNone},
		{"", DirectionNone},
	}

	for _, tt := range tests {
		if got := PumpDirectionForKey(tt.key); got != tt.expected {
			t.Errorf("PumpDirectionForKey(%q) = %v, want %v", tt.key, got, tt.expected)
		}
	}
}
