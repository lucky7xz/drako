package settings

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/lucky7xz/drako/internal/profile"
)

// What drako check says about [keys]. It reads the key table (keymap.go), the
// one the TUI acts on, so a warning here is what the TUI will do. Nothing
// here changes a binding; these are warnings for the person editing
// config.toml.

// KeyFindings checks the bindings of c, which must have its defaults applied.
func KeyFindings(c Config) []profile.Finding {
	var fs []profile.Finding
	warn := func(format string, args ...any) {
		fs = append(fs, profile.Finding{Level: "warning", Msg: fmt.Sprintf(format, args...)})
	}
	screens := c.Screens()

	// A setting that is not a key press can never be reached.
	keys := map[string]string{}
	for _, s := range screens {
		for _, b := range s.Binds {
			if b.Setting != "" {
				keys[b.Setting] = b.Keys[0]
			}
		}
	}
	names := make([]string, 0, len(keys))
	for name := range keys {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		if why := unknownKey(keys[name]); why != "" {
			warn("keys.%s = %q: %s, so this action cannot be reached", name, keys[name], why)
		}
	}

	// Two binds on one key: the screen gives it to the first, the other never
	// acts there. One warning per pair, naming every screen where they meet.
	type clash struct{ key, winner, loser string }
	var order []clash
	found := map[clash]*struct {
		does    string // what drako uses the key for, when one side is drako's own
		screens []string
	}{}
	for _, s := range screens {
		for _, b := range s.Binds {
			for _, key := range b.Keys {
				first, _ := s.Find(key)
				if first.Act == b.Act && first.Setting == b.Setting || first.Setting == "" && b.Setting == "" ||
					unknownKey(key) != "" {
					continue
				}
				cl := clash{key, first.Setting, b.Setting}
				f := found[cl]
				if f == nil {
					f = &struct {
						does    string
						screens []string
					}{}
					found[cl] = f
					order = append(order, cl)
					if first.Setting == "" {
						f.does = first.Does
					} else {
						f.does = b.Does
					}
				}
				if !slices.Contains(f.screens, s.Name) {
					f.screens = append(f.screens, s.Name)
				}
			}
		}
	}
	for _, cl := range order {
		f := found[cl]
		where := strings.Join(f.screens, ", ")
		switch {
		case cl.winner == "":
			warn("keys.%s = %q: drako uses %s itself (%s), so keys.%s does nothing on %s",
				cl.loser, cl.key, keyName(cl.key), f.does, cl.loser, where)
		case cl.loser == "":
			warn("keys.%s = %q: drako uses %s itself (%s), which stops working on %s",
				cl.winner, cl.key, keyName(cl.key), f.does, where)
		default:
			warn("keys.%s and keys.%s are both %q: on %s only keys.%s acts",
				cl.winner, cl.loser, cl.key, where, cl.winner)
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
