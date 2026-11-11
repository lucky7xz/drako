package config

import "github.com/lucky7xz/drako/internal/profile"

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
	Theme              string            `toml:"theme"`
	HeaderArt          *string           `toml:"header_art"`
	DefaultShell       string            `toml:"default_shell"`
	NumbModifier       string            `toml:"numb_modifier"`
	X                  int               `toml:"x"`
	Y                  int               `toml:"y"`
	Profile            string            `toml:"profile"`
	LockTimeoutMinutes *int              `toml:"lock_timeout_minutes"`
	AutoLockEnabled    *bool             `toml:"auto_lock_enabled"`
	BatchForceTmux     bool              `toml:"batch_force_tmux"`
	Keys               InputConfig       `toml:"keys"`
	Commands           []profile.Command `toml:"commands"`
}

// ProfileInfo holds metadata and content of a profile
type ProfileInfo struct {
	Name    string
	Path    string
	Profile profile.File
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
