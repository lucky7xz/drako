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
	Row                int           `toml:"row"` // 1-based indexing (1-9), -1 for last row
	Col                string        `toml:"col"`
	Description        string        `toml:"description"`
	AutoCloseExecution *bool         `toml:"auto_close_execution"`
	DebugExecution     *bool         `toml:"debug_execution"`
	Items              []CommandItem `toml:"items"`
}

// AppSettings represents the global configuration in config.toml
type AppSettings struct {
	DefaultShell           string      `toml:"default_shell"`
	NumbModifier           string      `toml:"numb_modifier"`
	Profile                string      `toml:"profile"`
	LockTimeoutMinutes     *int        `toml:"lock_timeout_minutes"`
	AutoLockEnabled        *bool       `toml:"auto_lock_enabled"`
	GridSelectionTimeoutMs int         `toml:"grid_selection_timeout_ms"`
	EnvWhitelist           []string    `toml:"env_whitelist"`
	EnvBlocklist           []string    `toml:"env_blocklist"`
	Theme                  string      `toml:"theme"` // Global Fallback Theme
	Keys                   InputConfig `toml:"keys"`
}

// Config represents the runtime application configuration (Settings + Active Profile)
type Config struct {
	Theme                  string      `toml:"theme"`
	HeaderArt              *string     `toml:"header_art"`
	HeaderCommandEnabled   bool        `toml:"header_command_enabled"`
	HeaderCommand          string      `toml:"header_command"`
	HeaderCommandArgs      []string    `toml:"header_command_args"`
	HeaderCommandTimeout   int         `toml:"header_command_timeout"`
	HeaderFallback         string      `toml:"header_fallback"`
	GridSelectionTimeoutMs int         `toml:"grid_selection_timeout_ms"`
	DefaultShell           string      `toml:"default_shell"`
	NumbModifier           string      `toml:"numb_modifier"`
	X                      int         `toml:"x"`
	Y                      int         `toml:"y"`
	Profile                string      `toml:"profile"`
	LockTimeoutMinutes     *int        `toml:"lock_timeout_minutes"`
	AutoLockEnabled        *bool       `toml:"auto_lock_enabled"`
	EnvWhitelist           []string    `toml:"env_whitelist"`
	EnvBlocklist           []string    `toml:"env_blocklist"`
	Keys                   InputConfig `toml:"keys"`
	Commands               []Command   `toml:"commands"`
}

// ProfileFile represents the content of a profile file (e.g. core.profile.toml)
type ProfileFile struct {
	X         int       `toml:"x"`
	Y         int       `toml:"y"`
	Theme     string    `toml:"theme"`
	Icon      string    `toml:"icon"`
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
	Settings    AppSettings
	Base        Config
	Config      Config
	Profiles    []ProfileInfo
	ActiveIndex int
	ConfigDir   string
	LockedName  string
	Broken      []ProfileParseError
}
