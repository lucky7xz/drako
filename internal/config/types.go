package config

import (
	"fmt"
	"sort"
	"strings"
)

// CommandItem represents a single item in a command dropdown
type CommandItem struct {
	Name               string `toml:"name"`
	Command            string `toml:"command"`
	Description        string `toml:"description"`
	AutoCloseExecution *bool  `toml:"auto_close_execution"`
	DebugExecution     *bool  `toml:"debug_execution"`
	// Danger keeps drako run from executing it; only the grid can.
	Danger bool `toml:"danger"`
}

// Command represents a grid command
type Command struct {
	Name               string        `toml:"name"`
	Command            string        `toml:"command"`
	Row                int           `toml:"row"`
	Col                string        `toml:"col"`
	Description        string        `toml:"description"`
	AutoCloseExecution *bool         `toml:"auto_close_execution"`
	DebugExecution     *bool         `toml:"debug_execution"`
	Items              []CommandItem `toml:"items"`
	// Danger keeps drako run from executing it; on a folder, every item too.
	Danger bool `toml:"danger"`
}

// AppSettings represents the global configuration in config.toml
type AppSettings struct {
	DefaultShell       string      `toml:"default_shell"`
	NumbModifier       string      `toml:"numb_modifier"`
	Profile            string      `toml:"profile"`
	LockTimeoutMinutes *int        `toml:"lock_timeout_minutes"`
	AutoLockEnabled    *bool       `toml:"auto_lock_enabled"`
	BatchForceTmux     bool        `toml:"batch_force_tmux"`
	Theme              string      `toml:"theme"` // Global Fallback Theme
	Keys               InputConfig `toml:"keys"`
}

// Config represents the runtime application configuration (Settings + Active Profile)
type Config struct {
	Theme              string      `toml:"theme"`
	HeaderArt          *string     `toml:"header_art"`
	DefaultShell       string      `toml:"default_shell"`
	NumbModifier       string      `toml:"numb_modifier"`
	X                  int         `toml:"x"`
	Y                  int         `toml:"y"`
	Profile            string      `toml:"profile"`
	LockTimeoutMinutes *int        `toml:"lock_timeout_minutes"`
	AutoLockEnabled    *bool       `toml:"auto_lock_enabled"`
	BatchForceTmux     bool        `toml:"batch_force_tmux"`
	Keys               InputConfig `toml:"keys"`
	Commands           []Command   `toml:"commands"`
}

// ProfileFile represents the content of a profile file (e.g. core.profile.toml)
type ProfileFile struct {
	X         int       `toml:"x"`
	Y         int       `toml:"y"`
	Theme     string    `toml:"theme"`
	HeaderArt *string   `toml:"header_art"`
	Shell     *string   `toml:"shell"`
	Assets    *[]string `toml:"assets"`
	Commands  []Command `toml:"commands"`
}

// ProfileInfo holds metadata and content of a profile
type ProfileInfo struct {
	Name    string
	Path    string
	Profile ProfileFile
}

// ProfileParseError holds details about a broken profile file
type ProfileParseError struct {
	Name string
	Path string
	Err  string
}

// ConfigBundle packages the base config, effective config, and profile data
type ConfigBundle struct {
	Base        Config
	Config      Config
	Profiles    []ProfileInfo
	ActiveIndex int
	ConfigDir   string
	LockedName  string
	Broken      []ProfileParseError
	// DroppedProfile names the profile that was requested on this load (via the
	// pivot lock, the session, or config.toml) but no longer exists — deleted or
	// moved to the inventory. The load fell through to the next source, or to
	// Rescue Mode; the UI uses it to explain the otherwise-silent switch.
	DroppedProfile string
	// Rescue means Config is the rescue grid, not Profiles[ActiveIndex].
	Rescue bool
}

// ── Platform-variant commands ────────────────────────────────────────
// A command (on a cell or a dropdown item) is either a plain string or a
// table of platform variants. Decoding keeps it as written; DecodeProfile
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
// fields in sync with ProfileFile.
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
func (r rawProfile) resolve(targets []string) ProfileFile {
	pf := ProfileFile{X: r.X, Y: r.Y, Theme: r.Theme, HeaderArt: r.HeaderArt, Shell: r.Shell, Assets: r.Assets}
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
