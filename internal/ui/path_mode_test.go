package ui

import (
	"os"
	"path/filepath"
	"strings"
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
			PathSearch:   "e",
			ToggleHidden: ".",
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

// collapseHome and splitPath take the separator as a parameter, so the Windows
// cases below run on Linux too — which is the point, since the old inline
// version hardcoded "/" and its "~" branch could never fire on Windows.
func TestCollapseHome(t *testing.T) {
	const unix, win = '/', '\\'
	tests := []struct {
		name       string
		path, home string
		sep        rune
		want       string
	}{
		{"exactly home", "/home/lucky", "/home/lucky", unix, "~"},
		{"under home", "/home/lucky/shara/drako", "/home/lucky", unix, "~/shara/drako"},
		{"outside home", "/etc/hosts", "/home/lucky", unix, "/etc/hosts"},
		{"prefix but not a child", "/home/lucky2/x", "/home/lucky", unix, "/home/lucky2/x"},
		{"unresolvable home", "/home/lucky/x", "", unix, "/home/lucky/x"},
		{"root", "/", "/home/lucky", unix, "/"},
		{"windows exactly home", `C:\Users\lucky`, `C:\Users\lucky`, win, "~"},
		{"windows under home", `C:\Users\lucky\dev`, `C:\Users\lucky`, win, `~\dev`},
		{"windows outside home", `C:\Windows`, `C:\Users\lucky`, win, `C:\Windows`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := collapseHome(tt.path, tt.home, tt.sep); got != tt.want {
				t.Errorf("collapseHome(%q, %q) = %q, want %q", tt.path, tt.home, got, tt.want)
			}
		})
	}
}

func TestSplitPath(t *testing.T) {
	const unix, win = '/', '\\'
	tests := []struct {
		name string
		path string
		sep  rune
		want []string
	}{
		{"root only", "/", unix, []string{"/"}},
		{"absolute", "/home/lucky", unix, []string{"/", "home", "lucky"}},
		{"tilde only", "~", unix, []string{"~"}},
		{"under tilde", "~/shara/drako", unix, []string{"~", "shara", "drako"}},
		{"windows drive root", `C:\`, win, []string{"C:"}},
		{"windows path", `C:\Users\lucky`, win, []string{"C:", "Users", "lucky"}},
		{"windows tilde", `~\dev`, win, []string{"~", "dev"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitPath(tt.path, tt.sep)
			if len(got) != len(tt.want) {
				t.Fatalf("splitPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("splitPath(%q)[%d] = %q, want %q", tt.path, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// A refused chdir used to be silent: the key did nothing and nothing said why.
func TestEnterDir_RefusalIsVisible(t *testing.T) {
	cfg := navKeys()

	t.Run("failed chdir names the directory and stays put", func(t *testing.T) {
		pm := pathTestModel(t)
		before, _ := os.Getwd()
		pm.UpdatePathMode(tea.KeyMsg{Type: tea.KeyDown}, cfg) // alpha

		// Removing it is the portable way to make chdir fail; 0000 perms would
		// still let root through.
		if err := os.RemoveAll(filepath.Join(pm.CurrentPath, "alpha")); err != nil {
			t.Fatal(err)
		}

		mode := pm.UpdateChildMode(tea.KeyMsg{Type: tea.KeyEnter}, cfg)
		if mode != childMode {
			t.Errorf("mode = %v, want childMode", mode)
		}
		if pm.DeniedDir != "alpha" {
			t.Errorf("DeniedDir = %q, want alpha", pm.DeniedDir)
		}
		if cwd, _ := os.Getwd(); cwd != before {
			t.Errorf("cwd = %q, want it unchanged at %q", cwd, before)
		}
	})

	t.Run("the refusal renders under the listing", func(t *testing.T) {
		pm := pathTestModel(t)
		pm.DeniedDir = "root"
		out := pm.RenderChildDirs(childMode, BuildStyles(config.Config{}), 80)
		if !strings.Contains(out, "cannot enter root") {
			t.Errorf("RenderChildDirs missing the refusal. Got:\n%s", out)
		}
		if !strings.Contains(out, "alpha") {
			t.Errorf("RenderChildDirs dropped the listing. Got:\n%s", out)
		}
	})

	t.Run("the next keypress dismisses it", func(t *testing.T) {
		pm := pathTestModel(t)
		pm.DeniedDir = "root"
		pm.UpdatePathMode(tea.KeyMsg{Type: tea.KeyDown}, cfg)
		if pm.DeniedDir != "" {
			t.Errorf("DeniedDir = %q, want it cleared", pm.DeniedDir)
		}
	})
}

// Both keys were hardcoded, so rebinding edit_file (which also defaults to "e")
// left the path filter stranded on "e".
func TestPathKeys_AreConfigurable(t *testing.T) {
	cfg := navKeys()
	cfg.Keys.PathSearch = "/"
	cfg.Keys.ToggleHidden = "H"

	t.Run("rebound search key opens the filter", func(t *testing.T) {
		pm := pathTestModel(t)
		pm.UpdatePathMode(keyRune("/"), cfg)
		if !pm.Searching {
			t.Error("Searching = false, want true")
		}
	})

	t.Run("the old default no longer searches", func(t *testing.T) {
		pm := pathTestModel(t)
		pm.UpdatePathMode(keyRune("e"), cfg)
		if pm.Searching {
			t.Error("Searching = true, want false — 'e' was rebound away")
		}
	})

	t.Run("rebound hidden toggle works", func(t *testing.T) {
		pm := pathTestModel(t)
		pm.UpdatePathMode(keyRune("H"), cfg)
		if !pm.ShowHidden {
			t.Error("ShowHidden = false, want true")
		}
	})

	t.Run("the old hidden default no longer toggles", func(t *testing.T) {
		pm := pathTestModel(t)
		pm.UpdatePathMode(keyRune("."), cfg)
		if pm.ShowHidden {
			t.Error("ShowHidden = true, want false — '.' was rebound away")
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

// pathWindow is the fix for a long cwd dragging the whole frame out of
// alignment: lipgloss.JoinVertical rectangularizes to the widest sibling, and
// lipgloss.Place refuses to pad content wider than the box, so an unbounded
// path bar silently un-centers the UI.
func TestPathWindow(t *testing.T) {
	// Four components of width 10, separators 3, ellipsis 3.
	widths := []int{10, 10, 10, 10}
	const sep, ell = 3, 3

	tests := []struct {
		name       string
		cursor     int
		budget     int
		start, end int
	}{
		// 4*10 + 3*3 = 49, no ellipsis needed.
		{"everything fits", 3, 49, 0, 4},
		// span(1,4) = 6 sep + 30 + 6 ellipsis = 42, so three components and a
		// left "…" still fit; only the root drops.
		{"one column short of fitting all", 3, 48, 1, 4},
		// Cursor at the deep end keeps the deep end.
		{"tight budget keeps the cursor", 3, 20, 3, 4},
		{"tight budget at the root", 0, 20, 0, 1},
		// Growing right needs span(1,3) = 35 > 30. Growing left instead drops
		// the left "…" entirely and lands at 29, so it wins.
		{"cursor in the middle", 1, 30, 0, 2},
		// Even an impossible budget still shows the selected component; the
		// caller truncates as the backstop.
		{"budget below one component", 2, 1, 2, 3},
		{"zero budget", 0, 0, 0, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := pathWindow(widths, sep, ell, tt.cursor, tt.budget)
			if start != tt.start || end != tt.end {
				t.Errorf("pathWindow(cursor=%d, budget=%d) = (%d,%d), want (%d,%d)",
					tt.cursor, tt.budget, start, end, tt.start, tt.end)
			}
			if tt.cursor >= start && tt.cursor < end {
				return
			}
			t.Errorf("window (%d,%d) excludes the cursor %d", start, end, tt.cursor)
		})
	}
}

func TestPathWindow_EmptyAndSingle(t *testing.T) {
	if s, e := pathWindow(nil, 3, 3, 0, 40); s != 0 || e != 0 {
		t.Errorf("empty = (%d,%d), want (0,0)", s, e)
	}
	if s, e := pathWindow([]int{5}, 3, 3, 0, 40); s != 0 || e != 1 {
		t.Errorf("single = (%d,%d), want (0,1)", s, e)
	}
}
