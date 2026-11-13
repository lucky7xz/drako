package settings

import (
	"strings"
	"testing"
)

func keyWarnings(t *testing.T, src string) string {
	t.Helper()
	var msgs []string
	for _, f := range Lint([]byte(src)) {
		if f.Level != "warning" {
			t.Fatalf("key findings are warnings, got %+v", f)
		}
		msgs = append(msgs, f.Msg)
	}
	return strings.Join(msgs, "\n")
}

// The defaults, and the defaults with a nav set turned off, are clean.
func TestKeyFindings_DefaultsAreClean(t *testing.T) {
	for _, src := range []string{"", "[keys]\ndisable_wasd_bindings = true\ndisable_vim_bindings = true\n"} {
		if got := keyWarnings(t, src); got != "" {
			t.Errorf("config %q should have no findings, got:\n%s", src, got)
		}
	}
}

// I-04: the config that left no cell runnable is now named by check.
func TestKeyFindings_BindingsThatCannotWork(t *testing.T) {
	got := keyWarnings(t, `numb_modifier = "banana"
[keys]
explain = "i"
inventory = "i"
leader = "q"
help = "enter"
path_grid_mode = "tabb"
toggle_hidden = "space"
session_lock = "rr"
`)
	for _, want := range []string{
		`keys.explain and keys.inventory are both "i": on the grid`,
		`keys.leader = "q": drako uses q itself`,
		`keys.help = "enter": drako uses enter itself`,
		`keys.path_grid_mode = "tabb": it is not a key name`,
		`keys.toggle_hidden = "space": the space key is written " "`,
		`keys.session_lock = "rr": must be one character`,
		`numb_modifier = "banana"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

// Keys on different screens may share a key: e explains on the grid, edits in
// the inventory and searches in path mode by default.
func TestKeyFindings_ScreensDoNotCollide(t *testing.T) {
	if got := keyWarnings(t, "[keys]\ndelete = \"x\"\nexplain = \"x\"\n"); got != "" {
		t.Errorf("delete (inventory) and explain (grid) never meet, got:\n%s", got)
	}
	// A nav key is fixed only while its set is on.
	if got := keyWarnings(t, "[keys]\nexplain = \"w\"\n"); !strings.Contains(got, "move the cursor") {
		t.Errorf("w moves the cursor by default, got:\n%s", got)
	}
	if got := keyWarnings(t, "[keys]\ndisable_wasd_bindings = true\nexplain = \"w\"\n"); got != "" {
		t.Errorf("with wasd off, w is free, got:\n%s", got)
	}
}

func TestUnknownKey(t *testing.T) {
	for _, ok := range []string{"e", "E", "?", ".", " ", "tab", "delete", "f2", "f12", "ctrl+e", "alt+e", "alt+enter", "shift+tab", "ctrl+up", "é"} {
		if why := unknownKey(ok); why != "" {
			t.Errorf("%q is a key, got: %s", ok, why)
		}
	}
	for _, bad := range []string{"", "tabb", "space", "escape", "ctrl+m", "f0", "f123", "banana", "ctrl+", "ee"} {
		if unknownKey(bad) == "" {
			t.Errorf("%q is not a key drako can receive", bad)
		}
	}
}
