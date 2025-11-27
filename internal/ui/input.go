package ui

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/config"
)

// Matches checks if the key message matches a specific action binding.
func Matches(c config.InputConfig, msg tea.KeyMsg, binding string) bool {
	return msg.String() == binding
}

// IsUp checks if the key matches any "up" navigation key.
func IsUp(c config.InputConfig, msg tea.KeyMsg) bool {
	return slices.Contains(c.NavUp, msg.String())
}

// IsDown checks if the key matches any "down" navigation key.
func IsDown(c config.InputConfig, msg tea.KeyMsg) bool {
	return slices.Contains(c.NavDown, msg.String())
}

// IsLeft checks if the key matches any "left" navigation key.
func IsLeft(c config.InputConfig, msg tea.KeyMsg) bool {
	return slices.Contains(c.NavLeft, msg.String())
}

// IsRight checks if the key matches any "right" navigation key.
func IsRight(c config.InputConfig, msg tea.KeyMsg) bool {
	return slices.Contains(c.NavRight, msg.String())
}

// IsExplain checks if the key matches the explain action.
func IsExplain(c config.InputConfig, msg tea.KeyMsg) bool {
	return msg.String() == c.Explain
}

// IsInventory checks if the key matches the inventory action.
func IsInventory(c config.InputConfig, msg tea.KeyMsg) bool {
	return msg.String() == c.Inventory
}

// IsPathGridMode checks if the key matches the path/grid toggle action.
func IsPathGridMode(c config.InputConfig, msg tea.KeyMsg) bool {
	return msg.String() == c.PathGridMode
}

// IsLock checks if the key matches the lock action.
func IsLock(c config.InputConfig, msg tea.KeyMsg) bool {
	return msg.String() == c.Lock
}

// IsProfilePrev checks if the key matches the previous profile action.
func IsProfilePrev(c config.InputConfig, msg tea.KeyMsg) bool {
	return msg.String() == c.ProfilePrev
}

// IsProfileNext checks if the key matches the next profile action.
func IsProfileNext(c config.InputConfig, msg tea.KeyMsg) bool {
	return msg.String() == c.ProfileNext
}

// IsProfileSwitch checks if the key is a profile switch command (Modifier + 1-9).
// Returns true and the 0-based index if matched.
func IsProfileSwitch(c config.InputConfig, msg tea.KeyMsg, modifier string) (bool, int) {
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

