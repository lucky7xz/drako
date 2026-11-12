package tui

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucky7xz/drako/internal/settings"
)

// Matches checks if the key message matches a specific action binding.
func Matches(c settings.InputConfig, msg tea.KeyMsg, binding string) bool {
	return msg.String() == binding
}

// IsQuit checks for global quit keys (ctrl+c, q).
func IsQuit(c settings.InputConfig, msg tea.KeyMsg) bool {
	k := msg.String()
	return k == "ctrl+c" || k == "q"
}

// IsCancel checks for cancel keys (q, esc).
// Often used to exit sub-modes like dropdowns or inventory.
func IsCancel(c settings.InputConfig, msg tea.KeyMsg) bool {
	k := msg.String()
	return k == "q" || k == "esc"
}

// IsConfirm checks for confirmation keys (enter, space).
func IsConfirm(c settings.InputConfig, msg tea.KeyMsg) bool {
	k := msg.String()
	return k == "enter" || k == " "
}

// IsUp checks if the key matches any "up" navigation key.
func IsUp(c settings.InputConfig, msg tea.KeyMsg) bool {
	return slices.Contains(c.NavUp, msg.String())
}

// IsDown checks if the key matches any "down" navigation key.
func IsDown(c settings.InputConfig, msg tea.KeyMsg) bool {
	return slices.Contains(c.NavDown, msg.String())
}

// IsLeft checks if the key matches any "left" navigation key.
func IsLeft(c settings.InputConfig, msg tea.KeyMsg) bool {
	return slices.Contains(c.NavLeft, msg.String())
}

// IsRight checks if the key matches any "right" navigation key.
func IsRight(c settings.InputConfig, msg tea.KeyMsg) bool {
	return slices.Contains(c.NavRight, msg.String())
}

// IsExplain checks if the key matches the explain action.
func IsExplain(c settings.InputConfig, msg tea.KeyMsg) bool {
	return msg.String() == c.Explain
}

// IsInventory checks if the key matches the inventory action.
func IsInventory(c settings.InputConfig, msg tea.KeyMsg) bool {
	return msg.String() == c.Inventory
}

// IsPathGridMode checks if the key matches the path/grid toggle action.
func IsPathGridMode(c settings.InputConfig, msg tea.KeyMsg) bool {
	return msg.String() == c.PathGridMode
}

// IsPathSearch checks if the key matches the directory-filter action.
func IsPathSearch(c settings.InputConfig, msg tea.KeyMsg) bool {
	return msg.String() == c.PathSearch
}

// IsToggleHidden checks if the key matches the hidden-entry toggle.
func IsToggleHidden(c settings.InputConfig, msg tea.KeyMsg) bool {
	return msg.String() == c.ToggleHidden
}

// IsLock checks if the key matches the lock action.
func IsLock(c settings.InputConfig, msg tea.KeyMsg) bool {
	return msg.String() == c.Lock
}

// IsProfilePrev checks if the key matches the previous profile action.
func IsProfilePrev(c settings.InputConfig, msg tea.KeyMsg) bool {
	return msg.String() == c.ProfilePrev
}

// IsProfileNext checks if the key matches the next profile action.
func IsProfileNext(c settings.InputConfig, msg tea.KeyMsg) bool {
	return msg.String() == c.ProfileNext
}

// IsProfileSwitch checks if the key is a profile switch command (Modifier + 1-9).
// Returns true and the 0-based index if matched.
func IsProfileSwitch(c settings.InputConfig, msg tea.KeyMsg, modifier string) (bool, int) {
	digit, ok := strings.CutPrefix(msg.String(), modifier+"+")
	if ok && len(digit) == 1 && digit[0] >= '1' && digit[0] <= '9' {
		return true, int(digit[0] - '1')
	}
	return false, -1
}

// IsSessionLock checks if the key is the manual session-lock chord
// (Modifier + the configured letter).
func IsSessionLock(c settings.InputConfig, msg tea.KeyMsg, modifier string) bool {
	return msg.String() == modifier+"+"+c.SessionLock
}

// IsHelp checks if the key opens the key sheet.
func IsHelp(c settings.InputConfig, msg tea.KeyMsg) bool {
	return msg.String() == c.Help
}

// IsEditFile checks if the key matches the edit-file action (inventory mode).
func IsEditFile(c settings.InputConfig, msg tea.KeyMsg) bool {
	return msg.String() == c.EditFile
}

// IsDelete checks if the key matches the trash-profile action (inventory mode).
func IsDelete(c settings.InputConfig, msg tea.KeyMsg) bool {
	return msg.String() == c.Delete
}
