package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
)

// navKeys is a minimal binding set for driving path/child navigation.
func navKeys() config.Config {
	return config.Config{
		Keys: config.InputConfig{
			NavUp:        []string{"up", "k"},
			NavDown:      []string{"down", "j"},
			NavLeft:      []string{"left", "h"},
			NavRight:     []string{"right", "l"},
			PathGridMode: "tab",
		},
	}
}

// pathTestDir builds a temp dir with three visible subdirs and one hidden,
// and returns a PathModel rooted there. The process working directory is
// restored after the test so os.Chdir in confirm paths can't leak.
func pathTestModel(t *testing.T) *PathModel {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"alpha", "beta", "gamma", ".hidden"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	orig, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(orig) })

	pm := InitPathModel(dir)
	return &pm
}

func keyRune(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestUpdatePathMode_Characterization(t *testing.T) {
	cfg := navKeys()

	t.Run("init lists visible dirs sorted, hidden excluded", func(t *testing.T) {
		pm := pathTestModel(t)
		want := []string{"alpha", "beta", "gamma"}
		if len(pm.ChildDirs) != len(want) {
			t.Fatalf("ChildDirs = %v, want %v", pm.ChildDirs, want)
		}
		for i := range want {
			if pm.ChildDirs[i] != want[i] {
				t.Errorf("ChildDirs[%d] = %q, want %q", i, pm.ChildDirs[i], want[i])
			}
		}
	})

	t.Run("q exits to grid mode", func(t *testing.T) {
		pm := pathTestModel(t)
		mode, _ := pm.UpdatePathMode(keyRune("q"), cfg)
		if mode != gridMode {
			t.Errorf("mode = %v, want gridMode", mode)
		}
	})

	t.Run("down enters child mode when dirs exist", func(t *testing.T) {
		pm := pathTestModel(t)
		mode, _ := pm.UpdatePathMode(tea.KeyMsg{Type: tea.KeyDown}, cfg)
		if mode != childMode {
			t.Errorf("mode = %v, want childMode", mode)
		}
		if pm.SelectedChildIndex != 0 {
			t.Errorf("SelectedChildIndex = %d, want 0", pm.SelectedChildIndex)
		}
	})

	t.Run("dot toggles hidden files into the list", func(t *testing.T) {
		pm := pathTestModel(t)
		pm.UpdatePathMode(keyRune("."), cfg)
		if !pm.ShowHidden {
			t.Error("ShowHidden = false, want true")
		}
		found := false
		for _, d := range pm.ChildDirs {
			if d == ".hidden" {
				found = true
			}
		}
		if !found {
			t.Errorf("ChildDirs = %v, want it to include .hidden", pm.ChildDirs)
		}
	})

	t.Run("e enters search mode", func(t *testing.T) {
		pm := pathTestModel(t)
		mode, _ := pm.UpdatePathMode(keyRune("e"), cfg)
		if !pm.Searching {
			t.Error("Searching = false, want true")
		}
		if mode != pathMode {
			t.Errorf("mode = %v, want pathMode", mode)
		}
	})

	t.Run("typing in search filters the dir list", func(t *testing.T) {
		pm := pathTestModel(t)
		pm.UpdatePathMode(keyRune("e"), cfg)
		pm.UpdatePathMode(keyRune("a"), cfg) // all of alpha/beta/gamma contain "a"
		if len(pm.ChildDirs) != 3 {
			t.Fatalf("after 'a' ChildDirs = %v, want 3", pm.ChildDirs)
		}
		pm.UpdatePathMode(keyRune("l"), cfg) // "al" -> only alpha
		if len(pm.ChildDirs) != 1 || pm.ChildDirs[0] != "alpha" {
			t.Errorf("after 'al' ChildDirs = %v, want [alpha]", pm.ChildDirs)
		}
	})
}

func TestUpdateChildMode_Characterization(t *testing.T) {
	cfg := navKeys()

	t.Run("down then up navigates and clamps", func(t *testing.T) {
		pm := pathTestModel(t)
		// enter child mode
		pm.UpdatePathMode(tea.KeyMsg{Type: tea.KeyDown}, cfg)

		mode, _ := pm.UpdateChildMode(tea.KeyMsg{Type: tea.KeyDown}, cfg)
		if mode != childMode || pm.SelectedChildIndex != 1 {
			t.Fatalf("after down: mode=%v idx=%d, want childMode 1", mode, pm.SelectedChildIndex)
		}

		// up from index 0 returns to path mode
		pm.SelectedChildIndex = 0
		mode, _ = pm.UpdateChildMode(tea.KeyMsg{Type: tea.KeyUp}, cfg)
		if mode != pathMode {
			t.Errorf("up at index 0: mode = %v, want pathMode", mode)
		}
	})

	t.Run("down stops at last index", func(t *testing.T) {
		pm := pathTestModel(t)
		pm.UpdatePathMode(tea.KeyMsg{Type: tea.KeyDown}, cfg)
		for i := 0; i < 10; i++ {
			pm.UpdateChildMode(tea.KeyMsg{Type: tea.KeyDown}, cfg)
		}
		if pm.SelectedChildIndex != len(pm.ChildDirs)-1 {
			t.Errorf("SelectedChildIndex = %d, want %d", pm.SelectedChildIndex, len(pm.ChildDirs)-1)
		}
	})

	t.Run("confirm chdirs into selected child", func(t *testing.T) {
		pm := pathTestModel(t)
		pm.UpdatePathMode(tea.KeyMsg{Type: tea.KeyDown}, cfg)
		mode, cmd := pm.UpdateChildMode(tea.KeyMsg{Type: tea.KeyEnter}, cfg)
		if mode != gridMode {
			t.Errorf("mode = %v, want gridMode", mode)
		}
		if cmd == nil {
			t.Error("expected a pathChangedMsg command, got nil")
		}
		cwd, _ := os.Getwd()
		if filepath.Base(cwd) != "alpha" {
			t.Errorf("cwd base = %q, want alpha", filepath.Base(cwd))
		}
	})

	t.Run("q exits to grid mode", func(t *testing.T) {
		pm := pathTestModel(t)
		pm.UpdatePathMode(tea.KeyMsg{Type: tea.KeyDown}, cfg)
		mode, _ := pm.UpdateChildMode(keyRune("q"), cfg)
		if mode != gridMode {
			t.Errorf("mode = %v, want gridMode", mode)
		}
	})
}
