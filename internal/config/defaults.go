package config

import "strings"

// ApplyDefaults fills in any missing fields with default values.
// It ensures the configuration is valid and complete.
func (c *Config) ApplyDefaults() {
	defaults := RescueConfig()

	if strings.TrimSpace(c.NumbModifier) == "" {
		c.NumbModifier = defaults.NumbModifier
	}
	if strings.TrimSpace(c.DefaultShell) == "" {
		c.DefaultShell = defaults.DefaultShell
	}
	if strings.TrimSpace(c.Theme) == "" {
		c.Theme = defaults.Theme
	}

	if c.AutoLockEnabled == nil {
		enabled := true
		c.AutoLockEnabled = &enabled
	}

	// Apply key defaults if missing
	if strings.TrimSpace(c.Keys.Explain) == "" {
		c.Keys.Explain = defaults.Keys.Explain
	}
	if strings.TrimSpace(c.Keys.Inventory) == "" {
		c.Keys.Inventory = defaults.Keys.Inventory
	}
	if strings.TrimSpace(c.Keys.PathGridMode) == "" {
		c.Keys.PathGridMode = defaults.Keys.PathGridMode
	}
	if strings.TrimSpace(c.Keys.Lock) == "" {
		c.Keys.Lock = defaults.Keys.Lock
	}
	if strings.TrimSpace(c.Keys.SessionLock) == "" {
		c.Keys.SessionLock = defaults.Keys.SessionLock
	}
	if strings.TrimSpace(c.Keys.ProfilePrev) == "" {
		c.Keys.ProfilePrev = defaults.Keys.ProfilePrev
	}
	if strings.TrimSpace(c.Keys.ProfileNext) == "" {
		c.Keys.ProfileNext = defaults.Keys.ProfileNext
	}
	if strings.TrimSpace(c.Keys.EditFile) == "" {
		c.Keys.EditFile = defaults.Keys.EditFile
	}
	if strings.TrimSpace(c.Keys.Delete) == "" {
		c.Keys.Delete = defaults.Keys.Delete
	}
	if strings.TrimSpace(c.Keys.Leader) == "" {
		c.Keys.Leader = defaults.Keys.Leader
	}

	// Initialize control sets (WASD, Vim, arrows)
	c.Keys.InitControls()
}
