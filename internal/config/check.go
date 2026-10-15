package config

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
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

// knownVariantKeys is the closed vocabulary for command variant tables:
// every distro key drako can detect, plus the generic fallbacks.
// linux_immutable is listed by hand — it is probed from the filesystem, not
// matched from /etc/os-release, so it has no DistroKeywords entry.
func knownVariantKeys() []string {
	keys := []string{"linux_generic", ImmutableKey, "macos", "windows"}
	for k := range DistroKeywords {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// loaderFindings reports what the profile loader itself rejects, so check never
// passes a deck the TUI would show as broken.
func loaderFindings(src []byte) []Finding {
	_, problems, err := DecodeProfile(src)
	if err != nil {
		return []Finding{errorf("%v", err)}
	}
	var fs []Finding
	for _, p := range problems {
		fs = append(fs, errorf("%s", p))
	}
	return fs
}

// CheckSettings lints raw config.toml: what sends the loader to rescue mode, and
// keys it ignores (a typo, or a setting drako no longer has).
func CheckSettings(src []byte) []Finding {
	var settings AppSettings
	meta, err := toml.Decode(string(src), &settings)
	if err != nil {
		return []Finding{errorf("%v", err)}
	}
	if keys := undecodedKeys(meta); len(keys) > 0 {
		return []Finding{warnf("unrecognized key(s): %s (a typo, or a setting drako no longer has)", strings.Join(keys, ", "))}
	}
	return nil
}

// CheckProfile lints raw profile TOML. It never touches the filesystem and
// is independent of the platform it runs on.
func CheckProfile(src []byte) []Finding {
	var raw map[string]any
	if _, err := toml.Decode(string(src), &raw); err != nil {
		var pe toml.ParseError
		if errors.As(err, &pe) {
			return []Finding{errorf("TOML parse error at line %d: %s", pe.Position.Line, pe.Message)}
		}
		return []Finding{errorf("TOML parse error: %v", err)}
	}

	var fs []Finding
	x, y := rawInt(raw, "x"), rawInt(raw, "y")

	cmds := rawTables(rawKey(raw, "commands"))
	if len(cmds) == 0 {
		return append(fs, errorf("no [[commands]] found"))
	}
	fs = append(fs, loaderFindings(src)...)

	seenNames := map[string]bool{}
	seenCells := map[string]string{} // cell -> first claimant
	for i, cmd := range cmds {
		name := strings.TrimSpace(tomlStr(cmd, "name"))
		label := fmt.Sprintf("command %q", name)
		if name == "" {
			label = fmt.Sprintf("command #%d", i+1)
			fs = append(fs, warnf("%s has no name — its cell renders empty and can never be selected", label))
		} else if seenNames[name] {
			fs = append(fs, errorf("duplicate command name %q — cells are looked up by name, so one of them can never run", name))
		}
		seenNames[name] = true
		fs = append(fs, checkFields(label, cmd, cellFields)...)

		// Cell address: parse the column, resolve 'z'/-1 against the grid
		// when its size is known, then check bounds and collisions.
		colStr := tomlStr(cmd, "col")
		row := tomlInt(cmd, "row")
		col, err := letterToColumn(colStr)
		switch {
		case colStr == "":
			fs = append(fs, errorf("%s has no col", label))
		case err != nil:
			fs = append(fs, errorf("%s: invalid col %q: %v", label, colStr, err))
		case row < -1:
			fs = append(fs, errorf("%s: invalid row %d (rows start at 0; -1 is the last)", label, row))
		default:
			if col == -1 && x > 0 {
				col = x - 1
			}
			if row == -1 && y > 0 {
				row = y - 1
			}
			if x > 0 && col >= x {
				fs = append(fs, errorf("%s sits in column %q but the grid is only %d wide (x = %d)", label, colStr, x, x))
			}
			if y > 0 && row >= y {
				fs = append(fs, errorf("%s sits in row %d but the grid is only %d tall (y = %d)", label, row, y, y))
			}
			cell := fmt.Sprintf("%c%d", 'A'+col, row)
			if col < 0 {
				cell = fmt.Sprintf("z%d", row)
			}
			if first, taken := seenCells[cell]; taken {
				fs = append(fs, errorf("%s and command %q share cell %s — only the last one placed is visible", label, first, cell))
			} else {
				seenCells[cell] = name
			}
		}

		items := rawTables(cmd["items"])
		fs = append(fs, checkCommandValue(label, cmd["command"], len(items) > 0)...)
		fs = append(fs, checkDanger(label, cmd)...)

		if len(items) > 9 {
			fs = append(fs, warnf("%s has %d items — dropdowns support at most 9", label, len(items)))
		}
		for j, item := range items {
			iname := strings.TrimSpace(tomlStr(item, "name"))
			ilabel := fmt.Sprintf("%s, item %q", label, iname)
			if iname == "" {
				ilabel = fmt.Sprintf("%s, item #%d", label, j+1)
				fs = append(fs, warnf("%s has no name", ilabel))
			}
			fs = append(fs, checkFields(ilabel, item, itemFields)...)
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
		known := knownVariantKeys()
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

func rawInt(m map[string]any, key string) int {
	n, _ := rawKey(m, key).(int64)
	return int(n)
}
