package config

// InputConfig defines the user-configurable keybindings and toggles.
type InputConfig struct {
	// Toggles for standard navigation sets
	DisableWasd bool `toml:"disable_wasd_bindings"`
	DisableVim  bool `toml:"disable_vim_bindings"`

	// Configurable single-key actions
	Explain      string `toml:"explain"`
	Inventory    string `toml:"inventory"`
	PathGridMode string `toml:"path_grid_mode"`
	Lock         string `toml:"lock"`
	ProfilePrev  string `toml:"profile_prev"`
	ProfileNext  string `toml:"profile_next"`

	// Internal computed sets for fast lookup
	NavUp    []string
	NavDown  []string
	NavLeft  []string
	NavRight []string
}

// InitControls prepares the input config by populating the internal navigation sets
// based on the disable flags. It should be called after loading the config.
func (c *InputConfig) InitControls() {
	// Always include arrow keys
	c.NavUp = []string{"up"}
	c.NavDown = []string{"down"}
	c.NavLeft = []string{"left"}
	c.NavRight = []string{"right"}

	if !c.DisableWasd {
		c.NavUp = append(c.NavUp, "w")
		c.NavDown = append(c.NavDown, "s")
		c.NavLeft = append(c.NavLeft, "a")
		c.NavRight = append(c.NavRight, "d")
	}

	if !c.DisableVim {
		c.NavUp = append(c.NavUp, "k")
		c.NavDown = append(c.NavDown, "j")
		c.NavLeft = append(c.NavLeft, "h")
		c.NavRight = append(c.NavRight, "l")
	}
}
