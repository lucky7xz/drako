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
}

// AppSettings represents the global configuration in config.toml
type AppSettings struct {
	DefaultShell       string      `toml:"default_shell"`
	NumbModifier       string      `toml:"numb_modifier"`
	Profile            string      `toml:"profile"`
	LockTimeoutMinutes *int        `toml:"lock_timeout_minutes"`
	AutoLockEnabled    *bool       `toml:"auto_lock_enabled"`
	EnvWhitelist       []string    `toml:"env_whitelist"`
	EnvBlocklist       []string    `toml:"env_blocklist"`
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
	EnvWhitelist       []string    `toml:"env_whitelist"`
	EnvBlocklist       []string    `toml:"env_blocklist"`
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
}

// ── Platform-variant commands ────────────────────────────────────────
// A command (on a cell or a dropdown item) is either a plain string or a
// table of platform variants, resolved once at decode time:
//
//   command = "echo same everywhere"
//   command = { linux_debian = "apt ...", linux_arch = "pacman ...", macos = "brew ..." }
//
// Keys use the weaver vocabulary (linux_debian, linux_arch, linux_fedora,
// linux_void, linux_generic, macos, windows). On Linux, the detected
// target is tried first, then linux_generic. No match: the command stays
// empty (the cell renders, Enter reports "no command configured") and a
// note naming the available variants is appended to the description.

// runtimeTargetFn resolves the platform key; a var so tests can pin one.
var runtimeTargetFn = detectRuntimeTarget

// resolveCommandField turns a decoded TOML command value into a string.
func resolveCommandField(v any) (cmd string, note string, err error) {
	switch t := v.(type) {
	case nil:
		return "", "", nil
	case string:
		return t, "", nil
	case map[string]any:
		variants := make(map[string]string, len(t))
		keys := make([]string, 0, len(t))
		for k, raw := range t {
			s, ok := raw.(string)
			if !ok {
				return "", "", fmt.Errorf("command variant %q must be a string", k)
			}
			variants[k] = s
			keys = append(keys, k)
		}
		target := runtimeTargetFn()
		if c, ok := variants[target]; ok {
			return c, "", nil
		}
		if c, ok := variants["linux_generic"]; ok && strings.HasPrefix(target, "linux_") {
			return c, "", nil
		}
		sort.Strings(keys)
		return "", fmt.Sprintf("[no command variant for this platform (%s); available: %s]",
			target, strings.Join(keys, ", ")), nil
	default:
		return "", "", fmt.Errorf("command must be a string or a table of platform variants, got %T", v)
	}
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
func tomlBoolPtr(m map[string]any, key string) *bool {
	if b, ok := m[key].(bool); ok {
		return &b
	}
	return nil
}

// UnmarshalTOML decodes a command table manually so the command field can
// be polymorphic. Keep the field list in sync with the Command struct.
func (c *Command) UnmarshalTOML(v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("command entry must be a table")
	}
	c.Name = tomlStr(m, "name")
	c.Description = tomlStr(m, "description")
	c.Row = tomlInt(m, "row")
	c.Col = tomlStr(m, "col")
	c.AutoCloseExecution = tomlBoolPtr(m, "auto_close_execution")
	c.DebugExecution = tomlBoolPtr(m, "debug_execution")

	cmd, note, err := resolveCommandField(m["command"])
	if err != nil {
		return fmt.Errorf("command %q: %w", c.Name, err)
	}
	c.Command = cmd
	c.Description = appendNote(c.Description, note)

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
	for _, ri := range rawItems {
		im, ok := ri.(map[string]any)
		if !ok {
			return fmt.Errorf("command %q: items must be tables", c.Name)
		}
		item := CommandItem{
			Name:               tomlStr(im, "name"),
			Description:        tomlStr(im, "description"),
			AutoCloseExecution: tomlBoolPtr(im, "auto_close_execution"),
			DebugExecution:     tomlBoolPtr(im, "debug_execution"),
		}
		icmd, inote, err := resolveCommandField(im["command"])
		if err != nil {
			return fmt.Errorf("command %q, item %q: %w", c.Name, item.Name, err)
		}
		item.Command = icmd
		item.Description = appendNote(item.Description, inote)
		c.Items = append(c.Items, item)
	}
	return nil
}
