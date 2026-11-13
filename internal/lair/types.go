package lair

import (
	"github.com/lucky7xz/drako/internal/profile"
	"github.com/lucky7xz/drako/internal/settings"
)

// ProfileInfo holds metadata and content of a profile
type ProfileInfo struct {
	Name    string
	Path    string
	Profile profile.File
	// Flaws is what is wrong with a deck that still loads (profile.Flaws).
	Flaws []profile.Finding
}

// ProfileParseError holds details about a broken profile file
type ProfileParseError struct {
	Name string
	Path string
	Err  string
}

// ConfigBundle packages the base config, effective config, and profile data
type ConfigBundle struct {
	Base        settings.Config
	Config      settings.Config
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
