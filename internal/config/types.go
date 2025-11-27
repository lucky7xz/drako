package config

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

// Config represents the main application configuration
type Config struct {
	DR4koPath          string      `toml:"dR4ko_path"`
	Theme              string      `toml:"theme"`
	HeaderArt          *string     `toml:"header_art"`
	DefaultShell       string      `toml:"default_shell"`
	NumbModifier       string      `toml:"numb_modifier"`
	X                  int         `toml:"x"`
	Y                  int         `toml:"y"`
	Profile            string      `toml:"profile"`
	LockTimeoutMinutes *int        `toml:"lock_timeout_minutes"`
	Keys               InputConfig `toml:"keys"`
	Commands           []Command   `toml:"commands"`
}

// ProfileOverlay represents the overrides in a profile file
type ProfileOverlay struct {
	DR4koPath          *string    `toml:"dR4ko_path"`
	X                  *int       `toml:"x"`
	Y                  *int       `toml:"y"`
	Theme              *string    `toml:"theme"`
	HeaderArt          *string    `toml:"header_art"`
	DefaultShell       *string    `toml:"default_shell"`
	NumbModifier       *string    `toml:"numb_modifier"`
	LockTimeoutMinutes *int       `toml:"lock_timeout_minutes"`
	Assets             *[]string  `toml:"assets"`
	Commands           *[]Command `toml:"commands"`
}

// ProfileInfo holds metadata and content of a profile
type ProfileInfo struct {
	Name    string
	Path    string
	Overlay ProfileOverlay
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

