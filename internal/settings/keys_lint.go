package settings

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/lucky7xz/drako/internal/profile"
)

// What drako check says about [keys]. The TUI takes a binding as written and
// the first action that asks for a key gets it; nothing here changes that.
// These are warnings for the person editing config.toml.

// keyScreens lists, per screen, the settings whose keys are live there. Two
// of them on one key cannot both act.
var keyScreens = []struct {
	screen  string
	actions []string
}{
	{"the grid", []string{"help", "leader", "explain", "inventory", "path_grid_mode", "lock", "profile_prev", "profile_next"}},
	{"a folder", []string{"help", "leader", "explain"}},
	{"the inventory", []string{"help", "path_grid_mode", "edit_file", "delete"}},
	{"path mode", []string{"help", "path_grid_mode", "path_search", "toggle_hidden", "lock", "profile_prev", "profile_next"}},
}

// bindings is every single-key setting by its name in config.toml.
func (c InputConfig) bindings() map[string]string {
	return map[string]string{
		"explain": c.Explain, "inventory": c.Inventory, "path_grid_mode": c.PathGridMode,
		"lock": c.Lock, "profile_prev": c.ProfilePrev, "profile_next": c.ProfileNext,
		"edit_file": c.EditFile, "delete": c.Delete, "path_search": c.PathSearch,
		"toggle_hidden": c.ToggleHidden, "leader": c.Leader, "help": c.Help,
	}
}

// fixedKeys are the keys drako uses itself on every screen, with what for.
func (c InputConfig) fixedKeys() map[string]string {
	fixed := map[string]string{
		"ctrl+c": "quit", "q": "quit or go back", "esc": "go back",
		"enter": "run or confirm", " ": "run, confirm or mark",
		"pgup": "scroll", "pgdown": "scroll",
	}
	for _, set := range [][]string{c.NavUp, c.NavDown, c.NavLeft, c.NavRight} {
		for _, k := range set {
			fixed[k] = "move the cursor"
		}
	}
	for d := '1'; d <= '9'; d++ {
		fixed[string(d)] = "quick navigation"
	}
	return fixed
}

// KeyFindings checks the bindings of c, which must have its defaults applied.
func KeyFindings(c Config) []profile.Finding {
	var fs []profile.Finding
	warn := func(format string, args ...any) {
		fs = append(fs, profile.Finding{Level: "warning", Msg: fmt.Sprintf(format, args...)})
	}
	keys := c.Keys.bindings()
	names := make([]string, 0, len(keys))
	for name := range keys {
		names = append(names, name)
	}
	slices.Sort(names)

	fixed := c.Keys.fixedKeys()
	for _, name := range names {
		key := keys[name]
		if why := unknownKey(key); why != "" {
			warn("keys.%s = %q: %s, so this action cannot be reached", name, key, why)
			continue
		}
		if use, taken := fixed[key]; taken {
			warn("keys.%s = %q: drako uses %s itself (%s); one of the two stops working", name, key, keyName(key), use)
		}
	}

	seen := map[string]bool{}
	for _, s := range keyScreens {
		for i, a := range s.actions {
			for _, b := range s.actions[i+1:] {
				first, second := min(a, b), max(a, b)
				if keys[a] != keys[b] || seen[first+" "+second] {
					continue
				}
				seen[first+" "+second] = true
				warn("keys.%s and keys.%s are both %q: on %s only one of them can act", first, second, keys[a], s.screen)
			}
		}
	}

	if key := c.Keys.SessionLock; utf8.RuneCountInString(key) != 1 {
		warn("keys.session_lock = %q: must be one character, pressed with %s", key, c.NumbModifier)
	}
	if mod := c.NumbModifier; mod != "alt" && mod != "ctrl" {
		warn("numb_modifier = %q: not a modifier drako can receive (use \"alt\"), so %s+1-9 and the session lock cannot be reached", mod, mod)
	}
	return fs
}

func keyName(key string) string {
	if key == " " {
		return "space"
	}
	return key
}

var namedKeys = []string{
	"tab", "enter", "esc", "backspace", "delete", "insert", "home", "end",
	"pgup", "pgdown", "up", "down", "left", "right",
}

// unknownKey says why key is not a key press the terminal can send drako, in
// the spelling drako compares it with, or "" when it is one.
func unknownKey(key string) string {
	switch key {
	case "space":
		return `the space key is written " "`
	case "ctrl+i":
		return `the terminal sends ctrl+i as "tab"`
	case "ctrl+m":
		return `the terminal sends ctrl+m as "enter"`
	case "escape":
		return `the escape key is written "esc"`
	case "pageup", "pagedown":
		return `page keys are written "pgup" and "pgdown"`
	}
	rest := strings.TrimPrefix(key, "alt+")
	if r, size := utf8.DecodeRuneInString(rest); size == len(rest) && size > 0 && unicode.IsPrint(r) {
		return ""
	}
	if c, ok := strings.CutPrefix(rest, "ctrl+"); ok && len(c) == 1 && (c[0] >= 'a' && c[0] <= 'z' || strings.Contains(`@\]^_`, c)) {
		return ""
	}
	base := strings.TrimPrefix(strings.TrimPrefix(rest, "ctrl+"), "shift+")
	if slices.Contains(namedKeys, base) {
		return ""
	}
	if n, ok := strings.CutPrefix(base, "f"); ok && len(n) > 0 && len(n) <= 2 && strings.Trim(n, "0123456789") == "" && n[0] != '0' {
		return ""
	}
	return `it is not a key name (examples: "e", "tab", "ctrl+e", "f2")`
}
