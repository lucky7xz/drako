package config

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

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
	navUp    []string
	navDown  []string
	navLeft  []string
	navRight []string
}

// InitControls prepares the input config by populating the internal navigation sets
// based on the disable flags. It should be called after loading the config.
func (c *InputConfig) InitControls() {
	// Always include arrow keys
	c.navUp = []string{"up"}
	c.navDown = []string{"down"}
	c.navLeft = []string{"left"}
	c.navRight = []string{"right"}

	if !c.DisableWasd {
		c.navUp = append(c.navUp, "w")
		c.navDown = append(c.navDown, "s")
		c.navLeft = append(c.navLeft, "a")
		c.navRight = append(c.navRight, "d")
	}

	if !c.DisableVim {
		c.navUp = append(c.navUp, "k")
		c.navDown = append(c.navDown, "j")
		c.navLeft = append(c.navLeft, "h")
		c.navRight = append(c.navRight, "l")
	}
}

// Matches checks if the key message matches a specific action binding.
func (c InputConfig) Matches(msg tea.KeyMsg, binding string) bool {
	return msg.String() == binding
}

// IsUp checks if the key matches any "up" navigation key.
func (c InputConfig) IsUp(msg tea.KeyMsg) bool {
	return slices.Contains(c.navUp, msg.String())
}

// IsDown checks if the key matches any "down" navigation key.
func (c InputConfig) IsDown(msg tea.KeyMsg) bool {
	return slices.Contains(c.navDown, msg.String())
}

// IsLeft checks if the key matches any "left" navigation key.
func (c InputConfig) IsLeft(msg tea.KeyMsg) bool {
	return slices.Contains(c.navLeft, msg.String())
}

// IsRight checks if the key matches any "right" navigation key.
func (c InputConfig) IsRight(msg tea.KeyMsg) bool {
	return slices.Contains(c.navRight, msg.String())
}

// IsExplain checks if the key matches the explain action.
func (c InputConfig) IsExplain(msg tea.KeyMsg) bool {
	return msg.String() == c.Explain
}

// IsInventory checks if the key matches the inventory action.
func (c InputConfig) IsInventory(msg tea.KeyMsg) bool {
	return msg.String() == c.Inventory
}

// IsPathGridMode checks if the key matches the path/grid toggle action.
func (c InputConfig) IsPathGridMode(msg tea.KeyMsg) bool {
	return msg.String() == c.PathGridMode
}

// IsLock checks if the key matches the lock action.
func (c InputConfig) IsLock(msg tea.KeyMsg) bool {
	return msg.String() == c.Lock
}

// IsProfilePrev checks if the key matches the previous profile action.
func (c InputConfig) IsProfilePrev(msg tea.KeyMsg) bool {
	return msg.String() == c.ProfilePrev
}

// IsProfileNext checks if the key matches the next profile action.
func (c InputConfig) IsProfileNext(msg tea.KeyMsg) bool {
	return msg.String() == c.ProfileNext
}

// IsProfileSwitch checks if the key is a profile switch command (Modifier + 1-9).
// Returns true and the 0-based index if matched.
func (c InputConfig) IsProfileSwitch(msg tea.KeyMsg, modifier string) (bool, int) {
	key := msg.String()
	prefix := modifier + "+"
	if strings.HasPrefix(key, prefix) && len(key) > len(prefix) {
		numberChar := key[len(key)-1]
		if numberChar >= '1' && numberChar <= '9' {
			return true, int(numberChar - '1')
		}
	}
	return false, -1
}
