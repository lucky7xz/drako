package profile

import (
	"fmt"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// Decode decodes a profile file the way the loader does: err is a TOML
// error, problems what validate rejects. targets are the platform's
// variant keys, most specific first; each command becomes the text written
// for them. A caller that only validates passes nil.
func Decode(raw []byte, targets []string) (pf File, problems []string, err error) {
	var written rawProfile
	meta, err := toml.Decode(string(raw), &written)
	if err != nil {
		return File{}, nil, err
	}
	pf = written.resolve(targets)
	_, problems = validate(pf, raw, meta)
	return pf, problems, nil
}

// validate checks a profile has at least one command, x/y in 1-9,
// and no unrecognized top-level keys (meta names the typo, e.g. "ssy").
func validate(pf File, raw []byte, meta toml.MetaData) (bool, []string) {
	var problems []string
	if len(pf.Commands) == 0 {
		problems = append(problems, "needs at least one command")
	}
	problems = append(problems, validateGridDim("x", pf.X, raw)...)
	problems = append(problems, validateGridDim("y", pf.Y, raw)...)
	if unknown := undecodedTopLevelKeys(meta); len(unknown) > 0 {
		problems = append(problems, fmt.Sprintf("unrecognized key(s): %s (check for a typo)", strings.Join(unknown, ", ")))
	}
	return len(problems) == 0, problems
}

// undecodedTopLevelKeys returns top-level keys that matched no field at all
// — a longer path (e.g. "commands.5.foo") means the top level DID match.
func undecodedTopLevelKeys(meta toml.MetaData) []string {
	var keys []string
	for _, k := range meta.Undecoded() {
		if len(k) == 1 {
			keys = append(keys, k[0])
		}
	}
	return keys
}

// validateGridDim distinguishes "key absent" from "present but out of
// range" instead of reporting the zero-default as if it were typed.
func validateGridDim(key string, val int, raw []byte) []string {
	if val >= 1 && val <= 9 {
		return nil
	}
	line := findKeyLine(raw, key)
	if line == 0 {
		return []string{fmt.Sprintf("%s is missing (must be 1-9)", key)}
	}
	return []string{fmt.Sprintf("%s = %d is invalid (must be 1-9) (line %d)", key, val, line)}
}

// findKeyLine returns the 1-based line of a top-level `key =` assignment, matched
// ignoring case like the decoder, or 0.
func findKeyLine(raw []byte, key string) int {
	for i, line := range strings.Split(string(raw), "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "[") {
			return 0 // the first table header ends the top level
		}
		if k, _, ok := strings.Cut(t, "="); ok && strings.EqualFold(strings.TrimSpace(k), key) {
			return i + 1
		}
	}
	return 0
}

// ── Platform-variant commands ────────────────────────────────────────
// A command (on a cell or a dropdown item) is either a plain string or a
// table of platform variants. Decoding keeps it as written; Decode
// then picks the text for the platform it is given:
//
//   command = "echo same everywhere"
//   command = { linux_debian = "apt ...", linux_arch = "pacman ...", macos = "brew ..." }
//
// Keys come from a closed vocabulary (linux_debian, linux_arch, linux_fedora,
// linux_suse, linux_void, linux_nixos, linux_immutable, linux_generic, macos,
// windows). Keys are tried most specific first, then linux_generic: on an
// image-managed host that is linux_immutable, then the base distro, so only
// cells that install something need the immutable variant. No match: the
// command stays empty (the cell renders, Enter reports "no command
// configured") and a note naming the available variants is appended to the
// description.

// commandText is a command as written in a profile: plain text, or one text
// per platform.
type commandText struct {
	plain    string
	variants map[string]string // nil for the plain form
}

// parseCommandText reads a decoded TOML command value.
func parseCommandText(v any) (commandText, error) {
	switch t := v.(type) {
	case nil:
		return commandText{}, nil
	case string:
		return commandText{plain: t}, nil
	case map[string]any:
		variants := make(map[string]string, len(t))
		for k, raw := range t {
			s, ok := raw.(string)
			if !ok {
				return commandText{}, fmt.Errorf("command variant %q must be a string", k)
			}
			variants[k] = s
		}
		return commandText{variants: variants}, nil
	default:
		return commandText{}, fmt.Errorf("command must be a string or a table of platform variants, got %T", v)
	}
}

// pick chooses the text for targets, the platform's variant keys, most
// specific first. A table with no variant for them yields no command and a
// note saying so. With no targets at all nothing is picked and nothing noted:
// enough for callers that only validate.
func (c commandText) pick(targets []string) (cmd string, note string) {
	if c.variants == nil {
		return c.plain, ""
	}
	if len(targets) == 0 {
		return "", ""
	}
	for _, target := range targets {
		if v, ok := c.variants[target]; ok {
			return v, ""
		}
	}
	// The last target is the base platform, so it decides whether the
	// generic Linux fallback applies.
	base := targets[len(targets)-1]
	if v, ok := c.variants["linux_generic"]; ok && strings.HasPrefix(base, "linux_") {
		return v, ""
	}
	keys := make([]string, 0, len(c.variants))
	for k := range c.variants {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// Name the most specific key: that is the one an author should add.
	return "", fmt.Sprintf("[no command variant for this platform (%s); available: %s]",
		targets[0], strings.Join(keys, ", "))
}

// Written is one command text as its author wrote it, for a reader to review
// before trusting a deck. Item is "" for the cell's own command; Platform is
// "" when one text serves every platform.
type Written struct {
	Cell, Item string
	Platform   string
	Text       string
}

// WrittenCommands lists every command text in raw, each platform's variant
// included, in file order: what the deck can run on any machine, not only
// this one. Cells without a command (folders) are left out.
func WrittenCommands(raw []byte) ([]Written, error) {
	var r rawProfile
	if _, err := toml.Decode(string(raw), &r); err != nil {
		return nil, err
	}
	var out []Written
	for _, c := range r.Commands {
		out = append(out, c.text.written(c.cell.Name, "")...)
		for _, ri := range c.items {
			out = append(out, ri.text.written(c.cell.Name, ri.item.Name)...)
		}
	}
	return out, nil
}

// written lists c's texts for the cell (and item) it belongs to, variants
// sorted by platform key.
func (c commandText) written(cell, item string) []Written {
	if c.variants == nil {
		if c.plain == "" {
			return nil
		}
		return []Written{{Cell: cell, Item: item, Text: c.plain}}
	}
	keys := make([]string, 0, len(c.variants))
	for k := range c.variants {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]Written, 0, len(keys))
	for _, k := range keys {
		out = append(out, Written{Cell: cell, Item: item, Platform: k, Text: c.variants[k]})
	}
	return out
}

// appendNote attaches a resolution note to a description.
func appendNote(desc, note string) string {
	if note == "" {
		return desc
	}
	if strings.TrimSpace(desc) == "" {
		return note
	}
	return desc + "\n\n" + note
}

// small typed getters for the primitive map UnmarshalTOML receives
func tomlStr(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}
func tomlInt(m map[string]any, key string) int {
	if n, ok := m[key].(int64); ok {
		return int(n)
	}
	return 0
}

// tomlDanger fails closed: a danger value that isn't a bool counts as true.
func tomlDanger(m map[string]any) bool {
	v, ok := m["danger"]
	if !ok {
		return false
	}
	b, isBool := v.(bool)
	return !isBool || b
}
func tomlBoolPtr(m map[string]any, key string) *bool {
	if b, ok := m[key].(bool); ok {
		return &b
	}
	return nil
}

// rawProfile is a profile as written, before a platform is chosen. Keep its
// fields in sync with File.
type rawProfile struct {
	X         int          `toml:"x"`
	Y         int          `toml:"y"`
	Theme     string       `toml:"theme"`
	HeaderArt *string      `toml:"header_art"`
	Shell     *string      `toml:"shell"`
	Assets    *[]string    `toml:"assets"`
	Commands  []rawCommand `toml:"commands"`
}

// resolve picks every command's text for targets.
func (r rawProfile) resolve(targets []string) File {
	pf := File{X: r.X, Y: r.Y, Theme: r.Theme, HeaderArt: r.HeaderArt, Shell: r.Shell, Assets: r.Assets}
	for _, c := range r.Commands {
		pf.Commands = append(pf.Commands, c.resolve(targets))
	}
	return pf
}

// rawCommand is a cell as written: everything but its command text and its
// items' sits in cell already.
type rawCommand struct {
	cell  Command
	text  commandText
	items []rawItem
}

// rawItem is a dropdown item as written.
type rawItem struct {
	item CommandItem
	text commandText
}

// resolve picks the cell's command text, and each item's, for targets.
func (c rawCommand) resolve(targets []string) Command {
	cell := c.cell
	var note string
	cell.Command, note = c.text.pick(targets)
	cell.Description = appendNote(cell.Description, note)
	for _, ri := range c.items {
		item := ri.item
		item.Command, note = ri.text.pick(targets)
		item.Description = appendNote(item.Description, note)
		cell.Items = append(cell.Items, item)
	}
	return cell
}

// UnmarshalTOML decodes a command table manually so the command field can
// be polymorphic. Keep the field list in sync with the Command struct.
func (c *rawCommand) UnmarshalTOML(v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("command entry must be a table")
	}
	c.cell = Command{
		Name:               tomlStr(m, "name"),
		Description:        tomlStr(m, "description"),
		Row:                tomlInt(m, "row"),
		Col:                tomlStr(m, "col"),
		AutoCloseExecution: tomlBoolPtr(m, "auto_close_execution"),
		DebugExecution:     tomlBoolPtr(m, "debug_execution"),
		Danger:             tomlDanger(m),
	}

	text, err := parseCommandText(m["command"])
	if err != nil {
		return fmt.Errorf("command %q: %w", c.cell.Name, err)
	}
	c.text = text

	// Inline arrays arrive as []any, [[commands.items]] headers as []map[string]any.
	var rawItems []any
	switch t := m["items"].(type) {
	case []any:
		rawItems = t
	case []map[string]any:
		for _, im := range t {
			rawItems = append(rawItems, im)
		}
	}
	c.items = nil
	for _, ri := range rawItems {
		im, ok := ri.(map[string]any)
		if !ok {
			return fmt.Errorf("command %q: items must be tables", c.cell.Name)
		}
		item := CommandItem{
			Name:               tomlStr(im, "name"),
			Description:        tomlStr(im, "description"),
			AutoCloseExecution: tomlBoolPtr(im, "auto_close_execution"),
			DebugExecution:     tomlBoolPtr(im, "debug_execution"),
			Danger:             tomlDanger(im),
		}
		itext, err := parseCommandText(im["command"])
		if err != nil {
			return fmt.Errorf("command %q, item %q: %w", c.cell.Name, item.Name, err)
		}
		c.items = append(c.items, rawItem{item: item, text: itext})
	}
	return nil
}
