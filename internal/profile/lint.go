package profile

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/lucky7xz/drako/internal/glyph"
)

// drako check: lint a profile file for authoring mistakes the loader
// deliberately tolerates at runtime. The loader stays permissive so a typo
// never bricks a deck mid-session; this is the strict gate for authors.
//
// Checks work on the RAW decoded TOML, not on Command structs — decoding a
// Command resolves platform variants for the machine running the check, which
// would hide exactly the class of bug (a misspelled variant key) this exists
// to catch.

// Finding is one problem found in a profile file.
type Finding struct {
	Level string // "error" or "warning"
	Msg   string
}

func errorf(format string, args ...any) Finding {
	return Finding{Level: "error", Msg: fmt.Sprintf(format, args...)}
}

func warnf(format string, args ...any) Finding {
	return Finding{Level: "warning", Msg: fmt.Sprintf(format, args...)}
}

// DistroKeywords says what each Linux variant key means: the /etc/os-release
// ids (matched by prefix) that select it. To add a distro, append its id to
// the right list or add an entry; the key then joins the vocabulary below and
// host detects it.
var DistroKeywords = map[string][]string{
	"linux_arch":   {"arch", "manjaro", "endeavouros", "cachy", "garuda", "omarchy"},
	"linux_fedora": {"fedora", "rhel", "centos", "nobara"},
	"linux_debian": {"debian", "ubuntu", "pop", "mint", "kali", "mx", "zorin", "elementary", "neon"},
	"linux_suse":   {"suse", "opensuse", "sles"},
	"linux_void":   {"void"},
	"linux_nixos":  {"nixos"},
}

// VariantKeys is the closed vocabulary for command variant tables, sorted:
// every distro key, plus the generic fallbacks. linux_immutable is listed by
// hand — host probes it from the filesystem, not from /etc/os-release.
func VariantKeys() []string {
	keys := []string{"linux_generic", "linux_immutable", "macos", "windows"}
	for k := range DistroKeywords {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// loaderFindings reports what the loader says about the deck: what it refuses
// (errors), and the flaws it loads the deck with (Flaws). Check asks the
// loader's own rules, so the two cannot disagree.
func loaderFindings(src []byte) []Finding {
	pf, problems, err := Decode(src, nil) // the loader's verdict does not depend on the platform
	if err != nil {
		return []Finding{errorf("%v", err)}
	}
	var fs []Finding
	for _, p := range problems {
		fs = append(fs, errorf("%s", p))
	}
	return append(fs, Flaws(pf)...)
}

// Lint checks raw profile TOML for authoring mistakes. It never touches the
// filesystem and is independent of the platform it runs on.
func Lint(src []byte) []Finding {
	var raw map[string]any
	if _, err := toml.Decode(string(src), &raw); err != nil {
		var pe toml.ParseError
		if errors.As(err, &pe) {
			return []Finding{errorf("TOML parse error at line %d: %s", pe.Position.Line, pe.Message)}
		}
		return []Finding{errorf("TOML parse error: %v", err)}
	}

	var fs []Finding

	cmds := rawTables(rawKey(raw, "commands"))
	if len(cmds) == 0 {
		return append(fs, errorf("no [[commands]] found"))
	}
	fs = append(fs, loaderFindings(src)...)
	fs = append(fs, checkDrawn("profile", raw, "header_art")...)

	for i, cmd := range cmds {
		name := strings.TrimSpace(tomlStr(cmd, "name"))
		label := fmt.Sprintf("command %q", name)
		if name == "" {
			label = fmt.Sprintf("command #%d", i+1) // Flaws says the name is missing
		}
		fs = append(fs, checkFields(label, cmd, cellFields)...)
		fs = append(fs, checkDrawn(label, cmd, "name", "description")...)

		items := rawTables(cmd["items"])
		fs = append(fs, checkCommandValue(label, cmd["command"], len(items) > 0)...)
		fs = append(fs, checkDanger(label, cmd)...)

		if len(items) > 9 {
			fs = append(fs, warnf("%s has %d items — the digit keys reach only the first 9; the rest are picked with the cursor", label, len(items)))
		}
		for j, item := range items {
			iname := strings.TrimSpace(tomlStr(item, "name"))
			ilabel := fmt.Sprintf("%s, item %q", label, iname)
			if iname == "" {
				ilabel = fmt.Sprintf("%s, item #%d", label, j+1)
			}
			fs = append(fs, checkFields(ilabel, item, itemFields)...)
			fs = append(fs, checkDrawn(ilabel, item, "name", "description")...)
			fs = append(fs, checkCommandValue(ilabel, item["command"], false)...)
			fs = append(fs, checkDanger(ilabel, item)...)
		}
	}
	return fs
}

// cellFields and itemFields are the keys the loader reads, with the type each
// must have; "" marks a key checked on its own (command, items, danger).
var (
	cellFields = map[string]string{
		"name": "a string", "description": "a string", "col": "a string", "row": "an integer",
		"auto_close_execution": "a boolean", "debug_execution": "a boolean",
		"command": "", "items": "", "danger": "",
	}
	itemFields = map[string]string{
		"name": "a string", "description": "a string",
		"auto_close_execution": "a boolean", "debug_execution": "a boolean",
		"command": "", "danger": "",
	}
)

// checkFields flags keys the loader ignores and values it would read as unset.
func checkFields(label string, m map[string]any, fields map[string]string) []Finding {
	var fs []Finding
	for _, k := range slices.Sorted(maps.Keys(m)) {
		want, known := fields[k]
		switch got := kindOf(m[k]); {
		case !known:
			fs = append(fs, warnf("%s: unrecognized key %q is ignored (check for a typo)", label, k))
		case want != "" && got != want:
			fs = append(fs, errorf("%s: %s must be %s, got %s", label, k, want, got))
		}
	}
	return fs
}

// checkDrawn warns about labels drako draws differently from how they are
// written: joiners and variation selectors, which terminals draw at different
// widths (glyph.Clean), and in a name, which is one line, line breaks and
// tabs (glyph.Label). Commands are not labels; they run as written.
func checkDrawn(label string, m map[string]any, keys ...string) []Finding {
	var fs []Finding
	for _, k := range keys {
		s, ok := m[k].(string)
		if !ok || glyph.Check(s) != nil {
			continue // refused by the loader, reported there
		}
		if drawn := glyph.Clean(s); drawn != s {
			fs = append(fs, warnf("%s: %s holds characters terminals draw at different widths (joiners, variation selectors); drako shows it as %q", label, k, drawn))
		}
		if k == "name" && strings.ContainsAny(s, "\n\r\t") {
			fs = append(fs, warnf("%s: name holds a line break or tab; a name is one line, so drako shows it as %q", label, glyph.Label(s)))
		}
	}
	return fs
}

func kindOf(v any) string {
	switch v.(type) {
	case string:
		return "a string"
	case int64:
		return "an integer"
	case bool:
		return "a boolean"
	case float64:
		return "a float"
	case map[string]any, []any, []map[string]any:
		return "a table or array"
	}
	return fmt.Sprintf("%T", v)
}

// checkDanger rejects a non-boolean danger: decoding treats it as true, so a
// danger = "false" typo silently guards the cell.
func checkDanger(label string, m map[string]any) []Finding {
	v, ok := m["danger"]
	if !ok {
		return nil
	}
	if _, isBool := v.(bool); !isBool {
		return []Finding{errorf("%s: danger must be true or false, got %v", label, v)}
	}
	return nil
}

// checkCommandValue lints one raw `command` value: a string, a variant
// table with only known platform keys, or absent-but-has-items.
func checkCommandValue(label string, v any, hasItems bool) []Finding {
	switch t := v.(type) {
	case nil:
		if !hasItems {
			return []Finding{errorf("%s has neither a command nor items", label)}
		}
		return nil
	case string:
		if strings.TrimSpace(t) == "" && !hasItems {
			return []Finding{errorf("%s has an empty command and no items", label)}
		}
		return nil
	case map[string]any:
		if len(t) == 0 {
			return []Finding{errorf("%s has an empty variant table", label)}
		}
		var fs []Finding
		known := VariantKeys()
		for k, raw := range t {
			if !slices.Contains(known, k) {
				fs = append(fs, errorf("%s: unknown variant key %q (known: %s)", label, k, strings.Join(known, ", ")))
			}
			if _, ok := raw.(string); !ok {
				fs = append(fs, errorf("%s: variant %q must be a string, got %T", label, k, raw))
			}
		}
		return fs
	default:
		return []Finding{errorf("%s: command must be a string or a table of platform variants, got %T", label, v)}
	}
}

// rawTables normalizes the two shapes TOML table arrays decode to in a
// generic map: inline arrays give []any, [[header]] syntax []map[string]any.
func rawTables(v any) []map[string]any {
	switch t := v.(type) {
	case []map[string]any:
		return t
	case []any:
		out := make([]map[string]any, 0, len(t))
		for _, e := range t {
			if m, ok := e.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}

// rawKey reads a top-level key the way the loader matches it: exact, else
// ignoring case (decks write both `x` and `X`).
func rawKey(m map[string]any, key string) any {
	if v, ok := m[key]; ok {
		return v
	}
	for k, v := range m {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return nil
}
