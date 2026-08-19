package config

import (
	"errors"
	"fmt"
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

	cmds := rawTables(raw["commands"])
	if len(cmds) == 0 {
		return append(fs, errorf("no [[commands]] found"))
	}

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
			fs = append(fs, checkCommandValue(ilabel, item["command"], false)...)
		}
	}
	return fs
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

// rawInt reads an int key case-insensitively — generic decode preserves the
// author's casing (decks write both `x` and `X`).
func rawInt(m map[string]any, key string) int {
	for k, v := range m {
		if strings.EqualFold(k, key) {
			if n, ok := v.(int64); ok {
				return int(n)
			}
		}
	}
	return 0
}
