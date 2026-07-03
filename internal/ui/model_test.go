package ui

import "testing"

func TestLockStatePump(t *testing.T) {
	t.Run("alternating directions fill to the goal and unlock", func(t *testing.T) {
		l := lockState{pumpGoal: 3}
		dirs := []int{1, -1, 1}
		for i, d := range dirs {
			unlocked := l.pump(d)
			want := i == len(dirs)-1
			if unlocked != want {
				t.Errorf("pump %d: unlocked = %v, want %v (progress %d)", i, unlocked, want, l.progress)
			}
		}
	})

	t.Run("repeating a direction drains progress", func(t *testing.T) {
		l := lockState{pumpGoal: 6}
		l.pump(1)
		l.pump(-1) // progress 2
		if l.pump(-1) {
			t.Error("repeat direction must not unlock")
		}
		if l.progress != 1 {
			t.Errorf("progress = %d, want 1 after drain", l.progress)
		}
	})

	t.Run("progress never goes negative or past the goal", func(t *testing.T) {
		l := lockState{pumpGoal: 2}
		l.pump(1)
		l.pump(1)
		l.pump(1) // repeated drains: 1, 0, then floor
		if l.progress < 0 {
			t.Errorf("progress = %d, went negative", l.progress)
		}
		l = lockState{pumpGoal: 2, progress: 2, lastDirection: -1}
		l.pump(1)
		if l.progress > l.pumpGoal {
			t.Errorf("progress = %d, exceeded goal %d", l.progress, l.pumpGoal)
		}
	})
}

func TestGridNavClampCursor(t *testing.T) {
	grid := [][]string{
		{"A", "B"},
		{"C", "D"},
	}
	tests := []struct {
		name             string
		g                gridNav
		wantRow, wantCol int
	}{
		{"inside bounds is untouched", gridNav{grid: grid, cursorRow: 1, cursorCol: 0}, 1, 0},
		{"overflow clamps to last cell", gridNav{grid: grid, cursorRow: 9, cursorCol: 9}, 1, 1},
		{"negative clamps to zero", gridNav{grid: grid, cursorRow: -2, cursorCol: -2}, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.g.clampCursor()
			if tt.g.cursorRow != tt.wantRow || tt.g.cursorCol != tt.wantCol {
				t.Errorf("cursor = (%d,%d), want (%d,%d)", tt.g.cursorRow, tt.g.cursorCol, tt.wantRow, tt.wantCol)
			}
		})
	}

	t.Run("empty grid does not panic", func(t *testing.T) {
		g := gridNav{cursorRow: 5, cursorCol: 5}
		g.clampCursor() // must not panic; cursor left as-is is acceptable
	})
}
