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
// and returns a PathModel rooted there. "alpha" is nested two deep so a
// descent has somewhere to go; "beta" and "gamma" stay leaves. The process
// working directory is restored after the test so os.Chdir in confirm paths
// can't leak.
func pathTestModel(t *testing.T) *PathModel {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"alpha/one/deep", "alpha/two", "beta", "gamma", ".hidden"} {
		if err := os.MkdirAll(filepath.Join(dir, filepath.FromSlash(name)), 0o755); err != nil {
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
		mode := pm.UpdatePathMode(keyRune("q"), cfg)
		if mode != gridMode {
			t.Errorf("mode = %v, want gridMode", mode)
		}
	})

	t.Run("down enters child mode when dirs exist", func(t *testing.T) {
		pm := pathTestModel(t)
		mode := pm.UpdatePathMode(tea.KeyMsg{Type: tea.KeyDown}, cfg)
		if mode != childMode {
			t.Errorf("mode = %v, want childMode", mode)
		}
		if pm.SelectedChildIndex != 0 {
			t.Errorf("SelectedChildIndex = %d, want 0", pm.SelectedChildIndex)
		}
	})

	t.Run("breadcrumb confirm jumps up and stays in path mode", func(t *testing.T) {
		pm := pathTestModel(t)
		wantBase := filepath.Base(filepath.Dir(pm.CurrentPath))

		pm.UpdatePathMode(tea.KeyMsg{Type: tea.KeyLeft}, cfg) // select the parent crumb
		mode := pm.UpdatePathMode(tea.KeyMsg{Type: tea.KeyEnter}, cfg)
		if mode != pathMode {
			t.Errorf("mode = %v, want pathMode", mode)
		}
		cwd, _ := os.Getwd()
		if filepath.Base(cwd) != wantBase {
			t.Errorf("cwd base = %q, want %q", filepath.Base(cwd), wantBase)
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
		mode := pm.UpdatePathMode(keyRune("e"), cfg)
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

		mode := pm.UpdateChildMode(tea.KeyMsg{Type: tea.KeyDown}, cfg)
		if mode != childMode || pm.SelectedChildIndex != 1 {
			t.Fatalf("after down: mode=%v idx=%d, want childMode 1", mode, pm.SelectedChildIndex)
		}

		// up from index 0 returns to path mode
		pm.SelectedChildIndex = 0
		mode = pm.UpdateChildMode(tea.KeyMsg{Type: tea.KeyUp}, cfg)
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

	t.Run("confirm chdirs into selected child and stays on the list", func(t *testing.T) {
		pm := pathTestModel(t)
		pm.UpdatePathMode(tea.KeyMsg{Type: tea.KeyDown}, cfg) // alpha
		mode := pm.UpdateChildMode(tea.KeyMsg{Type: tea.KeyEnter}, cfg)
		if mode != childMode {
			t.Errorf("mode = %v, want childMode", mode)
		}
		cwd, _ := os.Getwd()
		if filepath.Base(cwd) != "alpha" {
			t.Errorf("cwd base = %q, want alpha", filepath.Base(cwd))
		}
		want := []string{"one", "two"}
		if len(pm.ChildDirs) != 2 || pm.ChildDirs[0] != want[0] || pm.ChildDirs[1] != want[1] {
			t.Errorf("ChildDirs = %v, want %v", pm.ChildDirs, want)
		}
		if pm.SelectedChildIndex != 0 {
			t.Errorf("SelectedChildIndex = %d, want 0", pm.SelectedChildIndex)
		}
	})

	t.Run("confirm into a leaf lands on the breadcrumb", func(t *testing.T) {
		pm := pathTestModel(t)
		pm.UpdatePathMode(tea.KeyMsg{Type: tea.KeyDown}, cfg)  // alpha
		pm.UpdateChildMode(tea.KeyMsg{Type: tea.KeyDown}, cfg) // beta, a leaf
		mode := pm.UpdateChildMode(tea.KeyMsg{Type: tea.KeyEnter}, cfg)
		if mode != pathMode {
			t.Errorf("mode = %v, want pathMode", mode)
		}
		cwd, _ := os.Getwd()
		if filepath.Base(cwd) != "beta" {
			t.Errorf("cwd base = %q, want beta", filepath.Base(cwd))
		}
	})

	// The point of the whole change: descending is Enter, Enter, Enter with
	// no detour through the grid.
	t.Run("repeated confirms descend without leaving path navigation", func(t *testing.T) {
		pm := pathTestModel(t)
		pm.UpdatePathMode(tea.KeyMsg{Type: tea.KeyDown}, cfg) // alpha

		for _, want := range []string{"alpha", "one", "deep"} {
			if mode := pm.UpdateChildMode(tea.KeyMsg{Type: tea.KeyEnter}, cfg); mode == gridMode {
				t.Fatalf("descending into %s dropped to gridMode", want)
			}
			cwd, _ := os.Getwd()
			if filepath.Base(cwd) != want {
				t.Fatalf("cwd base = %q, want %q", filepath.Base(cwd), want)
			}
		}
	})

	t.Run("confirm with an emptied listing does not panic", func(t *testing.T) {
		pm := pathTestModel(t)
		pm.UpdatePathMode(tea.KeyMsg{Type: tea.KeyDown}, cfg)
		pm.UpdateChildMode(keyRune("e"), cfg)
		for _, r := range []string{"z", "z", "z"} { // matches nothing
			pm.UpdateChildMode(keyRune(r), cfg)
		}
		if len(pm.ChildDirs) != 0 {
			t.Fatalf("ChildDirs = %v, want empty", pm.ChildDirs)
		}
		if mode := pm.UpdateChildMode(tea.KeyMsg{Type: tea.KeyEnter}, cfg); mode != pathMode {
			t.Errorf("mode = %v, want pathMode", mode)
		}
	})

	t.Run("entering a directory clears the search filter", func(t *testing.T) {
		pm := pathTestModel(t)
		pm.UpdatePathMode(tea.KeyMsg{Type: tea.KeyDown}, cfg)
		pm.UpdateChildMode(keyRune("e"), cfg)
		pm.UpdateChildMode(keyRune("a"), cfg)
		pm.UpdateChildMode(keyRune("l"), cfg) // "al" -> only alpha
		pm.UpdateChildMode(tea.KeyMsg{Type: tea.KeyEnter}, cfg)

		if pm.Searching || pm.Filter != "" {
			t.Errorf("Searching = %v, Filter = %q, want false and empty", pm.Searching, pm.Filter)
		}
		// "one" and "two" survive only if the stale "al" filter was dropped.
		if len(pm.ChildDirs) != 2 {
			t.Errorf("ChildDirs = %v, want [one two]", pm.ChildDirs)
		}
	})

	t.Run("q exits to grid mode", func(t *testing.T) {
		pm := pathTestModel(t)
		pm.UpdatePathMode(tea.KeyMsg{Type: tea.KeyDown}, cfg)
		mode := pm.UpdateChildMode(keyRune("q"), cfg)
		if mode != gridMode {
			t.Errorf("mode = %v, want gridMode", mode)
		}
	})
}

// The path search is the TUI's other typed-text field, and it had the same
// problem as the inventory prompt: bindings checked in Update, above the mode
// dispatch, ate characters before the filter could see them. Directories are
// named freely, so every letter has to reach the buffer.
func TestPathSearch_TypesKeysBoundElsewhere(t *testing.T) {
	cases := []struct {
		name string
		mode navMode
		keys []string
	}{
		// 'r' is the lock key, checked globally for every mode.
		{name: "path mode", mode: pathMode, keys: []string{"r"}},
		// Child mode is additionally inside the profile-cycle gate, so it
		// lost 'o' and 'p' on top of 'r'.
		{name: "child mode", mode: childMode, keys: []string{"r", "o", "p"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, key := range tc.keys {
				pm := pathTestModel(t)
				pm.Searching = true

				m := Model{
					mode:    tc.mode,
					path:    *pm,
					gridNav: gridNav{grid: [][]string{{"A"}}},
					Config: config.Config{
						Keys: config.InputConfig{Lock: "r", ProfilePrev: "o", ProfileNext: "p"},
					},
				}

				next, _ := m.Update(keyRune(key))
				got := next.(Model)

				if got.path.Filter != key {
					t.Errorf("%s: filter = %q, want %q", key, got.path.Filter, key)
				}
			}
		})
	}
}
